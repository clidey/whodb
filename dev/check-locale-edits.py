#!/usr/bin/env python3
"""Name locale files whose non-en_US blocks changed since HEAD.

Agents edit en_US only unless the user asks for other languages; dev/translate
fills in the rest. Run by the agent hook after locale edits; prints a reminder
and exits 0, since a requested translation edit is legitimate.

Usage: check-locale-edits.py FILE.yaml...
"""

import subprocess
import sys
from pathlib import Path


def blocks(text):
    """Map each top-level language key to its lines."""
    found, current = {}, None
    for line in text.splitlines():
        if line and not line[0].isspace() and line.rstrip().endswith(":"):
            current = line.rstrip()[:-1]
            found[current] = []
        elif current:
            found[current].append(line)
    return found


def main(files):
    changed = []
    for name in files:
        path = Path(name)
        if not name.endswith(".yaml") or not path.is_file():
            continue
        head = subprocess.run(["git", "-C", str(path.parent), "show", f"HEAD:./{path.name}"], capture_output=True, text=True)
        if head.returncode != 0:
            continue
        before, after = blocks(head.stdout), blocks(path.read_text())
        languages = sorted(lang for lang in before.keys() | after.keys() if lang != "en_US" and before.get(lang) != after.get(lang))
        if languages:
            changed.append(f"{name} ({', '.join(languages)})")
    if changed:
        print(
            "Non-en_US translations changed in " + "; ".join(changed) + ". Edit en_US only unless the user "
            "asked for other languages; dev/translate fills them in."
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
