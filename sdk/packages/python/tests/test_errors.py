"""Error-mapping and version-check tests (mirrors the TS errors tests)."""

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from whodb._errors import (  # noqa: E402
    AuthError,
    NotFoundError,
    PlatformError,
    ValidationError,
    WhoDBVersionError,
    map_graphql_errors,
)
from whodb._manifest_check import interpret_server_error, reset_warnings, warn_if_flagged  # noqa: E402


class TestErrorMapping(unittest.TestCase):
    def test_code_mapping(self):
        self.assertIsInstance(map_graphql_errors([{"message": "x", "extensions": {"code": "UNAUTHENTICATED"}}]), AuthError)
        self.assertIsInstance(map_graphql_errors([{"message": "x", "extensions": {"code": "FORBIDDEN"}}]), AuthError)
        self.assertIsInstance(map_graphql_errors([{"message": "x", "extensions": {"code": "NOT_FOUND"}}]), NotFoundError)
        self.assertIsInstance(map_graphql_errors([{"message": "x", "extensions": {"code": "BAD_USER_INPUT"}}]), ValidationError)
        platform = map_graphql_errors([{"message": "x", "extensions": {"code": "OTHER"}}])
        self.assertIsInstance(platform, PlatformError)
        self.assertEqual(platform.code, "OTHER")

    def test_empty_errors(self):
        error = map_graphql_errors([])
        self.assertIsInstance(error, PlatformError)

    def test_unknown_operation_becomes_version_error(self):
        converted = interpret_server_error(Exception('Cannot query field "New" on type "Query"'), "1.2.3")
        self.assertIsInstance(converted, WhoDBVersionError)
        self.assertIn("1.2.3", str(converted))

    def test_other_errors_pass_through(self):
        original = Exception("connection refused")
        self.assertIs(interpret_server_error(original, "1.2.3"), original)

    def test_unflagged_operation_no_warning(self):
        reset_warnings()
        import warnings as warnings_module

        with warnings_module.catch_warnings(record=True) as captured:
            warnings_module.simplefilter("always")
            warn_if_flagged("OntologyRows")
        self.assertEqual(len(captured), 0)


if __name__ == "__main__":
    unittest.main()
