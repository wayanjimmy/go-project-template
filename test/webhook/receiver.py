"""Local-only webhook receiver with controllable response sequences for Tilt verification."""
import json
import os
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

lock = threading.Lock()
statuses = [204]
requests = []
response_delay = 0


class Handler(BaseHTTPRequestHandler):
    def send_json(self, value):
        body = json.dumps(value).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(body)

    def do_PUT(self):
        global statuses, requests, response_delay
        if self.path != "/control":
            self.send_error(404)
            return
        data = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))))
        with lock:
            statuses = data["statuses"] or [204]
            requests = []
            response_delay = data.get("response_delay", 0)
        self.send_json({"statuses": statuses})

    def do_GET(self):
        if self.path != "/requests":
            self.send_error(404)
            return
        with lock:
            self.send_json(requests)

    def do_POST(self):
        if self.path != "/hooks":
            self.send_error(404)
            return
        body = self.rfile.read(int(self.headers.get("Content-Length", 0))).decode()
        with lock:
            status = statuses[min(len(requests), len(statuses) - 1)]
            record = {"body": body, "delivery_id": self.headers.get("X-Webhook-ID"),
                      "timestamp": self.headers.get("X-Webhook-Timestamp"),
                      "signature": self.headers.get("X-Webhook-Signature"), "status": status}
            requests.append(record)
            delay = response_delay
        print(json.dumps({"msg": "webhook.receiver", **record}), flush=True)
        time.sleep(delay)
        self.send_response(status)
        if status == 302:
            self.send_header("Location", "/must-not-follow")
        self.end_headers()


ThreadingHTTPServer((os.environ.get("WEBHOOK_RECEIVER_HOST", "127.0.0.1"), int(os.environ.get("WEBHOOK_RECEIVER_PORT", "8092"))), Handler).serve_forever()
