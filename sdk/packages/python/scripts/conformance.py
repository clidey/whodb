#!/usr/bin/env python3
"""Python SDK fixture runner: reads fixtures as JSON lines on stdin, executes
each against the SDK with a mock transport, writes one JSON result line per
fixture to stdout. Protocol shared with the TypeScript runner."""

from __future__ import annotations

import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from whodb import WhoDB  # noqa: E402
from whodb._errors import map_graphql_errors  # noqa: E402


class MockTransport:
    """Replays a fixture's scripted transcript, asserting each request."""

    def __init__(self, transcript: list[dict]):
        self._transcript = list(transcript)
        self._index = 0

    def execute(self, operation: str, document: str, variables: dict[str, Any]) -> dict[str, Any]:
        if self._index >= len(self._transcript):
            raise AssertionError(f"unexpected call #{self._index + 1}: {operation}")
        step = self._transcript[self._index]
        self._index += 1
        expect = step.get("expectRequest") or {}
        expected_operation = expect.get("operation")
        if expected_operation and expected_operation != operation:
            raise AssertionError(f"expected operation {expected_operation}, got {operation}")
        for key, value in (expect.get("variables") or {}).items():
            _assert_matches(variables.get(key), value, f"variables.{key}")
        for key, value in (expect.get("variablesContain") or {}).items():
            _assert_matches(variables.get(key), value, f"variables.{key}")
        for key, value in (expect.get("inputContains") or {}).items():
            _assert_matches((variables.get("input") or {}).get(key), value, f"input.{key}")
        response = step["response"]
        if response.get("errors"):
            raise map_graphql_errors(response["errors"])
        return response["data"]


def _assert_matches(actual: Any, expected: Any, path: str) -> None:
    if json.dumps(actual, sort_keys=True) != json.dumps(expected, sort_keys=True):
        raise AssertionError(
            f"mismatch at {path}: expected {json.dumps(expected)}, got {json.dumps(actual)}"
        )


def _canonical(value: Any) -> Any:
    """Serialize results for comparison; datetimes become '@date:<iso Z>'."""
    if isinstance(value, datetime):
        iso = value.isoformat().replace("+00:00", "Z")
        return f"@date:{iso}"
    if isinstance(value, list):
        return [_canonical(item) for item in value]
    if isinstance(value, dict):
        return {key: _canonical(item) for key, item in value.items()}
    return value


def _snake(name: str) -> str:
    out = []
    for index, char in enumerate(name):
        if char.isupper() and index > 0:
            out.append("_")
        out.append(char.lower())
    return "".join(out)


def run_fixture(fixture: dict) -> dict:
    transport = MockTransport(fixture.get("transcript") or [])
    client = WhoDB(
        api_key="test",
        org="00000000-0000-0000-0000-000000000001",
        project="proj-1",
        transport=transport,
    )
    call = fixture["call"]
    try:
        domain = getattr(client, call["domain"])
        handle = domain(call["handle"])
        method = getattr(handle, _snake(call["method"]))
        args = call.get("args") or []
        # Fixture args are TS-shaped: trailing options objects become kwargs.
        positional = list(args)
        kwargs: dict[str, Any] = {}
        if positional and isinstance(positional[-1], dict) and call["method"] in (
            "list",
            "createMany",
            "followLink",
        ):
            options = positional.pop()
            for key, value in options.items():
                kwargs[_snake(key)] = value
        result: Any = method(*positional, **kwargs)
        if call.get("collect") == "pages":
            result = [page.rows for page in result.pages()]
        elif hasattr(result, "all") and callable(result.all):
            result = result.all()
        if fixture.get("expectError"):
            return {
                "name": fixture["name"],
                "pass": False,
                "reason": f"expected {fixture['expectError']['type']}, got success",
            }
        got = _canonical(result if result is not None else None)
        want = fixture.get("expectResult")
        if json.dumps(got, sort_keys=True) != json.dumps(want, sort_keys=True):
            return {
                "name": fixture["name"],
                "pass": False,
                "reason": f"result mismatch:\n  want {json.dumps(want)}\n  got  {json.dumps(got)}",
            }
        return {"name": fixture["name"], "pass": True}
    except Exception as error:  # noqa: BLE001 — fixture harness
        expected = fixture.get("expectError")
        if expected:
            type_ok = type(error).__name__ == expected["type"]
            message_ok = expected.get("messageContains", "") in str(error)
            if type_ok and message_ok:
                return {"name": fixture["name"], "pass": True}
            return {
                "name": fixture["name"],
                "pass": False,
                "reason": f"expected {expected['type']}({expected.get('messageContains', '')}), "
                f"got {type(error).__name__}: {error}",
            }
        return {"name": fixture["name"], "pass": False, "reason": f"{type(error).__name__}: {error}"}


def main() -> None:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        result = run_fixture(json.loads(line))
        sys.stdout.write(json.dumps(result) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
