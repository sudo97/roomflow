"""${{ values.name }}: ${{ values.description }}"""
from http.server import BaseHTTPRequestHandler, HTTPServer

REQUESTS = 0


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        global REQUESTS
        REQUESTS += 1
        if self.path == "/healthz":
            body = b"ok\n"
        elif self.path == "/metrics":
            body = f"# TYPE service_requests_total counter\nservice_requests_total {REQUESTS}\n".encode()
        else:
            self.send_response(404)
            self.end_headers()
            return
        self.send_response(200)
        self.end_headers()
        self.wfile.write(body)


if __name__ == "__main__":
    HTTPServer(("", 8080), Handler).serve_forever()
