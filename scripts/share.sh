#!/usr/bin/env bash
# Builds share/carrybee-handbook.zip for people who have neither the app repos nor Go: the content,
# the scan snapshot, a site prebuilt from that snapshot, start scripts and the handbook program for
# macOS (arm64, amd64), Linux (amd64) and Windows (amd64). Needs the app repos present.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

name=carrybee-handbook
share_dir="$root/share"
stage="$share_dir/$name"
zip_path="$share_dir/$name.zip"
targets=(darwin/arm64 darwin/amd64 linux/amd64 windows/amd64)

fail() {
  echo "share: $*" >&2
  exit 1
}

echo "==> Scanning the apps and refreshing snapshot/scan.json"
build_log="$(go run ./cmd/handbook build)"
echo "$build_log"
if grep -qE "NOT SCANNED|from snapshot" <<<"$build_log"; then
  fail "every app repo must be present and scan cleanly (see above)"
fi

echo "==> Running the tests"
go test -count=1 ./...

rm -rf "$stage" "$zip_path"
mkdir -p "$stage"

for target in "${targets[@]}"; do
  os="${target%/*}"
  arch="${target#*/}"
  ext=""
  if [ "$os" = windows ]; then
    ext=".exe"
  fi
  echo "==> Building bin/${os}-${arch}/handbook${ext}"
  CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build -trimpath -ldflags "-s -w" \
    -o "$stage/bin/${os}-${arch}/handbook${ext}" ./cmd/handbook
done

echo "==> Copying content, data, snapshot and start scripts"
cp handbook.json "$stage/"
cp -R content data snapshot "$stage/"
cp scripts/share/README-FIRST.txt scripts/share/start.sh "$stage/"
chmod +x "$stage/start.sh"
awk '{ printf "%s\r\n", $0 }' scripts/share/start.bat >"$stage/start.bat"

# The app paths in handbook.json point at ../hive and friends, which do not exist next to the stage,
# so this build runs from the snapshot exactly as it will for a recipient.
echo "==> Prebuilding dist/ from the snapshot"
snap_log="$(go run ./cmd/handbook build -config "$stage/handbook.json" -out "$stage/dist")"
echo "$snap_log"
if grep -q "NOT SCANNED" <<<"$snap_log" || ! grep -q "from snapshot" <<<"$snap_log"; then
  fail "the shared copy did not build from the snapshot (see above)"
fi

find "$stage" -name .DS_Store -delete
(cd "$share_dir" && zip -r -q "$name.zip" "$name" -x '*.DS_Store')

echo "==> Ready: $zip_path ($(du -h "$zip_path" | awk '{print $1}'))"
