#!/usr/bin/env bash
# usage: tools/install-baselines.sh <export_dir> <repo_ui_dir>   (copies screenshots/<page>/<variant>/*.png to <repo_ui_dir>/tests/visual/__baselines__/<variant>/<page>/)
set -euo pipefail
src="$1/screenshots"; dst="$2/tests/visual/__baselines__"
for page in "$src"/*/; do p=$(basename "$page"); for v in "$page"*/; do vv=$(basename "$v"); mkdir -p "$dst/$vv/$p"; cp "$v"*.png "$dst/$vv/$p/"; done; done
echo "installed into $dst"
