#!/usr/bin/env python3
"""Repair downloaded policy HTML files in place.

Official pages commonly keep images and stylesheets as relative URLs. This
utility embeds same-site assets as data URLs so the page remains readable when
rendered from the local policy store. It also fixes common GB18030 pages and
records a repair report without inventing missing content.
"""
from __future__ import annotations

import argparse
import base64
import html
import json
import mimetypes
import re
import shutil
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

UA = "SoulCourse-policy-repair/1.0"
URL_RE = re.compile(r'''(?P<prefix>(?:src|href)\s*=\s*["'])(?P<url>[^"']+)(?P<suffix>["'])''', re.I)
CSS_URL_RE = re.compile(r'''url\(\s*["']?(?P<url>[^"')]+)["']?\s*\)''', re.I)


def fetch(url: str, limit: int = 8 * 1024 * 1024) -> tuple[str, bytes] | None:
    try:
        req = urllib.request.Request(url, headers={"User-Agent": UA})
        with urllib.request.urlopen(req, timeout=5) as response:
            body = response.read(limit + 1)
            if len(body) > limit or response.status < 200 or response.status >= 300:
                return None
            return response.headers.get("Content-Type", "application/octet-stream"), body
    except Exception:
        return None


def data_url(content_type: str, body: bytes) -> str:
    mime = content_type.split(";", 1)[0].strip() or "application/octet-stream"
    if mime == "text/css":
        try:
            css = decode(body)
            css = CSS_URL_RE.sub(lambda m: m.group(0), css)
            body = css.encode("utf-8")
        except Exception:
            pass
    return "data:" + mime + ";base64," + base64.b64encode(body).decode("ascii")


def decode(body: bytes) -> str:
    if body.startswith(b"%PDF-"):
        return ""
    for encoding in ("utf-8", "gb18030", "big5"):
        text = body.decode(encoding, errors="replace")
        if text.count("�") <= 3:
            return text
    return body.decode("utf-8", errors="replace")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", required=True, type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    report: list[dict[str, object]] = []
    for manifest_path in sorted(args.root.glob("*/manifest.json")):
        try:
            manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        except Exception:
            continue
        changed = 0
        for item in manifest.get("files", []):
            if not isinstance(item, dict):
                continue
            stored = str(item.get("storedName", "") or Path(str(item.get("path", ""))).name)
            path = manifest_path.parent / stored
            if path.suffix.lower() not in {".html", ".htm"} or not path.is_file():
                continue
            body = path.read_bytes()
            if body.startswith(b"%PDF-"):
                report.append({"file": str(path), "action": "pdf-disguised-as-html"})
                continue
            source = str(item.get("sourceUrl", "") or "")
            if not source:
                continue
            text = decode(body)
            if not text.strip():
                continue
            base = urllib.parse.urljoin(source, "./")
            assets = 0

            def replace(match: re.Match[str]) -> str:
                nonlocal assets
                raw = html.unescape(match.group("url")).strip()
                if not raw or raw.startswith(("#", "data:", "javascript:", "mailto:", "tel:")):
                    return match.group(0)
                absolute = urllib.parse.urljoin(base, raw)
                if urllib.parse.urlparse(absolute).netloc != urllib.parse.urlparse(base).netloc:
                    return match.group(0)
                fetched = fetch(absolute)
                if not fetched:
                    return match.group(0)
                assets += 1
                return match.group("prefix") + data_url(*fetched) + match.group("suffix")

            repaired = URL_RE.sub(replace, text)
            repaired = re.sub(r"<meta[^>]+charset=[\"']?[^>\"']+", '<meta charset="utf-8"', repaired, flags=re.I)
            if repaired != text:
                if not args.dry_run:
                    backup = path.with_suffix(path.suffix + ".before-repair")
                    if not backup.exists():
                        shutil.copy2(path, backup)
                    path.write_text(repaired, encoding="utf-8")
                changed += 1
                report.append({"file": str(path), "assetsEmbedded": assets, "encoding": "utf-8"})
        if changed and not args.dry_run:
            manifest["repairedAt"] = datetime.now(timezone.utc).isoformat()
            manifest_path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"repaired": report, "count": len(report), "dryRun": args.dry_run}, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
