#!/usr/bin/env python3
"""Linux immutable-batch drainer. Ambiguous starts require operator reconciliation."""
import fcntl,hashlib,json,os,pathlib,sqlite3,subprocess,sys

def drain(root,cli,allowed):
    root=pathlib.Path(root);root.mkdir(parents=True,exist_ok=True)
    for name in ('ready','claimed','results','dead'): (root/name).mkdir(exist_ok=True)
    with (root/'consumer.lock').open('a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        db=sqlite3.connect(root/'jobs.sqlite')
        db.execute('PRAGMA journal_mode=WAL');db.execute('PRAGMA synchronous=FULL')
        db.execute('CREATE TABLE IF NOT EXISTS jobs(id TEXT PRIMARY KEY,digest TEXT NOT NULL,state TEXT NOT NULL,result TEXT)');db.commit()
        # Retained claimed batches are inspected first on restart.
        paths=list(sorted((root/'claimed').glob('*.jsonl')))
        for ready in sorted((root/'ready').glob('*.jsonl')):
            claim=root/'claimed'/ready.name
            if claim.exists(): raise ValueError('batch name collision')
            os.rename(ready,claim);paths.append(claim)
        failed=False
        for batch in paths:
            results=[]
            for line in batch.read_text().splitlines():
                if not line.strip(): continue
                try:
                    job=json.loads(line)
                    if not isinstance(job,dict) or not isinstance(job.get('id'),str) or not job['id'] or job.get('cmd') not in allowed: raise ValueError('invalid id/cmd')
                    args=job.get('args',[])
                    if not isinstance(args,list) or any(not isinstance(x,str) or '\0' in x for x in args): raise ValueError('invalid args')
                    digest=hashlib.sha256(json.dumps(job,sort_keys=True,separators=(',',':')).encode()).hexdigest()
                    prior=db.execute('SELECT digest,state,result FROM jobs WHERE id=?',(job['id'],)).fetchone()
                    if prior:
                        if prior[0]!=digest: raise ValueError('job id reused with different payload')
                        if prior[1]=='done': results.append(json.loads(prior[2])|{'deduplicated':True});continue
                        raise ValueError('job outcome uncertain or failed: reconcile before replay')
                    db.execute('INSERT INTO jobs VALUES(?,?,?,NULL)',(job['id'],digest,'started'));db.commit()
                    # Never invoke a shell; target must implement its own idempotency if retries are needed.
                    r=subprocess.run([cli,job['cmd'],*args],capture_output=True,text=True,timeout=60)
                    result={'id':job['id'],'ok':r.returncode==0,'status':r.returncode,'stdout':r.stdout,'stderr':r.stderr}
                    db.execute('UPDATE jobs SET state=?,result=? WHERE id=?',('done' if result['ok'] else 'failed',json.dumps(result),job['id']));db.commit()
                except (ValueError,TypeError,OSError,subprocess.SubprocessError): result={'ok':False,'error':'invalid, failed, or uncertain job; inspect locally'}
                results.append(result);failed |= not result['ok']
            destination=root/('dead' if any(not x['ok'] for x in results) else 'results')/(batch.stem+'.results.jsonl')
            temp=destination.with_suffix('.tmp')
            with temp.open('w') as f:
                for result in results: f.write(json.dumps(result)+'\n')
                f.flush();os.fsync(f.fileno())
            os.replace(temp,destination)
            # Keep the immutable input for diagnosis; do not truncate it.
            os.rename(batch,destination.with_suffix('.input.jsonl'))
        db.close();return 1 if failed else 0
if __name__=='__main__':
    if len(sys.argv)!=3: sys.exit('usage: drain.py QUEUE_DIRECTORY CLI')
    sys.exit(drain(sys.argv[1],sys.argv[2],set(os.environ.get('RUNNER_COMMANDS','info,lint').split(','))))
