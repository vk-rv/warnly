#!/bin/sh
# Install build tools inside the checkout; no sudo or global shell changes.
set -eu
cd "$(dirname "$0")/.."
case "$(uname -s)" in Linux) os=linux ;; Darwin) os=darwin ;; *) echo 'Supported systems: Linux and macOS' >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64; node_arch=x64 ;; arm64|aarch64) arch=arm64; node_arch=arm64 ;; *) echo 'Supported architectures: amd64 and arm64' >&2; exit 1 ;; esac
command -v curl >/dev/null || { echo 'Install curl first.' >&2; exit 1; }
mkdir -p .tools
export PATH="$PWD/.tools/node/bin:$PWD/.tools/go/bin:$PATH"
go_version=$(awk '/^go / {print $2}' go.mod)
if ! command -v go >/dev/null 2>&1 || ! go version | grep -q "go${go_version}[ -]"; then
  archive="go${go_version}.${os}-${arch}.tar.gz"
  curl --fail --location --retry 3 "https://go.dev/dl/$archive" -o ".tools/$archive"
  tar -xzf ".tools/$archive" -C .tools
fi
if ! command -v node >/dev/null 2>&1 || ! node -e 'const [major,minor]=process.versions.node.split(".").map(Number);process.exit(major>22||(major===22&&minor>=12)?0:1)'; then
  node_version=22.22.0
  archive="node-v${node_version}-${os}-${node_arch}.tar.gz"
  curl --fail --location --retry 3 "https://nodejs.org/dist/v${node_version}/$archive" -o ".tools/$archive"
  curl --fail --location --retry 3 "https://nodejs.org/dist/v${node_version}/SHASUMS256.txt" -o .tools/SHASUMS256.txt
  (cd .tools; grep "  $archive\$" SHASUMS256.txt > node-checksum.txt; if command -v sha256sum >/dev/null; then sha256sum -c node-checksum.txt; else shasum -a 256 -c node-checksum.txt; fi)
  mkdir -p .tools/node
  tar -xzf ".tools/$archive" --strip-components=1 -C .tools/node
fi
go version
node --version
npm --version
