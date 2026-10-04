import contextlib,http.server,json,os,pathlib,socket,subprocess,sys,tempfile,threading,unittest
ROOT=pathlib.Path(__file__).resolve().parents[1];sys.path.insert(0,str(ROOT/'python'))
from embedding_width import probe_embedding_width
class Fixture(http.server.BaseHTTPRequestHandler):
 status='degraded';notices=[];vector=[0.0]*768
 def do_GET(self):self.send_response(200);self.end_headers();self.wfile.write(json.dumps({'status':self.status}).encode())
 def do_POST(self):
  body=self.rfile.read(int(self.headers['Content-Length']))
  self.send_response(200);self.end_headers()
  if self.path=='/api/embed':self.wfile.write(json.dumps({'embeddings':[self.vector]}).encode())
  else:self.notices.append(json.loads(body));self.wfile.write(b'{}')
 def log_message(self,*args):pass
class Services(unittest.TestCase):
 def setUp(self):
  Fixture.notices=[];Fixture.status='degraded';Fixture.vector=[0.0]*768
  self.server=http.server.ThreadingHTTPServer(('127.0.0.1',0),Fixture);self.thread=threading.Thread(target=self.server.serve_forever,daemon=True);self.thread.start();self.url='http://127.0.0.1:'+str(self.server.server_port)
 def tearDown(self):self.server.shutdown();self.server.server_close();self.thread.join()
 def test_recovery_during_cooldown_and_unknown_unhealthy(self):
  with tempfile.TemporaryDirectory() as d:
   env=os.environ|{'HEALTH_STATE_DIR':d,'HEALTH_CHECK_URL':self.url,'TCP_CHECK_HOST':'127.0.0.1','TCP_CHECK_PORT':str(self.server.server_port),'ALERT_WEBHOOK_URL':self.url+'/alerts'}
   def run():return subprocess.run(['bash',str(ROOT/'shell/health-check-cron.sh')],env=env,capture_output=True,text=True)
   self.assertEqual(run().returncode,1);self.assertEqual(len(Fixture.notices),1)
   self.assertEqual(run().returncode,1);self.assertEqual(len(Fixture.notices),1)
   Fixture.status='ok';self.assertEqual(run().returncode,0);self.assertEqual(len(Fixture.notices),2);self.assertIn('recovered',Fixture.notices[-1]['text'])
   Fixture.status='unknown';self.assertEqual(run().returncode,1);self.assertEqual(len(Fixture.notices),3)
 def test_embedding_shape(self):
  self.assertEqual(probe_embedding_width(self.url,'fixture'),768)
  Fixture.vector=[]
  with self.assertRaises(ValueError):probe_embedding_width(self.url,'fixture')
if __name__=='__main__':unittest.main()
