#!/usr/bin/env python3
#
# Copyright 2026 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Fail when agent instruction files name repository paths that don't exist.

Checks every `backticked` token that looks like a repository file or directory
path in the given markdown files. A path passes if it exists relative to any
--root or to the file that mentions it. Third-party skills (listed in
skills-lock.json, plus --exclude-skill names) are skipped.

Usage: check-agent-doc-paths.py --root DIR [--root DIR ...]
                                [--exclude-skill NAME ...] FILE_OR_DIR ...
"""
import argparse
import json
import os
import re
import sys

TOKEN = re.compile(r"`([^`\s]+)`")
EXTENSIONS = (
    "md mdx go ts tsx js mjs cjs json yaml yml sh py sql graphqls graphql toml "
    "java kts swift kt rs css html tpl zed txt rules"
).split()
SKIP_MARKERS = ("<", ">", "{", "}", "*", "$", "...", "path/to", "://", "example")


def is_repo_path(token):
    if token.startswith(("/", "~", "@", "-", "http", "github.com", "golang.org")):
        return False
    if any(marker in token for marker in SKIP_MARKERS) or "/" not in token.strip("/"):
        return False
    if token.endswith("/"):
        return True
    last = token.rsplit("/", 1)[-1]
    return "." in last and last.rsplit(".", 1)[-1] in EXTENSIONS


def markdown_files(target, excluded_skills):
    if os.path.isfile(target):
        yield target
        return
    for dirpath, dirnames, filenames in os.walk(target):
        dirnames[:] = [d for d in dirnames if not (os.path.basename(dirpath) == "skills" and d in excluded_skills)]
        for name in filenames:
            if name.endswith(".md"):
                yield os.path.join(dirpath, name)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", action="append", required=True)
    parser.add_argument("--exclude-skill", action="append", default=[])
    parser.add_argument("targets", nargs="+")
    args = parser.parse_args()

    roots = [os.path.abspath(r) for r in args.root]
    excluded = set(args.exclude_skill)
    lock = os.path.join(roots[0], "skills-lock.json")
    if os.path.exists(lock):
        excluded.update(json.load(open(lock)).get("skills", {}).keys())

    missing = []
    for target in args.targets:
        for path in markdown_files(target, excluded):
            text = open(path, encoding="utf-8", errors="replace").read()
            for raw in TOKEN.findall(text):
                token = raw.rstrip(".,:;)")
                if not is_repo_path(token):
                    continue
                bases = roots + [os.path.dirname(os.path.abspath(path))]
                if not any(os.path.exists(os.path.join(base, token).rstrip("/")) for base in bases):
                    missing.append(f"{os.path.relpath(path, roots[0])}: {token}")

    for line in sorted(set(missing)):
        print(f"missing path: {line}", file=sys.stderr)
    return 1 if missing else 0


if __name__ == "__main__":
    sys.exit(main())
