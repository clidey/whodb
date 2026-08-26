"""Hydration tests (mirrors the TS hydrate tests; fixtures pin the contract)."""

import sys
import unittest
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from whodb._hydrate import coerce_value, hydrate_rows  # noqa: E402


class TestCoerceValue(unittest.TestCase):
    def test_kinds(self):
        self.assertEqual(coerce_value("42", "bigint"), 42)
        self.assertEqual(coerce_value("3.14", "numeric"), 3.14)
        self.assertIs(coerce_value("true", "boolean"), True)
        self.assertIs(coerce_value("f", "boolean"), False)
        self.assertIsInstance(coerce_value("2026-01-05T00:00:00Z", "timestamptz"), datetime)
        self.assertEqual(coerce_value('{"a": 1}', "jsonb"), {"a": 1})
        self.assertEqual(coerce_value("hello", "text"), "hello")
        self.assertIsNone(coerce_value(None, "bigint"))
        self.assertEqual(coerce_value("not-a-number", "bigint"), "not-a-number")

    def test_non_string_passthrough(self):
        # IPC transport delivers natively-typed values that need no coercion.
        self.assertEqual(coerce_value(42, "bigint"), 42)
        self.assertEqual(coerce_value({"a": 1}, "jsonb"), {"a": 1})


class TestHydrateRows(unittest.TestCase):
    def test_property_metadata_overrides_column_type(self):
        result = {"columns": ["age"], "rows": [["42"]], "total": 1}
        rows, _ = hydrate_rows(result, {"age": "Integer"})
        self.assertEqual(rows[0]["age"], 42)
        rows_untyped, _ = hydrate_rows(result)
        self.assertEqual(rows_untyped[0]["age"], "42")

    def test_pascal_case_rows_result(self):
        result = {"Columns": [{"Name": "n", "Type": "int4"}], "Rows": [["7"]], "TotalCount": 1}
        rows, total = hydrate_rows(result)
        self.assertEqual(rows[0]["n"], 7)
        self.assertEqual(total, 1)


if __name__ == "__main__":
    unittest.main()
