"""Credential-provider tests (subprocess mocked; no CLI required)."""

import json
import subprocess
import sys
import unittest
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from whodb._auth import CliCredentials, StaticCredentials  # noqa: E402
from whodb._errors import CliCredentialsError  # noqa: E402
from whodb.client import _resolve_credentials  # noqa: E402


class TestPrecedence(unittest.TestCase):
    def test_explicit_api_key_wins_over_env(self):
        provider, using_cli, using_api_key = _resolve_credentials(
            "whodb_sk_ctor", None, None, env={"WHODB_API_KEY": "whodb_sk_env"}
        )
        self.assertEqual(provider.token(), "whodb_sk_ctor")
        self.assertFalse(using_cli)
        self.assertTrue(using_api_key)

    def test_env_wins_over_cli(self):
        provider, using_cli, using_api_key = _resolve_credentials(
            None, None, None, env={"WHODB_API_KEY": "whodb_sk_env"}
        )
        self.assertEqual(provider.token(), "whodb_sk_env")
        self.assertTrue(using_api_key)

    def test_no_credentials_falls_back_to_cli(self):
        provider, using_cli, using_api_key = _resolve_credentials(None, None, None, env={})
        self.assertIsInstance(provider, CliCredentials)
        self.assertTrue(using_cli)
        self.assertFalse(using_api_key)

    def test_token_callback(self):
        provider, _, _ = _resolve_credentials(None, lambda: "tok-123", None, env={})
        self.assertEqual(provider.token(), "tok-123")


class TestCliCredentials(unittest.TestCase):
    def _completed(self, payload: dict, returncode: int = 0):
        return subprocess.CompletedProcess(
            args=[], returncode=returncode, stdout=json.dumps(payload).encode(), stderr=b""
        )

    def test_missing_cli_raises_actionable_error(self):
        with mock.patch("subprocess.run", side_effect=FileNotFoundError):
            with self.assertRaises(CliCredentialsError) as ctx:
                CliCredentials().token()
            self.assertIn("WHODB_API_KEY", str(ctx.exception))

    def test_token_and_defaults(self):
        payload = {
            "access_token": "tok",
            "expires_at": "2099-01-01T00:00:00Z",
            "host": "http://localhost:8080",
            "org_id": "org-1",
            "project_id": "proj-1",
        }
        with mock.patch("subprocess.run", return_value=self._completed(payload)) as run:
            provider = CliCredentials()
            self.assertEqual(provider.token(), "tok")
            self.assertEqual(provider.token(), "tok")  # cached — one exec
            self.assertEqual(run.call_count, 1)
            self.assertEqual(provider.defaults()["project_id"], "proj-1")

    def test_nonzero_exit_surfaces_stderr(self):
        completed = subprocess.CompletedProcess(args=[], returncode=1, stdout=b"", stderr=b"not logged in")
        with mock.patch("subprocess.run", return_value=completed):
            with self.assertRaises(CliCredentialsError) as ctx:
                CliCredentials().token()
            self.assertIn("not logged in", str(ctx.exception))


class TestStaticCredentials(unittest.TestCase):
    def test_static(self):
        provider = StaticCredentials("k")
        self.assertEqual(provider.token(), "k")
        provider.refresh()  # no-op
        self.assertEqual(provider.defaults(), {})


if __name__ == "__main__":
    unittest.main()
