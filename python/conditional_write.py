"""SQLite compare-and-swap for one reviewed document version.

The ETag is persisted with the representation. Conditional mutation is one SQL
UPDATE, so separate connections/processes cannot both replace the same version.
Authorization, approvals, remote side effects and idempotency are separate.
"""
from dataclasses import dataclass
import re
import sqlite3
import uuid


class PreconditionRequired(ValueError):
    pass


class InvalidETag(ValueError):
    pass


class PreconditionFailed(ValueError):
    pass


@dataclass(frozen=True)
class Document:
    value: str
    etag: str
    revision: int


def new_etag():
    return '"' + uuid.uuid4().hex + '"'


class Store:
    def __init__(self, path):
        self.path = str(path)
        db = self.connect()
        try:
            with db:
                db.execute("PRAGMA journal_mode=WAL")
                db.execute("""CREATE TABLE IF NOT EXISTS document (
                    id INTEGER PRIMARY KEY CHECK (id = 1),
                    value TEXT NOT NULL, etag TEXT NOT NULL, revision INTEGER NOT NULL
                )""")
                db.execute("INSERT OR IGNORE INTO document VALUES (1, ?, ?, 1)",
                           ("initial", new_etag()))
        finally:
            db.close()

    def connect(self):
        db = sqlite3.connect(self.path, timeout=5)
        db.execute("PRAGMA synchronous=FULL")
        return db

    def read(self):
        db = self.connect()
        try:
            return Document(*db.execute(
                "SELECT value, etag, revision FROM document WHERE id=1"
            ).fetchone())
        finally:
            db.close()

    def replace(self, expected_etag, value):
        if expected_etag is None:
            raise PreconditionRequired("If-Match required")
        # This endpoint accepts one persisted strong tag, or a weak tag that
        # cannot match. Wildcard/list syntax is outside its version-binding API.
        if not isinstance(expected_etag, str) or not re.fullmatch(
                r'(?:W/)?"[0-9a-f]{32}"', expected_etag):
            raise InvalidETag("one quoted version ETag required")
        if not isinstance(value, str):
            raise ValueError("value must be a string")
        tag = new_etag()
        db = self.connect()
        try:
            with db:
                row = db.execute("""UPDATE document
                    SET value=?, etag=?, revision=revision+1
                    WHERE id=1 AND etag=?
                    RETURNING value, etag, revision""",
                    (value, tag, expected_etag)).fetchone()
                if row is None:
                    raise PreconditionFailed("target changed")
            return Document(*row)
        finally:
            db.close()
