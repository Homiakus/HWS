#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import stat
import zipfile
from pathlib import Path

FIXED_TIMESTAMP = (2026, 1, 1, 0, 0, 0)
EXCLUDED_SUFFIXES = (".test.mjs", ".pyc")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build a deterministic GNOME Shell extension archive")
    parser.add_argument("--source", default="shell-extension", type=Path)
    parser.add_argument("--output", default="dist/hws@homiakus.zip", type=Path)
    return parser.parse_args()


def included(path: Path) -> bool:
    if path.name.startswith(".") or "__pycache__" in path.parts:
        return False
    return not path.name.endswith(EXCLUDED_SUFFIXES)


def main() -> int:
    args = parse_args()
    source = args.source.resolve()
    output = args.output.resolve()
    metadata_path = source / "metadata.json"
    if not metadata_path.is_file():
        raise SystemExit(f"metadata.json not found under {source}")

    metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
    uuid = metadata.get("uuid")
    versions = metadata.get("shell-version")
    if uuid != "hws@homiakus":
        raise SystemExit(f"unexpected extension uuid: {uuid!r}")
    if not isinstance(versions, list) or "50" not in versions:
        raise SystemExit(f"GNOME 50 is not declared in shell-version: {versions!r}")

    files = sorted(path for path in source.rglob("*") if path.is_file() and included(path.relative_to(source)))
    if not files:
        raise SystemExit("extension source contains no files")

    output.parent.mkdir(parents=True, exist_ok=True)
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in files:
            relative = path.relative_to(source).as_posix()
            info = zipfile.ZipInfo(relative, FIXED_TIMESTAMP)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = (stat.S_IFREG | 0o644) << 16
            archive.writestr(info, path.read_bytes(), compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)

    with zipfile.ZipFile(output, "r") as archive:
        names = archive.namelist()
        if "metadata.json" not in names:
            raise SystemExit("archive does not contain root metadata.json")
        if any(name.startswith("shell-extension/") for name in names):
            raise SystemExit("archive unexpectedly contains a shell-extension/ prefix")

    print(output)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
