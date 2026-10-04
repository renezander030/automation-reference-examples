#!/usr/bin/env bash
# Linux/Unix, Python 3. Health checks always run; cooldown only suppresses repeated alerts.
# Explicit ALERT_WEBHOOK_URL required. State uses a private per-user directory.
exec python3 - <<'PY'
import fcntl,json,os,pathlib,socket,tempfile,time,urllib.request
root=pathlib.Path(os.environ.get('HEALTH_STATE_DIR',str(pathlib.Path.home()/'.local/state/health-check')))
root.mkdir(parents=True,exist_ok=True,mode=0o700)
lock=(root/'lock').open('a');fcntl.flock(lock,fcntl.LOCK_EX)
state_file=root/'state.json'
try: state=json.loads(state_file.read_text())
except FileNotFoundError: state={'observed':'ok','notified':'ok','last_alert':0}
problems=[]
try:
    with urllib.request.urlopen(os.environ.get('HEALTH_CHECK_URL','http://localhost:3000/api/health'),timeout=10) as response:
        body=response.read(65537)
    if len(body)>65536 or json.loads(body).get('status')!='ok': problems.append('HTTP endpoint unhealthy or unknown')
except (OSError,ValueError,AttributeError): problems.append('HTTP endpoint unreadable or unreachable')
try:
    host=os.environ.get('TCP_CHECK_HOST','localhost');port=int(os.environ.get('TCP_CHECK_PORT','5432'))
    with socket.create_connection((host,port),timeout=5): pass
except (OSError,ValueError): problems.append('TCP port unavailable')
current='degraded' if problems else 'ok';now=time.time()
# Compare with last SUCCESSFUL notification, so failed sends are retried.
should_send=(current!=state['notified']) or (current=='degraded' and now-state['last_alert']>=900)
state['observed']=current
if should_send:
    text='Health recovered. All checks passing.' if current=='ok' else 'Health check FAILED\n'+'\n'.join('- '+x for x in problems)
    try:
        request=urllib.request.Request(os.environ['ALERT_WEBHOOK_URL'],data=json.dumps({'text':text}).encode(),headers={'Content-Type':'application/json'},method='POST')
        with urllib.request.urlopen(request,timeout=10) as response: response.read(1024)
        state['notified']=current;state['last_alert']=now
    except (OSError,ValueError,KeyError):
        print('alert delivery failed; will retry',file=__import__('sys').stderr)
        should_send='failed'
fd,temp=tempfile.mkstemp(dir=root)
with os.fdopen(fd,'w') as f: json.dump(state,f);f.flush();os.fsync(f.fileno())
os.replace(temp,state_file)
raise SystemExit(1 if problems or should_send=='failed' else 0)
PY
