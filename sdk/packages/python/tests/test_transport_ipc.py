"""IPC transport dispatch tests against a fake functions-runtime server."""

import json
import sys
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from whodb._errors import NotFoundError, TransportCapabilityError  # noqa: E402
from whodb._transport_ipc import IpcTransport  # noqa: E402

_RESPONSES = {
    "/entities": [{"id": "ent-1", "apiName": "User", "primaryKey": "id"}],
    "/query": {"columns": ["id"], "rows": [["u1"]], "total": 1},
    "/create": {},
    "/update": {},
    "/delete": {},
    "/create_many": ["id-1", "id-2"],
}


class _IpcHandler(BaseHTTPRequestHandler):
    """Fake IPC server recording each call's path and body."""

    calls: list[tuple[str, dict]] = []

    def do_POST(self):  # noqa: N802 — BaseHTTPRequestHandler API
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        body = json.loads(raw or b"{}")
        type(self).calls.append((self.path, body))
        if self.headers.get("Authorization") != "ipc-token" or self.headers.get("X-Job-ID") != "job-1":
            self.send_response(401)
            self.end_headers()
            return
        payload = _RESPONSES.get(self.path)
        status = 200 if payload is not None else 404
        encoded = json.dumps(payload or {}).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def log_message(self, *args):
        pass


class TransportIpcTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), _IpcHandler)
        cls.address = f"127.0.0.1:{cls.server.server_port}"
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        _IpcHandler.calls = []
        self.transport = IpcTransport(address=self.address, job_id="job-1", token="ipc-token")

    def test_query_converts_where_json_and_drops_nones(self):
        data = self.transport.execute("OntologyQuery", "", {
            "input": {"entity": "User", "whereJson": '{"id":{"eq":"u1"}}', "pageSize": 1, "sort": None},
        })
        self.assertIn("OntologyQuery", data)
        path, body = _IpcHandler.calls[0]
        self.assertEqual(path, "/query")
        self.assertEqual(body["where"], {"id": {"eq": "u1"}})
        self.assertNotIn("whereJson", body)
        self.assertNotIn("sort", body)

    def test_add_row_resolves_entity_and_converts_record_inputs(self):
        data = self.transport.execute("OntologyAddRow", "", {
            "entityId": "ent-1",
            "values": [{"Key": "name", "Value": "Ada"}],
        })
        self.assertTrue(data["OntologyAddRow"]["Status"])
        path, body = _IpcHandler.calls[1]  # call 0 is /entities
        self.assertEqual(path, "/create")
        self.assertEqual(body["entity"], "User")
        self.assertEqual(body["data"], {"name": "Ada"})

    def test_update_row_splits_primary_key(self):
        self.transport.execute("OntologyUpdateRow", "", {
            "entityId": "ent-1",
            "values": [{"Key": "id", "Value": "u1"}, {"Key": "plan", "Value": "pro"}],
        })
        path, body = _IpcHandler.calls[1]
        self.assertEqual(path, "/update")
        self.assertEqual(body["pk"], "u1")
        self.assertEqual(body["data"], {"plan": "pro"})

    def test_create_many_returns_inserted_count(self):
        data = self.transport.execute("OntologyAddRows", "", {
            "entityId": "ent-1",
            "rows": [{"values": [{"Key": "name", "Value": "Ada"}]}],
            "idempotencyKey": "batch-1",
        })
        self.assertEqual(data["OntologyAddRows"]["inserted"], 2)
        _, body = _IpcHandler.calls[1]
        self.assertEqual(body["idempotencyKey"], "batch-1")

    def test_unknown_entity_raises_not_found(self):
        with self.assertRaises(NotFoundError):
            self.transport.execute("OntologyAddRow", "", {"entityId": "nope", "values": []})

    def test_unsupported_operation_raises_capability_error(self):
        with self.assertRaises(TransportCapabilityError):
            self.transport.execute("RunSourceQuery", "", {})


if __name__ == "__main__":
    unittest.main()
