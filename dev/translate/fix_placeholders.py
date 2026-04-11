#!/usr/bin/env python3
"""
Fixes broken placeholders in translation YAML files.

Google Translate sometimes mangles {name} placeholders that were protected as
[[[0]]] during translation. This script finds all broken bracket patterns and
restores them to the correct {name} form by looking up the en_US source.

Usage: uv run python fix_placeholders.py [--dry-run]
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[1]
CE_LOCALES = REPO_ROOT / "frontend" / "src" / "locales"
EE_LOCALES = REPO_ROOT / "ee" / "frontend" / "src" / "locales"

# Matches broken bracket placeholders: [[0]], [[[0]]], ([0]]), etc.
# Google Translate sometimes swaps brackets for parentheses.
BROKEN_PLACEHOLDER = re.compile(r"[\[\(]{1,3}\s*(\d+)\s*[\]\)]{1,3}")


def get_placeholders(text: str) -> list[str]:
    """Extract ordered {name} placeholders from a string."""
    return re.findall(r"\{(\w+)\}", str(text))


def fix_value(value: str, placeholders: list[str]) -> str | None:
    """Replace broken bracket patterns with the correct {name} placeholders.
    Returns None if no changes were made."""

    def replacer(m: re.Match) -> str:
        idx = int(m.group(1))
        if idx < len(placeholders):
            return "{" + placeholders[idx] + "}"
        return m.group(0)  # leave as-is if index out of range

    fixed = BROKEN_PLACEHOLDER.sub(replacer, value)
    return fixed if fixed != value else None


def main() -> None:
    dry_run = "--dry-run" in sys.argv

    total_fixed = 0
    files_fixed = 0

    for locales_dir in (CE_LOCALES, EE_LOCALES):
        if not locales_dir.exists():
            continue

        for yaml_path in sorted(locales_dir.rglob("*.yaml")):
            with open(yaml_path) as f:
                data = yaml.safe_load(f) or {}

            en_us = data.get("en_US")
            if not en_us or not isinstance(en_us, dict):
                continue

            # Build placeholder map: key → ordered list of placeholder names
            placeholder_map: dict[str, list[str]] = {}
            for key, value in en_us.items():
                phs = get_placeholders(value)
                if phs:
                    placeholder_map[key] = phs

            if not placeholder_map:
                continue

            # Scan all non-en_US sections for broken placeholders
            file_fixes = 0
            for locale, locale_data in data.items():
                if locale == "en_US" or not isinstance(locale_data, dict):
                    continue
                for key, value in locale_data.items():
                    if key not in placeholder_map:
                        continue
                    str_value = str(value)
                    if not BROKEN_PLACEHOLDER.search(str_value):
                        continue
                    fixed = fix_value(str_value, placeholder_map[key])
                    if fixed is not None:
                        locale_data[key] = fixed
                        file_fixes += 1

            if file_fixes == 0:
                continue

            rel_path = yaml_path.relative_to(REPO_ROOT)
            print(f"  {rel_path}: {file_fixes} placeholders fixed")
            total_fixed += file_fixes
            files_fixed += 1

            if not dry_run:
                # Read original to preserve en_US formatting, rebuild rest
                content = yaml_path.read_text()

                # Extract en_US section from original
                lines = content.split("\n")
                en_start = next(i for i, l in enumerate(lines) if l.startswith("en_US:"))
                en_end = en_start + 1
                while en_end < len(lines) and not re.match(r"^[a-zA-Z_]+:", lines[en_end]):
                    en_end += 1
                en_us_section = "\n".join(lines[en_start:en_end]).rstrip()

                # Get original section order
                original_order: list[str] = []
                for m in re.finditer(r"^([a-zA-Z_]+):", content, re.MULTILINE):
                    name = m.group(1)
                    if name != "en_US" and name not in original_order:
                        original_order.append(name)

                all_locales = [k for k in data if k != "en_US"]
                ordered = [
                    *[l for l in original_order if l in all_locales],
                    *[l for l in all_locales if l not in original_order],
                ]

                sections = [en_us_section]
                for locale in ordered:
                    ld = data[locale]
                    if not isinstance(ld, dict) or not ld:
                        continue
                    block_lines = [f"{locale}:"]
                    for k, v in ld.items():
                        escaped = str(v).replace("\\", "\\\\").replace('"', '\\"')
                        block_lines.append(f'  {k}: "{escaped}"')
                    sections.append("\n".join(block_lines))

                yaml_path.write_text("\n\n".join(sections) + "\n")

    if dry_run:
        print(f"\n[DRY RUN] Would fix {total_fixed} placeholders in {files_fixed} files.")
    else:
        print(f"\nFixed {total_fixed} placeholders in {files_fixed} files.")


if __name__ == "__main__":
    main()
