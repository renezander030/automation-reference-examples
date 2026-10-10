import concurrent.futures
from pathlib import Path
import sqlite3
import sys
import tempfile
import threading
import unittest
from urllib.error import HTTPError
from urllib.request import Request, urlopen

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "python"))
from conditional_write import Store, PreconditionFailed, PreconditionRequired, InvalidETag
from conditional_write_demo import make_server, request


def process_write(path, tag, value):
    try:
        Store(path).replace(tag, value)
        return "written"
    except PreconditionFailed:
        return "stale"


class ConditionalWrites(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.store = Store(Path(self.temporary.name) / "document.sqlite")

    def test_stale_refusal_preserves_newer_content(self):
        reviewed = self.store.read()
        newer = self.store.replace(reviewed.etag, "human edit")
        with self.assertRaises(PreconditionFailed):
            self.store.replace(reviewed.etag, "agent edit")
        self.assertEqual(self.store.read(), newer)

    def test_missing_weak_and_malformed_tag_do_not_write(self):
        original = self.store.read()
        for tag, error in [(None, PreconditionRequired), ("W/" + original.etag, PreconditionFailed),
                           ("*", InvalidETag), ('"bad"', InvalidETag),
                           (original.etag + ", " + original.etag, InvalidETag)]:
            with self.assertRaises(error):
                self.store.replace(tag, "changed")
        self.assertEqual(self.store.read(), original)

    def test_etag_does_not_reappear_when_value_reverts(self):
        first = self.store.read()
        second = self.store.replace(first.etag, "new")
        third = self.store.replace(second.etag, first.value)
        self.assertEqual(first.value, third.value)
        self.assertNotEqual(first.etag, third.etag)
        with self.assertRaises(PreconditionFailed):
            self.store.replace(first.etag, "stale")

    def test_reopen_keeps_revision_and_tag(self):
        value = self.store.replace(self.store.read().etag, "persistent")
        self.assertEqual(Store(self.store.path).read(), value)

    def test_two_processes_compete_for_same_version(self):
        tag = self.store.read().etag
        with concurrent.futures.ProcessPoolExecutor(max_workers=2) as pool:
            results = list(pool.map(process_write, [self.store.path] * 2, [tag] * 2, ["A", "B"]))
        self.assertEqual(sorted(results), ["stale", "written"])
        self.assertEqual(self.store.read().revision, 2)

    def test_http_refusals_and_concurrent_writes(self):
        server = make_server(self.store)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        url = f"http://127.0.0.1:{server.server_port}/document"
        try:
            status, tag, body = request(url)
            self.assertEqual(status, 200)
            self.assertEqual(request(url, "PUT", "none")[0], 428)
            self.assertEqual(request(url, "PUT", "weak", "W/" + tag)[0], 412)
            self.assertEqual(request(url, "PUT", "wildcard", "*")[0], 400)
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                results = list(pool.map(lambda v: request(url, "PUT", v, tag)[0], ["A", "B"]))
            self.assertEqual(sorted(results), [200, 412])
            self.assertEqual(request(url, "PUT", "stale", tag)[0], 412)
            before = self.store.read()
            for raw in [b'null', b'{"value":123}', b'{"value":"x","extra":1}', b'{', b'x' * 4097]:
                req = Request(url, raw, {"If-Match": before.etag}, method="PUT")
                with self.assertRaises(HTTPError) as error:
                    urlopen(req, timeout=5)
                self.assertIn(error.exception.code, [400, 413])
                error.exception.close()
            self.assertEqual(self.store.read(), before)
        finally:
            server.shutdown()
            server.server_close()
            thread.join()


if __name__ == "__main__":
    unittest.main()
