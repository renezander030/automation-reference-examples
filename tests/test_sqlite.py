import concurrent.futures,pathlib,sqlite3,tempfile,unittest
class SQLiteContracts(unittest.TestCase):
 def test_fetch_mark_loss_and_atomic_claims(self):
  with tempfile.TemporaryDirectory() as d:
   path=pathlib.Path(d)/'state.sqlite'
   with sqlite3.connect(path) as db:
    db.execute('PRAGMA journal_mode=WAL');db.execute('PRAGMA synchronous=FULL');db.execute('CREATE TABLE seen(id TEXT PRIMARY KEY)');db.execute('INSERT INTO seen VALUES(?)',('fetched-before-downstream-failure',))
   # Reopen after a process-level interruption: the committed marker remains.
   with sqlite3.connect(path) as db:self.assertIsNotNone(db.execute('SELECT id FROM seen').fetchone())
   def claim(i):
    db=sqlite3.connect(path,timeout=10)
    try:
     with db:cursor=db.execute('INSERT OR IGNORE INTO seen VALUES(?)',('same-item',))
     return cursor.rowcount
    finally:db.close()
   with concurrent.futures.ThreadPoolExecutor(max_workers=10) as pool:self.assertEqual(sum(pool.map(claim,range(10))),1)
   # A unique local claim says nothing about whether a remote side effect happened.
if __name__=='__main__':unittest.main()
