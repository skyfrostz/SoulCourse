#!/usr/bin/env python3
"""Discover ordinary-gaokao article candidates from official province portals.

This only produces a reviewable JSON candidate list. It never writes policy files.
"""

from __future__ import annotations

import argparse
import html
import json
import re
import urllib.parse
import urllib.request
from collections import deque
from html.parser import HTMLParser
from pathlib import Path


KEYWORDS = ("普通高考", "普通高校招生", "招生规定", "志愿", "录取", "投档", "征集", "招生计划", "报名", "体检", "考试安排", "分数线", "成绩")
EXCLUDED = ("成人", "体育", "艺术", "研究生", "硕士", "同等学力", "自学考试", "中考", "中招", "学业水平", "专升本", "五年一贯制")
STAGES = (("报名", "报名"), ("体检", "体检"), ("考试", "考试"), ("计划", "招生计划"), ("志愿", "志愿"), ("投档", "录取"), ("录取", "录取"), ("征集", "征集志愿"), ("分数线", "成绩与分数线"))


class LinkParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.links: list[tuple[str, str]] = []
        self.current_href = ""
        self.current_text: list[str] = []

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        if tag.lower() == "a":
            self.current_href = dict(attrs).get("href") or ""
            self.current_text = []

    def handle_data(self, data: str) -> None:
        if self.current_href:
            self.current_text.append(data)

    def handle_endtag(self, tag: str) -> None:
        if tag.lower() == "a" and self.current_href:
            self.links.append((self.current_href, " ".join("".join(self.current_text).split())))
            self.current_href = ""
            self.current_text = []


def fetch(url: str) -> str:
    request = urllib.request.Request(url, headers={"User-Agent": "SoulCourse-policy-discovery/1.0"})
    with urllib.request.urlopen(request, timeout=25) as response:
        return response.read().decode(response.headers.get_content_charset() or "utf-8", "ignore")


def clean_title(value: str, fallback: str) -> str:
    value = html.unescape(re.sub(r"\s+", " ", value)).strip(" -_")
    return value or fallback


def stage_for(title: str) -> str:
    for keyword, stage in STAGES:
        if keyword in title:
            return stage
    return "其他普通高考政策"


def discover(province: str, source_url: str, max_pages: int) -> list[dict]:
    source = urllib.parse.urlparse(source_url)
    queue = deque([(source_url, 0)])
    seen: set[str] = set()
    candidates: dict[str, dict] = {}
    while queue and len(seen) < max_pages:
        page_url, depth = queue.popleft()
        normalized = page_url.split("#", 1)[0]
        if normalized in seen:
            continue
        seen.add(normalized)
        try:
            content = fetch(normalized)
        except Exception:
            continue
        parser = LinkParser()
        parser.feed(content)
        for href, anchor_title in parser.links:
            candidate_url = urllib.parse.urljoin(normalized, href).split("#", 1)[0]
            parsed = urllib.parse.urlparse(candidate_url)
            if parsed.scheme != "https" or parsed.hostname != source.hostname:
                continue
            title = clean_title(anchor_title, parsed.path.rsplit("/", 1)[-1])
            text = f"{title} {candidate_url}"
            if any(word in text for word in EXCLUDED):
                continue
            if any(segment in parsed.path.lower() for segment in ("/zkzz/", "/hk/", "/yjs", "/crgk/")):
                continue
            if not any(word in text for word in KEYWORDS):
                if depth < 2 and parsed.path.endswith((".html", ".htm")):
                    queue.append((candidate_url, depth + 1))
                continue
            if candidate_url == source_url:
                continue
            year_match = re.search(r"20\d{2}", title)
            if year_match and int(year_match.group()) < 2025:
                continue
            if len(candidates) >= 12:
                continue
            display_name = title
            if not display_name.endswith((".pdf", ".doc", ".docx", ".xls", ".xlsx", ".zip", ".html")):
                display_name += ".html"
            candidates[candidate_url] = {
                "province": province,
                "url": candidate_url,
                "displayName": clean_title(display_name, f"{province}-普通高考政策.html"),
                "examType": "ordinary",
                "stage": stage_for(title),
                "year": int(year_match.group()) if year_match else 2026,
                "sourceTitle": title,
                "sourceUrl": source_url,
                "verificationStatus": "pending",
            }
        if len(candidates) >= 12:
            break
    return list(candidates.values())


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-file", required=True, type=Path, help="knowledgeBase.ts or JSON source registry")
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    text = args.source_file.read_text(encoding="utf-8")
    sources = re.findall(r"province:\s*'([^']+)'[^\n]*?portalUrl:\s*'([^']+)'", text)
    if not sources:
        fail = "no province portals found"
        raise SystemExit(fail)
    files: list[dict] = []
    for province, url in sources:
        if province in {"澳门", "香港", "台湾"}:
            continue
        files.extend(discover(province, url, 80))
    args.output.write_text(json.dumps({"files": files}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"discovered {len(files)} candidates across {len(sources)} portals")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
