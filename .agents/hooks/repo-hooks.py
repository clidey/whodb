#!/usr/bin/env python3
"""Agent hooks driven by .agents/hooks/repo-hooks.json, for Claude Code and Codex.

The same file is copied into every clidey repository; only the JSON differs.
Globs match repository-relative paths, and `*` also matches `/`. A `{files}` in
a command expands to the matching files, shell-quoted. Sections:

  generated      [{"paths": [...], "source": "..."}]  PreToolUse: refuse edits to
                 these files and tell the agent what to edit and run instead.
  after_edit     [{"paths": [...], "run": "..."}]     PostToolUse: formatters and
                 generators for the edited files; a failure is sent to the agent.
  notes          [{"paths": [...], "run": "..."}]     PostToolUse: output is added
                 to the agent's context without blocking.
  before_finish  [{"paths": [...], "run": "..."}]     Stop: checks for files
                 changed in the working tree; a failure keeps the agent working.

Edited files are handled by the configuration of the repository that contains
them, so a session in whodb applies ee's file for ee/ paths. Stop also checks
submodules that have their own configuration. Stop skips a repository whose
changes already passed, and lets the agent stop if the same failure repeats.
Only the config committed at HEAD is used; uncommitted edits to it are ignored.
"""

import fnmatch
import hashlib
import json
import os
import re
import shlex
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
CONFIG = ".agents/hooks/repo-hooks.json"
FILE_HEADER = re.compile(r"^\*\*\* (?:(?:Add|Update|Delete) File|Move to): (.+)$", re.M)


def git(repo, *args):
    result = subprocess.run(["git", "-C", str(repo), *args], capture_output=True, text=True)
    return result.stdout if result.returncode == 0 else ""


def load_config(repo):
    """Return the committed hook config. Uncommitted edits are ignored, so a
    command an agent writes into the config cannot run before a person reviews
    and commits it."""
    try:
        return json.loads(git(repo, "show", f"HEAD:{CONFIG}"))
    except ValueError:
        return None


def owning_repo(path):
    """Return the repo that owns path: this one or a submodule inside it. Files
    in any other repository are not ours to run hooks for."""
    directory = path.parent
    while not directory.is_dir():
        directory = directory.parent
    top = git(directory, "rev-parse", "--show-toplevel").strip()
    if not top:
        return None
    top = Path(top).resolve()
    root = ROOT.resolve()
    return top if top == root or root in top.parents else None


def edited_files(payload):
    """Group the files a tool call edited by owning repository, as relative paths."""
    cwd = Path(payload.get("cwd") or os.getcwd())
    tool_input = payload.get("tool_input") or {}
    if payload.get("tool_name") == "apply_patch":
        raw = FILE_HEADER.findall(tool_input.get("command") or "")
    else:
        raw = [tool_input.get("file_path") or ""]
    groups = {}
    for name in filter(None, (value.strip() for value in raw)):
        path = (cwd / name).resolve()
        repo = owning_repo(path)
        if repo and load_config(repo) is not None:
            groups.setdefault(repo, []).append(path.relative_to(repo).as_posix())
    return groups


def matching(entry, files):
    return [name for name in files if any(fnmatch.fnmatchcase(name, glob) for glob in entry["paths"])]


def run_entries(repo, entries, files):
    """Run each entry whose globs match; return (failures, outputs)."""
    failures, outputs = [], []
    for entry in entries:
        matched = matching(entry, files)
        existing = [name for name in matched if (repo / name).is_file()]
        if not matched or ("{files}" in entry["run"] and not existing):
            continue
        command = entry["run"].replace("{files}", " ".join(shlex.quote(name) for name in existing))
        result = subprocess.run(["bash", "-c", command], cwd=repo, capture_output=True, text=True)
        output = (result.stdout + result.stderr).strip()
        if result.returncode != 0:
            tail = "\n".join(output.splitlines()[-40:])
            failures.append(f"### {repo.name}: `{command}` failed\n{tail}")
        elif output:
            outputs.append(output)
    return failures, outputs


def changed_files(repo):
    status = git(repo, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=all")
    files, entries = [], status.split("\0")
    index = 0
    while index < len(entries):
        entry = entries[index]
        if entry:
            files.append(entry[3:])
            if entry[0] in "RC":
                index += 1
        index += 1
    return files


def fingerprint(repo, files):
    digest = hashlib.sha256(git(repo, "diff", "HEAD", "--no-ext-diff", "--binary").encode())
    for name in sorted(files):
        path = repo / name
        if path.is_file():
            digest.update(f"{name}\0".encode() + path.read_bytes())
    return digest.hexdigest()


def state_file(repo, name):
    git_dir = git(repo, "rev-parse", "--absolute-git-dir").strip()
    return Path(git_dir) / f"repo-hooks-{name}" if git_dir else None


def before_finish(payload):
    if payload.get("stop_hook_active"):
        return 0
    repos = [ROOT]
    submodules = git(ROOT, "config", "--file", ".gitmodules", "--get-regexp", r"^submodule\..*\.path$")
    repos += [ROOT / line.split(" ", 1)[1] for line in submodules.splitlines() if " " in line]
    failures, passed = [], []
    for repo in repos:
        config = load_config(repo)
        if config is None or not config.get("before_finish"):
            continue
        files = changed_files(repo)
        if not files:
            continue
        current = fingerprint(repo, files)
        ok_file = state_file(repo, "ok")
        if ok_file and ok_file.is_file() and ok_file.read_text() == current:
            continue
        repo_failures, _ = run_entries(repo, config["before_finish"], files)
        if repo_failures:
            failures += repo_failures
        elif ok_file:
            passed.append((ok_file, current))
    for ok_file, current in passed:
        ok_file.write_text(current)
    if not failures:
        return 0
    report = "\n\n".join(failures)
    # Let the agent stop when the same failure comes back unchanged, so an error
    # it cannot fix doesn't trap it in a loop.
    error_file = state_file(ROOT, "error")
    error_hash = hashlib.sha256(report.encode()).hexdigest()
    if error_file and error_file.is_file() and error_file.read_text() == error_hash:
        error_file.unlink()
        return 0
    if error_file:
        error_file.write_text(error_hash)
    print("Checks failed for files changed in the working tree; fix them before finishing:\n\n" + report, file=sys.stderr)
    return 2


def main():
    payload = json.load(sys.stdin)
    event = payload.get("hook_event_name")
    if event == "Stop":
        return before_finish(payload)

    groups = edited_files(payload)
    if event == "PreToolUse":
        refusals = []
        for repo, files in groups.items():
            for entry in load_config(repo).get("generated", []):
                refusals += [f"{name} is generated: {entry['source']}" for name in matching(entry, files)]
        if refusals:
            print("\n".join(refusals), file=sys.stderr)
            return 2
        return 0

    failures, notes = [], []
    for repo, files in groups.items():
        config = load_config(repo)
        repo_failures, _ = run_entries(repo, config.get("after_edit", []), files)
        _, repo_notes = run_entries(repo, config.get("notes", []), files)
        failures += repo_failures
        notes += repo_notes
    if failures:
        print("\n\n".join(failures + notes), file=sys.stderr)
        return 2
    if notes:
        print(json.dumps({"hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": "\n\n".join(notes)}}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
