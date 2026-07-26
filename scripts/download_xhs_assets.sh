#!/usr/bin/env bash

set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
data_dir="$repo_root/backend/internal/storage/xhs_data"
asset_dir="$repo_root/frontend/public/content/xhs"

for detail in "$data_dir"/*.json; do
  note_id=$(jq -r '.feed_id' "$detail")
  note_dir="$asset_dir/$note_id"
  mkdir -p "$note_dir"

  jq -r '.note.imageList[:9] | to_entries[] | "\(.key + 1)\t\(.value.urlDefault // .value.urlPre // .value.infoList[-1].url)"' "$detail" |
    while IFS=$'\t' read -r index url; do
      target="$note_dir/$index.webp"
      [[ -s "$target" ]] || curl -fsSL --retry 3 "${url/http:/https:}" -o "$target"
    done

  avatar_url=$(jq -r '.note.user.avatar // empty' "$detail")
  if [[ -n "$avatar_url" && ! -s "$note_dir/avatar.jpg" ]]; then
    curl -fsSL --retry 3 "${avatar_url/http:/https:}" -o "$note_dir/avatar.jpg"
  fi
done
