#!/usr/bin/env python3
"""
Checks that all non-en_US translations have the same {placeholders} as en_US.
Reports any missing, extra, or mangled placeholders.

Usage: uv run python check_placeholders.py
"""
from __future__ import annotations

import re
from pathlib import Path

import yaml

REPO_ROOT = Path(__file__).resolve().parent.parents[1]
CE_LOCALES = REPO_ROOT / "frontend" / "src" / "locales"
EE_LOCALES = REPO_ROOT / "ee" / "frontend" / "src" / "locales"

PLACEHOLDER_RE = re.compile(r"\{(\w+)\}")


def check_file(yaml_path: Path) -> list[str]:
    issues: list[str] = []
    rel = yaml_path.relative_to(REPO_ROOT)

    with open(yaml_path) as f:
        data = yaml.safe_load(f) or {}

    en_us = data.get("en_US")
    if not en_us or not isinstance(en_us, dict):
        return issues

    for locale, locale_data in data.items():
        if locale == "en_US" or not isinstance(locale_data, dict):
            continue
        for key, value in locale_data.items():
            if key not in en_us:
                continue
            expected = set(PLACEHOLDER_RE.findall(str(en_us[key])))
            if not expected:
                continue
            actual = set(PLACEHOLDER_RE.findall(str(value)))
            missing = expected - actual
            extra = actual - expected
            if missing:
                issues.append(f"  {rel} [{locale}] {key}: missing {missing} — value: {value}")
            if extra:
                issues.append(f"  {rel} [{locale}] {key}: extra {extra} — value: {value}")

    return issues


def main() -> None:
    all_issues: list[str] = []
    for locales_dir in (CE_LOCALES, EE_LOCALES):
        if not locales_dir.exists():
            continue
        for yaml_path in sorted(locales_dir.rglob("*.yaml")):
            all_issues.extend(check_file(yaml_path))

    if all_issues:
        print(f"Found {len(all_issues)} placeholder issues:\n")
        for issue in all_issues:
            print(issue)
    else:
        print("All placeholders correct.")


if __name__ == "__main__":
    main()
