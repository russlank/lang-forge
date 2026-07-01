#!/usr/bin/env sh
set -eu

# Install or update the LangForge CLI from release assets.
#
# Typical usage:
#   curl -fsSL https://github.com/russlank/lang-forge/releases/latest/download/install-lang-forge.sh | sh
#   wget -qO- https://github.com/russlank/lang-forge/releases/latest/download/install-lang-forge.sh | sh
#
# Useful overrides:
#   LANG_FORGE_VERSION=v0.1.0
#   LANG_FORGE_REPO_URL=https://github.com/russlank/lang-forge
#   LANG_FORGE_INSTALL_DIR="$HOME/.local/bin"
#   LANG_FORGE_SKIP_CHECKSUM=1

APP_NAME="lang-forge"
DEFAULT_REPO_URL="https://github.com/russlank/lang-forge"

repo_url="${LANG_FORGE_REPO_URL:-$DEFAULT_REPO_URL}"
version="${LANG_FORGE_VERSION:-${VERSION:-latest}}"
install_dir="${LANG_FORGE_INSTALL_DIR:-${PREFIX:-/usr/local}/bin}"
bin_name="${LANG_FORGE_BIN_NAME:-$APP_NAME}"
skip_checksum="${LANG_FORGE_SKIP_CHECKSUM:-0}"
dry_run="${LANG_FORGE_DRY_RUN:-0}"

usage() {
  cat <<USAGE
Usage: install-lang-forge.sh [options]

Installs or updates the LangForge CLI from release assets.

Options:
  --repo-url URL       Release repository URL. Default: $DEFAULT_REPO_URL
  --version VERSION    Release tag, such as v0.1.0, or "latest". Default: latest
  --install-dir DIR    Installation directory. Default: \${PREFIX:-/usr/local}/bin
  --bin-name NAME      Installed executable name. Default: lang-forge
  --no-checksum        Skip SHA256SUMS verification
  --dry-run            Print the selected asset and destination without installing
  -h, --help           Show this help

Environment variables mirror these options:
  LANG_FORGE_REPO_URL, LANG_FORGE_VERSION, LANG_FORGE_INSTALL_DIR,
  LANG_FORGE_BIN_NAME, LANG_FORGE_SKIP_CHECKSUM, LANG_FORGE_DRY_RUN.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo-url)
      repo_url="${2:-}"
      [ -n "$repo_url" ] || { echo "error: --repo-url requires a value" >&2; exit 2; }
      shift 2
      ;;
    --version)
      version="${2:-}"
      [ -n "$version" ] || { echo "error: --version requires a value" >&2; exit 2; }
      shift 2
      ;;
    --install-dir)
      install_dir="${2:-}"
      [ -n "$install_dir" ] || { echo "error: --install-dir requires a value" >&2; exit 2; }
      shift 2
      ;;
    --bin-name)
      bin_name="${2:-}"
      [ -n "$bin_name" ] || { echo "error: --bin-name requires a value" >&2; exit 2; }
      shift 2
      ;;
    --no-checksum)
      skip_checksum=1
      shift
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

repo_url=$(printf '%s' "$repo_url" | sed 's#/*$##')

detect_os() {
  os=$(uname -s 2>/dev/null || echo unknown)
  case "$os" in
    Linux) echo linux ;;
    Darwin) echo darwin ;;
    MINGW*|MSYS*|CYGWIN*) echo windows ;;
    *)
      echo "error: unsupported operating system: $os" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  arch=$(uname -m 2>/dev/null || echo unknown)
  case "$arch" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *)
      echo "error: unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

os="${LANG_FORGE_OS:-$(detect_os)}"
arch="${LANG_FORGE_ARCH:-$(detect_arch)}"

case "$os/$arch" in
  linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64) ;;
  *)
    echo "error: no LangForge release asset for $os/$arch" >&2
    exit 1
    ;;
esac

asset="$APP_NAME-$os-$arch"
if [ "$os" = "windows" ]; then
  asset="$asset.exe"
fi

case "$version" in
  latest) release_path="latest/download" ;;
  v*) release_path="download/$version" ;;
  *) release_path="download/v$version" ;;
esac

asset_url="$repo_url/releases/$release_path/$asset"
checksums_url="$repo_url/releases/$release_path/SHA256SUMS"
destination="$install_dir/$bin_name"

download() {
  url=$1
  output=$2
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL --retry 3 --connect-timeout 15 -o "$output" "$url"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    wget -O "$output" "$url"
    return
  fi
  echo "error: install requires curl or wget" >&2
  exit 1
}

sha256_file() {
  file=$1
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file" | awk '{print $1}'
    return
  fi
  echo "error: checksum verification requires sha256sum or shasum" >&2
  exit 1
}

install_file() {
  source_file=$1
  target_file=$2
  target_dir=$(dirname "$target_file")

  if [ -d "$target_dir" ] && [ -w "$target_dir" ]; then
    install -m 0755 "$source_file" "$target_file"
    return
  fi

  if [ "$(id -u 2>/dev/null || echo 1)" = "0" ]; then
    mkdir -p "$target_dir"
    install -m 0755 "$source_file" "$target_file"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$target_dir"
    sudo install -m 0755 "$source_file" "$target_file"
    return
  fi

  echo "error: $target_dir is not writable and sudo is unavailable" >&2
  echo "hint: rerun with LANG_FORGE_INSTALL_DIR=\$HOME/.local/bin" >&2
  exit 1
}

echo "LangForge installer"
echo "  repository: $repo_url"
echo "  version:    $version"
echo "  asset:      $asset"
echo "  install:    $destination"

if [ "$dry_run" = "1" ]; then
  echo "dry run; no files downloaded or installed"
  exit 0
fi

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/lang-forge-install.XXXXXX")
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT HUP INT TERM

tmp_asset="$tmp_dir/$asset"
tmp_sums="$tmp_dir/SHA256SUMS"

echo "downloading $asset_url"
download "$asset_url" "$tmp_asset"

if [ "$skip_checksum" != "1" ]; then
  echo "downloading $checksums_url"
  download "$checksums_url" "$tmp_sums"
  expected=$(awk -v asset="$asset" '$2 == asset { print $1; found = 1 } END { if (!found) exit 1 }' "$tmp_sums") || {
    echo "error: SHA256SUMS does not contain $asset" >&2
    exit 1
  }
  actual=$(sha256_file "$tmp_asset")
  if [ "$actual" != "$expected" ]; then
    echo "error: checksum mismatch for $asset" >&2
    echo "expected: $expected" >&2
    echo "actual:   $actual" >&2
    exit 1
  fi
  echo "checksum verified"
else
  echo "checksum verification skipped"
fi

install_file "$tmp_asset" "$destination"
echo "installed $destination"

if "$destination" version >/dev/null 2>&1; then
  "$destination" version
else
  echo "warning: installed binary did not run successfully from $destination" >&2
fi
