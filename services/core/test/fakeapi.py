#!/usr/bin/env python3
"""Scriptable, recording stand-in for wormix-api's internal routes (used by diff_fake.py).

  POST /__set   {"route": [status, "body text"], ...}   answers for the next calls to these routes
  GET  /__log   the requests recorded since the last /__log (and clears it)
  POST /api/internal/<route>   answers with what was set for <route> (default 404), records the request
"""
import json
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

lock = threading.Lock()
answers = {}
log = []


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *a):
        pass

    def _send(self, status, body):
        data = body.encode() if isinstance(body, str) else body
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_GET(self):
        if self.path == "/__log":
            with lock:
                out = list(log)
                log.clear()
            return self._send(200, json.dumps(out))
        self._send(404, "{}")

    def do_POST(self):
        n = int(self.headers.get("Content-Length") or 0)
        body = self.rfile.read(n)
        if self.path == "/__set":
            with lock:
                answers.update(json.loads(body))
            return self._send(200, "{}")
        route = self.path.split("/api/internal/", 1)[-1]
        headers = {k.lower(): v for k, v in self.headers.items()
                   if k.lower() in ("content-type", "x-tcp-session", "x-session-key", "user-agent",
                                    "accept-encoding", "expect")}
        with lock:
            log.append({"route": route, "headers": headers, "body": body.decode("utf-8", "replace")})
            status, text = answers.get(route, [404, '{"message":"not set"}'])
        self._send(status, text)


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 8000), Handler).serve_forever()
