"""Action-bound authorization fixture. Remote effects require provider CAS/idempotency."""
import hashlib,sqlite3
SCHEMA='''
CREATE TABLE approvals(receipt TEXT PRIMARY KEY,action TEXT NOT NULL,payload_hash TEXT NOT NULL,target TEXT NOT NULL,version TEXT NOT NULL,operator TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE revocations(receipt TEXT PRIMARY KEY REFERENCES approvals(receipt));
CREATE TABLE dispatches(action TEXT PRIMARY KEY,receipt TEXT NOT NULL REFERENCES approvals(receipt),payload_hash TEXT NOT NULL,target TEXT NOT NULL,version TEXT NOT NULL,dispatched_at INTEGER NOT NULL);
CREATE TRIGGER approvals_no_update BEFORE UPDATE ON approvals BEGIN SELECT RAISE(ABORT,'immutable receipt'); END;
CREATE TRIGGER approvals_no_delete BEFORE DELETE ON approvals BEGIN SELECT RAISE(ABORT,'immutable receipt'); END;
'''
def open_store(path):
    db=sqlite3.connect(path);db.execute('PRAGMA foreign_keys=ON');db.execute('PRAGMA journal_mode=WAL');db.execute('PRAGMA synchronous=FULL');db.executescript(SCHEMA);return db

def digest(payload): return hashlib.sha256(payload).hexdigest()
def approve(db,receipt,action,payload,target,version,operator,expires):
    if not all(isinstance(x,str) and x for x in (receipt,action,target,version,operator)) or expires<=0: raise ValueError('invalid approval')
    # Caller authenticates operator outside the model/tool boundary.
    with db: db.execute('INSERT INTO approvals VALUES(?,?,?,?,?,?,?)',(receipt,action,digest(payload),target,version,operator,expires))

def consume(db,receipt,action,payload,target,current_version,now):
    db.execute('BEGIN IMMEDIATE')
    try:
        row=db.execute('SELECT action,payload_hash,target,version,expires FROM approvals WHERE receipt=?',(receipt,)).fetchone()
        if row is None or row[:4]!=(action,digest(payload),target,current_version) or now>=row[4] or db.execute('SELECT 1 FROM revocations WHERE receipt=?',(receipt,)).fetchone(): raise ValueError('approval invalid, expired or revoked')
        db.execute('INSERT INTO dispatches VALUES(?,?,?,?,?,?)',(action,receipt,digest(payload),target,current_version,now));db.commit()
    except BaseException: db.rollback();raise
    # Return an outbox authorization; do not equate its row with successful delivery.
    # Execute with the exact payload, action idempotency key and remote If-Match.
    return action

AUDIT='''SELECT d.action FROM dispatches d LEFT JOIN approvals a
ON d.receipt=a.receipt AND d.action=a.action AND d.payload_hash=a.payload_hash
AND d.target=a.target AND d.version=a.version AND d.dispatched_at<a.expires
WHERE a.receipt IS NULL'''
