#!/usr/bin/env python3
"""
Translation drift detector.

Scans YAML locale files in CE (and EE if present) to find:
- Missing keys: present in en_US but absent from a translation
- Stale keys: English text changed since the translation was last updated
- Orphaned keys: present in a translation but removed from en_US

Outputs drift.json for translate.mjs to consume.

Usage:
    python3 detect.py                    # all languages
    python3 detect.py -l fr_FR,de_DE     # specific languages only
"""
from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[1]
CE_LOCALES = REPO_ROOT / "frontend" / "src" / "locales"
EE_LOCALES = REPO_ROOT / "ee" / "frontend" / "src" / "locales"
CHECKSUMS_FILE = SCRIPT_DIR / "checksums.json"
DRIFT_FILE = SCRIPT_DIR / "drift.json"

LANGUAGES = [
    "af_ZA", "am_ET", "ar_AE", "az_AZ", "bg_BG", "bn_BD", "ca_ES", "cs_CZ",
    "da_DK", "de_DE", "el_GR", "en_GB", "es_ES", "et_EE", "fa_IR", "fi_FI",
    "fr_FR", "gu_IN", "ha_NG", "hi_IN", "hr_HR", "hu_HU", "id_ID", "it_IT",
    "iw_IL", "ja_JP", "ka_GE", "km_KH", "kn_IN", "ko_KR", "lt_LT", "lv_LV",
    "ml_IN", "mr_IN", "ms_MY", "nb_NO", "ne_NP", "nl_NL", "pa_IN", "pl_PL",
    "pt_BR", "ro_RO", "ru_RU", "si_LK", "sk_SK", "sl_SI", "sr_RS", "sv_SE",
    "sw_KE", "ta_IN", "te_IN", "th_TH", "tl_PH", "tr_TR", "uk_UA", "ur_PK",
    "uz_UZ", "vi_VN", "yo_NG", "zh_CN", "zh_TW",
]


def hash_value(value: str) -> str:
    """SHA-256 hash of a string value, truncated to 12 hex chars."""
    return hashlib.sha256(value.encode("utf-8")).hexdigest()[:12]


def find_yaml_files() -> list[tuple[str, Path]]:
    """Find all YAML locale files in CE and EE. Returns (label, path) tuples."""
    files: list[tuple[str, Path]] = []
    for label, locales_dir in [("CE", CE_LOCALES), ("EE", EE_LOCALES)]:
        if not locales_dir.exists():
            continue
        for yaml_file in sorted(locales_dir.rglob("*.yaml")):
            files.append((label, yaml_file))
    return files


def load_checksums() -> dict:
    """Load stored checksums, or empty dict if none exist."""
    if CHECKSUMS_FILE.exists():
        with open(CHECKSUMS_FILE) as f:
            return json.load(f)
    return {}


def save_checksums(checksums: dict) -> None:
    """Write checksums to disk."""
    with open(CHECKSUMS_FILE, "w") as f:
        json.dump(checksums, f, indent=2, sort_keys=True)
        f.write("\n")


def main() -> None:
    parser = argparse.ArgumentParser(description="Detect translation drift")
    parser.add_argument(
        "-l", "--languages",
        help="Comma-separated target languages (default: all)",
    )
    args = parser.parse_args()

    languages = LANGUAGES
    if args.languages:
        languages = [l.strip() for l in args.languages.split(",")]
        invalid = [l for l in languages if l not in LANGUAGES]
        if invalid:
            print(f"Unknown languages: {', '.join(invalid)}", file=sys.stderr)
            print(f"Valid: {', '.join(LANGUAGES)}", file=sys.stderr)
            sys.exit(1)

    yaml_files = find_yaml_files()
    checksums = load_checksums()
    is_first_run = not CHECKSUMS_FILE.exists()
    new_checksums: dict[str, dict[str, str]] = {}

    drift: dict[str, dict] = {}
    total_missing = 0
    total_stale = 0
    total_orphaned = 0
    ce_count = sum(1 for label, _ in yaml_files if label == "CE")
    ee_count = sum(1 for label, _ in yaml_files if label == "EE")

    for _label, yaml_path in yaml_files:
        rel_path = str(yaml_path.relative_to(REPO_ROOT))

        with open(yaml_path) as f:
            data = yaml.safe_load(f) or {}

        en_us = data.get("en_US")
        if not en_us or not isinstance(en_us, dict):
            continue

        # Compute current hashes for all en_US values
        file_checksums: dict[str, str] = {}
        for key, value in en_us.items():
            file_checksums[key] = hash_value(str(value))
        new_checksums[rel_path] = file_checksums

        file_drift: dict[str, dict] = {}

        for lang in languages:
            lang_data = data.get(lang)
            if not isinstance(lang_data, dict):
                lang_data = {}

            missing: dict[str, str] = {}
            stale: dict[str, str] = {}
            orphaned: list[str] = []

            # Find missing and stale keys
            for key, value in en_us.items():
                str_value = str(value)
                if key not in lang_data:
                    missing[key] = str_value
                elif not is_first_run:
                    stored = checksums.get(rel_path, {}).get(key)
                    if stored is not None and stored != file_checksums[key]:
                        stale[key] = str_value

            # Find orphaned keys
            for key in lang_data:
                if key not in en_us:
                    orphaned.append(key)

            if missing or stale or orphaned:
                locale_drift: dict = {}
                if missing:
                    locale_drift["missing"] = missing
                    total_missing += len(missing)
                if stale:
                    locale_drift["stale"] = stale
                    total_stale += len(stale)
                if orphaned:
                    locale_drift["orphaned"] = orphaned
                    total_orphaned += len(orphaned)
                file_drift[lang] = locale_drift

        if file_drift:
            drift[rel_path] = file_drift

    # Write drift.json (includes checksums for translate.mjs to persist later)
    output = {
        "files": drift,
        "checksums": new_checksums,
        "summary": {
            "total_missing": total_missing,
            "total_stale": total_stale,
            "total_orphaned": total_orphaned,
        },
    }
    with open(DRIFT_FILE, "w") as f:
        json.dump(output, f, indent=2, ensure_ascii=False)
        f.write("\n")

    # Print summary
    print(f"Scanned {len(yaml_files)} YAML files ({ce_count} CE, {ee_count} EE)")
    if drift:
        print(f"Drift found in {len(drift)} files:")
        print(f"  {total_missing} missing, {total_stale} stale, {total_orphaned} orphaned")
        for rel_path, file_drift in sorted(drift.items()):
            m = sum(len(fd.get("missing", {})) for fd in file_drift.values())
            s = sum(len(fd.get("stale", {})) for fd in file_drift.values())
            o = sum(len(fd.get("orphaned", [])) for fd in file_drift.values())
            locales = len(file_drift)
            print(f"  {rel_path}: {m} missing, {s} stale, {o} orphaned ({locales} locales)")
    else:
        print("No drift detected — all translations up to date.")

    print(f"\nWrote {DRIFT_FILE.name}")

    # Bootstrap checksums on first run only
    if is_first_run:
        save_checksums(new_checksums)
        print(f"First run: bootstrapped {CHECKSUMS_FILE.name}")


if __name__ == "__main__":
    main()
