"""Atomic worst-case reservations. All provider billable components must be bounded."""
import sqlite3,contextlib

def initialize(path,token_cap,microusd_cap):
    if token_cap<=0 or microusd_cap<=0: raise ValueError('positive caps required')
    db=sqlite3.connect(path);db.executescript('CREATE TABLE cap(tokens INTEGER,cost INTEGER);CREATE TABLE calls(id TEXT PRIMARY KEY,tokens INTEGER,cost INTEGER,state TEXT);');db.execute('INSERT INTO cap VALUES(?,?)',(token_cap,microusd_cap));db.commit();db.close()

def reserve(path,id,input_bound,max_output,cost_bound):
    if not id or input_bound<0 or max_output<=0 or cost_bound<0: raise ValueError('invalid reservation')
    with contextlib.closing(sqlite3.connect(path,timeout=10)) as db, db:
        db.execute('BEGIN IMMEDIATE')
        cap=db.execute('SELECT tokens,cost FROM cap').fetchone()
        used=db.execute('SELECT COALESCE(SUM(tokens),0),COALESCE(SUM(cost),0) FROM calls').fetchone()
        if used[0]+input_bound+max_output>cap[0] or used[1]+cost_bound>cap[1]: return False
        db.execute('INSERT INTO calls VALUES(?,?,?,?)',(id,input_bound+max_output,cost_bound,'pending'))
    return True

def settle(path,id,tokens,cost):
    if tokens<0 or cost<0: raise ValueError('invalid usage')
    with contextlib.closing(sqlite3.connect(path)) as db, db:
        db.execute('BEGIN IMMEDIATE');row=db.execute('SELECT tokens,cost,state FROM calls WHERE id=?',(id,)).fetchone()
        if row is None or row[2]!='pending': raise ValueError('unknown or already settled call')
        # Record actual usage even if provider broke the claimed bound. Never erase a charge.
        db.execute('UPDATE calls SET tokens=?,cost=?,state=? WHERE id=?',(tokens,cost,'done',id))
    if tokens>row[0] or cost>row[1]: raise ValueError('provider exceeded declared reservation bound')
# Unknown outcomes remain pending and reserve their full bound until reconciled.
# The caller sets the provider's output limit and establishes the input/tool-price bound.
