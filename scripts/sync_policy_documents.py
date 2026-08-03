#!/usr/bin/env python3
"""Download a reviewed official-policy candidate list into a manifest-backed store.

Input JSON shape:
{"files": [{"province": "广东", "url": "https://...", "displayName":
"2026-广东-普通高考-招生工作规定.pdf", "examType": "ordinary", ...}]}
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import sys
import tempfile
import urllib.parse
import urllib.request
from pathlib import Path


ALLOWED_TYPES = {
    ".pdf": "application/pdf",
    ".doc": "application/msword",
    ".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    ".xls": "application/vnd.ms-excel",
    ".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    ".zip": "application/zip",
    ".html": "text/html",
    ".htm": "text/html",
}


def fail(message: str) -> None:
    raise SystemExit(f"[policy-sync] {message}")


def clean_name(value: str, fallback: str) -> str:
    value = re.sub(r"[\\/:*?\"<>|\x00-\x1f]+", "-", value).strip(" .-")
    if not value:
        value = fallback
    if len(value) > 180:
        stem, suffix = os.path.splitext(value)
        value = stem[: 180 - len(suffix)] + suffix
    return value


def official_url(url: str, source_url: str) -> bool:
    target = urllib.parse.urlparse(url)
    source = urllib.parse.urlparse(source_url)
    if target.scheme != "https" or not target.hostname or not source.hostname:
        return False
    return target.hostname == source.hostname or target.hostname.endswith("." + source.hostname)


def response_type(url: str, content_type: str) -> str:
    extension = Path(urllib.parse.urlparse(url).path).suffix.lower()
    if extension in ALLOWED_TYPES:
        return ALLOWED_TYPES[extension]
    return content_type.split(";", 1)[0].strip().lower()


def validate_payload(url: str, content_type: str, payload: bytes) -> str:
    kind = response_type(url, content_type)
    if not payload:
        fail(f"empty response: {url}")
    if kind == "application/pdf" and not payload.startswith(b"%PDF"):
        fail(f"response is not a PDF: {url}")
    if kind in {"application/zip", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"} and not payload.startswith(b"PK"):
        fail(f"response is not an Office/ZIP file: {url}")
    if kind.startswith("text/") and b"<html" not in payload[:4096].lower():
        fail(f"response is not HTML: {url}")
    return kind


def download(url: str) -> tuple[str, bytes]:
    request = urllib.request.Request(url, headers={"User-Agent": "SoulCourse-policy-sync/1.0"})
    with urllib.request.urlopen(request, timeout=45) as response:
        return response.headers.get("Content-Type", "application/octet-stream"), response.read()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--year", type=int, default=2026)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    payload = json.loads(args.input.read_text(encoding="utf-8"))
    candidates = payload.get("files", [])
    if not isinstance(candidates, list):
        fail("input.files must be an array")

    planned: dict[str, list[dict]] = {}
    hashes: set[str] = set()
    hashes_by_province: dict[str, set[str]] = {}
    errors: list[dict[str, str]] = []
    for candidate in candidates:
        province = str(candidate.get("province", "")).strip()
        url = str(candidate.get("url", "")).strip()
        source_url = str(candidate.get("sourceUrl", "")).strip()
        display_name = str(candidate.get("displayName", "")).strip()
        if not province or not url or not source_url or not display_name:
            fail("each file needs province, url, sourceUrl and displayName")
        if not official_url(url, source_url):
            fail(f"file is outside its official source domain: {url}")
        extension = Path(urllib.parse.urlparse(url).path).suffix.lower()
        display_extension = Path(display_name).suffix.lower()
        if extension not in ALLOWED_TYPES and display_extension in ALLOWED_TYPES:
            extension = display_extension
        if extension not in ALLOWED_TYPES:
            fail(f"unsupported file extension: {url}")
        display_name = clean_name(display_name, f"{args.year}-{province}-政策文件{extension}")
        try:
            content_type, body = download(url)
            kind = validate_payload(url, content_type, body)
        except Exception as error:
            errors.append({"province": province, "url": url, "error": str(error)})
            continue
        digest = hashlib.sha256(body).hexdigest()
        province_hashes = hashes_by_province.setdefault(province, set())
        if digest in province_hashes:
            continue
        province_hashes.add(digest)
        hashes.add(digest)
        item = dict(candidate)
        item.update({"displayName": display_name, "originalName": Path(urllib.parse.urlparse(url).path).name, "contentType": kind, "bytes": len(body), "sha256": digest, "year": int(candidate.get("year", args.year))})
        item["storedName"] = display_name
        planned.setdefault(province, []).append({"item": item, "body": body})

    if args.dry_run:
        print(json.dumps({"files": {province: [entry["item"] for entry in files] for province, files in planned.items()}, "errors": errors}, ensure_ascii=False, indent=2))
        return 0

    args.output.mkdir(parents=True, exist_ok=True)
    for province, files in planned.items():
        target = args.output / str(args.year) / province
        target.parent.mkdir(parents=True, exist_ok=True)
        temporary = Path(tempfile.mkdtemp(prefix=f".{province}-", dir=target.parent))
        try:
            existing_manifest = {"files": []}
            if target.exists():
                for old in target.iterdir():
                    if old.name != "manifest.json":
                        shutil.copy2(old, temporary / old.name)
                old_manifest = target / "manifest.json"
                if old_manifest.exists():
                    try:
                        existing_manifest = json.loads(old_manifest.read_text(encoding="utf-8"))
                    except (OSError, json.JSONDecodeError):
                        existing_manifest = {"files": []}
            manifest_by_name = {
                str(item.get("storedName") or Path(str(item.get("path", ""))).name): item
                for item in existing_manifest.get("files", [])
                if isinstance(item, dict)
            }
            for entry in files:
                item = entry["item"]
                (temporary / item["storedName"]).write_bytes(entry["body"])
                manifest_by_name[item["storedName"]] = item
            manifest = {"province": province, "fetchedAt": __import__("datetime").datetime.now(__import__("datetime").timezone.utc).isoformat(), "files": list(manifest_by_name.values()), "errors": [item for item in errors if item["province"] == province]}
            (temporary / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            backup = target.with_name(target.name + ".previous")
            if target.exists():
                if backup.exists():
                    shutil.rmtree(backup)
                target.rename(backup)
            temporary.rename(target)
        finally:
            if temporary.exists():
                shutil.rmtree(temporary)
    print(f"[policy-sync] synchronized {len(hashes)} unique files across {len(planned)} provinces; skipped {len(errors)} candidates")
    return 0


if __name__ == "__main__":
    sys.exit(main())
