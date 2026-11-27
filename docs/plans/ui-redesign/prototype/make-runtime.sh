#!/usr/bin/env bash
# Build a locally openable copy of the prototype: injects the React UMD scripts the Claude Design runtime provides.
# usage: ./make-runtime.sh   ->  ./runtime/<Page>.dc.html (open in Chrome; or pass the folder to tools/capture.py)
set -euo pipefail
cd "$(dirname "$0")"
rm -rf runtime && mkdir runtime
cp *.js runtime/ && cp vendor/*.js runtime/
for f in *.dc.html; do
  sed 's#<script src="./support.js">#<script src="./react.production.min.js"></script><script src="./react-dom.production.min.js"></script><script src="./support.js">#' "$f" > "runtime/$f"
done
echo "runtime/ ready: $(ls runtime/*.dc.html | wc -l) pages"
