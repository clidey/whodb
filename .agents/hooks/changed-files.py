#!/usr/bin/env python3
"""Print file paths from Claude-style file hooks or Codex apply_patch hooks."""

import json
import re
import sys


def add_path(paths: list[str], value: object) -> None:
    if not isinstance(value, str):
        return

    path = value.strip()
    if not path or "\n" in path or "\0" in path:
        return

    paths.append(path)


def main() -> int:
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return 0

    tool_input = payload.get("tool_input")
    if not isinstance(tool_input, dict):
        return 0

    paths: list[str] = []

    add_path(paths, tool_input.get("file_path"))
    add_path(paths, tool_input.get("path"))

    files = tool_input.get("files")
    if isinstance(files, list):
        for file_path in files:
            add_path(paths, file_path)

    command = tool_input.get("command")
    if isinstance(command, str):
        for pattern in (
            r"^\*\*\* (?:Add|Update|Delete) File: (.+)$",
            r"^\*\*\* Move to: (.+)$",
        ):
            for match in re.finditer(pattern, command, re.MULTILINE):
                add_path(paths, match.group(1))

    seen: set[str] = set()
    for path in paths:
        if path not in seen:
            seen.add(path)
            print(path)

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
