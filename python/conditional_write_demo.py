"""Bounded localhost HTTP fixture for conditional_write.Store; not a deployment."""
from concurrent.futures import ThreadPoolExecutor
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
from pathlib import Path
import sqlite3
import tempfile
import threading
from urllib.error import HTTPError
from urllib.request import Request, urlopen

from conditional_write import Store, PreconditionRequired, InvalidETag, PreconditionFailed


def make_server(store):
    class Handler(BaseHTTPRequestHandler):
        def log_message(self, *_):
            pass

        def setup(self):
            super().setup()
            self.connection.settimeout(5)

        def reply(self, status, payload, etag=None):
            body = json.dumps(payload, separators=(",", ":"), ensure_ascii=False).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Cache-Control", "no-store")
            if etag is not None:
                self.send_header("ETag", etag)
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            if self.path != "/document":
                return self.reply(404, {"error": "not found"})
            item = store.read()
            self.reply(200, {"value": item.value, "revision": item.revision}, item.etag)

        def do_PUT(self):
            if self.path != "/document":
                return self.reply(404, {"error": "not found"})
            if self.headers.get("If-Match") is None:
                return self.reply(428, {"error": "If-Match required"})
            if self.headers.get("Transfer-Encoding") is not None:
                return self.reply(400, {"error": "Content-Length required"})
            try:
                length = int(self.headers.get("Content-Length", "0"))
                if not 0 < length <= 4096:
                    return self.reply(413, {"error": "body must be 1..4096 bytes"})
                raw = self.rfile.read(length)
                if len(raw) != length:
                    return self.reply(400, {"error": "incomplete body"})
                data = json.loads(raw)
                if not isinstance(data, dict) or set(data) != {"value"}:
                    return self.reply(400, {"error": "expected a value string"})
                item = store.replace(self.headers["If-Match"], data["value"])
            except PreconditionFailed as error:
                return self.reply(412, {"error": str(error)})
            except (ValueError, UnicodeError) as error:
                return self.reply(400, {"error": "invalid request"})
            except (TimeoutError, OSError):
                return self.reply(400, {"error": "body read failed"})
            except sqlite3.Error:
                return self.reply(503, {"error": "storage unavailable; reconcile before retry"})
            self.reply(200, {"value": item.value, "revision": item.revision}, item.etag)

    return ThreadingHTTPServer(("127.0.0.1", 0), Handler)


def request(url, method="GET", value=None, tag=None):
    headers = {"Content-Type": "application/json"}
    if tag is not None:
        headers["If-Match"] = tag
    data = None if value is None else json.dumps({"value": value}).encode()
    req = Request(url, data=data, headers=headers, method=method)
    try:
        response = urlopen(req, timeout=5)
    except HTTPError as error:
        response = error
    with response:
        return response.status, response.headers.get("ETag"), json.load(response)


def main():
    with tempfile.TemporaryDirectory() as temporary:
        store = Store(Path(temporary) / "documents.sqlite")
        server = make_server(store)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        url = f"http://127.0.0.1:{server.server_port}/document"
        try:
            _, reviewed, _ = request(url)
            assert request(url, "PUT", "human edit", reviewed)[0] == 200
            assert request(url, "PUT", "stale agent edit", reviewed)[0] == 412
            assert request(url)[2]["value"] == "human edit"
            print("PASS stale write: 412; human edit preserved")
            assert request(url, "PUT", "no condition")[0] == 428
            print("PASS missing If-Match: 428")
            _, tag, _ = request(url)
            assert request(url, "PUT", "weak", "W/" + tag)[0] == 412
            print("PASS weak ETag: 412")
            with ThreadPoolExecutor(max_workers=2) as pool:
                statuses = sorted(pool.map(
                    lambda v: request(url, "PUT", v, tag)[0], ["writer A", "writer B"]))
            assert statuses == [200, 412], statuses
            print("PASS concurrent writes: [200, 412]")
            before = store.read()
            assert Store(store.path).read() == before
            print("PASS reopened store: value and ETag preserved")
        finally:
            server.shutdown()
            server.server_close()
            thread.join()


if __name__ == "__main__":
    main()
