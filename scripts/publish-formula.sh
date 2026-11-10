#!/bin/bash
# Renders the Homebrew formula for a release tag and pushes it to the tap.
# Requires TAG and TAP_GITHUB_TOKEN. Set TAP_DIR to an existing tap checkout
# to skip the clone and the push (local testing)
set -euo pipefail

: "${TAG:?TAG is required, e.g. v0.1.0}"

version="${TAG#v}"
url="https://github.com/MaxSiominDev/v2a/archive/refs/tags/${TAG}.tar.gz"

if [ -n "${TAP_DIR:-}" ]; then
  tap_dir="${TAP_DIR}"
  mkdir -p "${tap_dir}/Formula"
else
  : "${TAP_GITHUB_TOKEN:?TAP_GITHUB_TOKEN is required}"
  tap_dir="$(mktemp -d)/homebrew-v2a"
  git clone -q "https://x-access-token:${TAP_GITHUB_TOKEN}@github.com/MaxSiominDev/homebrew-v2a" "${tap_dir}"
fi

if [ -f "${tap_dir}/Formula/v2a.rb" ] && grep -q "version \"${version}\"" "${tap_dir}/Formula/v2a.rb"; then
  echo "formula is already at ${version}"
  exit 0
fi

sha256="$(curl -fsSL "${url}" | shasum -a 256 | cut -d' ' -f1)"

cat > "${tap_dir}/Formula/v2a.rb" <<EOF
# typed: false
# frozen_string_literal: true

class V2a < Formula
  desc "Extract and compress audio from media files"
  homepage "https://github.com/MaxSiominDev/v2a"
  version "${version}"
  license "MIT"

  depends_on "go" => :build
  depends_on "ffmpeg"
  depends_on arch: :arm64
  depends_on macos: :tahoe

  url "${url}"
  sha256 "${sha256}"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X github.com/MaxSiominDev/v2a/internal/buildinfo.Version=#{version}"), "./cmd/v2a"
  end

  test do
    system "#{bin}/v2a", "version"
  end
end
EOF

if [ -n "${TAP_DIR:-}" ]; then
  echo "formula written to ${tap_dir}/Formula/v2a.rb (no push: TAP_DIR is set)"
  exit 0
fi

cd "${tap_dir}"
git config user.name "github-actions[bot]"
git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
git add Formula/v2a.rb
git commit -m "v2a ${TAG}"
git push
