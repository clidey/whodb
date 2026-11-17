#!/usr/bin/env python3
"""Check that literal t('key') calls in the given files have an en_US entry.

A key resolves through `common.yaml`, then `<namespace>.yaml` from each locale
directory in order (later directories override, as extension layers do at
runtime), with plural variants such as `key_one` counting as the key. Calls whose
namespace or key isn't a string literal are skipped.

Usage: check-translation-keys.py [--locales DIR]... FILE...
       (default --locales frontend/src/locales)
"""

import re
import sys
from pathlib import Path

HOOK = re.compile(r"""const\s*\{\s*t(?:\s*:\s*(\w+))?\s*\}\s*=\s*useTranslation\(\s*['"]([^'"]+)['"]\s*\)""")
PLURAL = ("_zero", "_one", "_two", "_few", "_many", "_other")


def en_us_keys(path):
    keys, inside = set(), False
    if not path.is_file():
        return keys
    for line in path.read_text().splitlines():
        if line and not line[0].isspace():
            inside = line.startswith("en_US:")
            continue
        match = re.match(r"^  ([A-Za-z0-9_.-]+):", line)
        if inside and match:
            keys.add(match.group(1))
    return keys


def main(args):
    locales = []
    while args[:1] == ["--locales"]:
        locales.append(Path(args[1]))
        args = args[2:]
    locales = locales or [Path("frontend/src/locales")]
    problems = []
    for name in args:
        if not name.endswith((".ts", ".tsx")) or not Path(name).is_file():
            continue
        # Blank out comments (keeping line numbers) so documentation examples are skipped.
        text = re.sub(r"/\*.*?\*/|//[^\n]*", lambda m: re.sub(r"[^\n]", " ", m.group(0)), Path(name).read_text(), flags=re.S)
        # A file can hold several components, each binding its own `t`; a call may
        # use any namespace bound to that name in the file.
        namespaces = {}
        for match in HOOK.finditer(text):
            namespaces.setdefault(match.group(1) or "t", []).append(match.group(2))
        for variable, bound in namespaces.items():
            files = [root / "common.yaml" for root in locales] + [root / f"{ns}.yaml" for ns in bound for root in locales]
            known = set().union(*(en_us_keys(path) for path in files))
            call = re.compile(r"(?<![\w.$])" + re.escape(variable) + r"""\(\s*(['"])([^'"\n]+)\1""")
            for key_match in call.finditer(text):
                key = key_match.group(2)
                if key not in known and not any(key + suffix in known for suffix in PLURAL):
                    line = text.count("\n", 0, key_match.start()) + 1
                    problems.append(f"{name}:{line}: {variable}('{key}') has no en_US entry in common.yaml or {' / '.join(f'{ns}.yaml' for ns in bound)}")
    if problems:
        print("\n".join(sorted(set(problems))), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
