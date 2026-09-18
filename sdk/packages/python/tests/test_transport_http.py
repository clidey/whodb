"""HTTP transport and workspace-resolution tests against a real local stub."""

import json
import os
import sys
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from whodb import WhoDB  # noqa: E402
from whodb._errors import AuthError, NotFoundError, ValidationError  # noqa: E402
from whodb._transport import HttpTransport  # noqa: E402


class _StubHandler(BaseHTTPRequestHandler):
    """Replays scripted responses in order and records every request."""

    script: list[tuple[int, dict]] = []
    requests: list[dict] = []

    def do_POST(self):  # noqa: N802 — BaseHTTPRequestHandler API
        raw = self.rfile.read(int(self.headers.get("Content-Length", "0")))
        body = json.loads(raw or b"{}")
        query = body.get("query", "")
        operation = query.split()[1].split("(")[0].split("{")[0] if len(query.split()) > 1 else ""
        type(self).requests.append({"headers": dict(self.headers), "operation": operation})
        status, payload = (
            type(self).script.pop(0) if type(self).script else (500, {"unexpected": True})
        )
        encoded = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)

    def log_message(self, *args):  # silence test output
        pass


class _StaticCredentials:
    def __init__(self, value):
        self.value = value
        self.refreshes = 0

    def token(self):
        return self.value

    def refresh(self):
        self.refreshes += 1
        self.value = f"token-{self.refreshes}"


class TransportHttpTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), _StubHandler)
        cls.host = f"http://127.0.0.1:{cls.server.server_port}"
        threading.Thread(target=cls.server.serve_forever, daemon=True).start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        _StubHandler.script = []
        _StubHandler.requests = []
        for name in ("WHODB_API_KEY", "WHODB_ORG", "WHODB_PROJECT", "WHODB_HOST", "WHODB_IPC_TOKEN"):
            os.environ.pop(name, None)

    def test_sends_bearer_user_agent_and_workspace_headers(self):
        _StubHandler.script = [(200, {"data": {"Op": True}})]
        transport = HttpTransport(self.host, _StaticCredentials("tok"))
        transport.set_workspace("org-1", "proj-1")
        transport.execute("Op", "query Op { Op }", {})
        headers = _StubHandler.requests[0]["headers"]
        self.assertEqual(headers["Authorization"], "Bearer tok")
        self.assertTrue(headers["User-Agent"].startswith("clidey-whodb-python/"))
        self.assertEqual(headers["X-Whodb-Org-Id"], "org-1")
        self.assertEqual(headers["X-Whodb-Project-Id"], "proj-1")

    def test_retries_once_on_transient_5xx(self):
        _StubHandler.script = [(503, {}), (200, {"data": {"Op": 42}})]
        transport = HttpTransport(self.host, _StaticCredentials("tok"))
        data = transport.execute("Op", "query Op { Op }", {})
        self.assertEqual(data["Op"], 42)
        self.assertEqual(len(_StubHandler.requests), 2)

    def test_refreshes_credentials_once_on_401(self):
        _StubHandler.script = [(401, {}), (200, {"data": {"Op": True}})]
        credentials = _StaticCredentials("stale")
        transport = HttpTransport(self.host, credentials)
        transport.execute("Op", "query Op { Op }", {})
        self.assertEqual(credentials.refreshes, 1)
        self.assertEqual(_StubHandler.requests[1]["headers"]["Authorization"], "Bearer token-1")

    def test_persistent_401_raises_auth_error(self):
        _StubHandler.script = [(401, {}), (401, {})]
        transport = HttpTransport(self.host, _StaticCredentials("bad"))
        with self.assertRaises(AuthError):
            transport.execute("Op", "query Op { Op }", {})

    def test_graphql_errors_map_over_real_http(self):
        _StubHandler.script = [
            (200, {"errors": [{"message": "nope", "extensions": {"code": "NOT_FOUND"}}]})
        ]
        transport = HttpTransport(self.host, _StaticCredentials("tok"))
        with self.assertRaises(NotFoundError):
            transport.execute("Op", "query Op { Op }", {})

    def test_client_resolves_workspace_slugs(self):
        _StubHandler.script = [
            (200, {"data": {"MyOrganizations": [
                {"id": "11111111-1111-1111-1111-111111111111", "slug": "acme"}]}}),
            (200, {"data": {"Projects": [
                {"id": "22222222-2222-2222-2222-222222222222", "slug": "analytics"}]}}),
            (200, {"data": {"OntologyEntities": []}}),
        ]
        client = WhoDB(token="tok", org="acme", project="analytics", host=self.host)
        client.ontology_entities()
        self.assertEqual(
            [r["operation"] for r in _StubHandler.requests],
            ["MyOrganizations", "Projects", "OntologyEntities"],
        )
        final = _StubHandler.requests[2]["headers"]
        self.assertEqual(final["X-Whodb-Org-Id"], "11111111-1111-1111-1111-111111111111")
        self.assertEqual(final["X-Whodb-Project-Id"], "22222222-2222-2222-2222-222222222222")

    def test_api_key_discovers_workspace_via_my_workspace(self):
        _StubHandler.script = [
            (200, {"data": {"MyWorkspace": {
                "orgId": "33333333-3333-3333-3333-333333333333",
                "projectId": "44444444-4444-4444-4444-444444444444"}}}),
            (200, {"data": {"OntologyEntities": []}}),
        ]
        client = WhoDB(api_key="whodb_sk_test", host=self.host)
        client.ontology_entities()
        self.assertEqual(_StubHandler.requests[0]["operation"], "MyWorkspace")
        self.assertEqual(
            _StubHandler.requests[1]["headers"]["X-Whodb-Project-Id"],
            "44444444-4444-4444-4444-444444444444",
        )

    def test_multi_grant_key_without_project_raises(self):
        _StubHandler.script = [
            (200, {"data": {"MyWorkspace": {
                "orgId": "33333333-3333-3333-3333-333333333333", "projectId": None}}})
        ]
        client = WhoDB(api_key="whodb_sk_test", host=self.host)
        with self.assertRaises(ValidationError):
            client.ontology_entities()


if __name__ == "__main__":
    unittest.main()
