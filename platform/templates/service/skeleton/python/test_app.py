import threading
import unittest
import urllib.request
from http.server import HTTPServer

import app


class AppTest(unittest.TestCase):
    def test_health(self):
        srv = HTTPServer(("127.0.0.1", 0), app.Handler)
        threading.Thread(target=srv.serve_forever, daemon=True).start()
        with urllib.request.urlopen(f"http://127.0.0.1:{srv.server_port}/healthz") as r:
            self.assertEqual(r.status, 200)
        srv.shutdown()


if __name__ == "__main__":
    unittest.main()
