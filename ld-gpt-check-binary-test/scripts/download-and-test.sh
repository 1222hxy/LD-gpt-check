#!/usr/bin/env bash
set -euo pipefail

repo="${LD_GPT_CHECK_REPO:-1222hxy/LD-gpt-check}"
install_dir="${LD_GPT_CHECK_INSTALL_DIR:-.ld-gpt-check-bin}"
run_test=0
model=""
effort="medium"
timeout="30m"

usage() {
  cat <<'USAGE'
Download the latest LD-gpt-check release binary and optionally run one local test.

Usage:
  download-and-test.sh [--run-test] [--install-dir DIR] [--model MODEL] [--effort EFFORT] [--timeout DURATION]

Options:
  --run-test         Run one benchmark pass after download and smoke checks.
  --install-dir DIR  Directory for the downloaded binary. Default: .ld-gpt-check-bin
  --model MODEL      Optional model passed to ld-gpt-check run.
  --effort EFFORT    Reasoning effort: low, medium, high, or xhigh. Default: medium
  --timeout DURATION Per-test timeout. Default: 30m
  -h, --help         Show this help.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --run-test)
      run_test=1
      shift
      ;;
    --install-dir)
      install_dir="${2:?missing value for --install-dir}"
      shift 2
      ;;
    --model)
      model="${2:?missing value for --model}"
      shift 2
      ;;
    --effort)
      effort="${2:?missing value for --effort}"
      shift 2
      ;;
    --timeout)
      timeout="${2:?missing value for --timeout}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

detect_os() {
  case "$(uname -s)" in
    Linux*) echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *)
      echo "unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    armv7l|armv7*) echo "armv7" ;;
    armv6l|armv6*) echo "armv6" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

download() {
  url="$1"
  output="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fL "$url" -o "$output"
  elif command -v wget >/dev/null 2>&1; then
    wget -O "$output" "$url"
  else
    echo "curl or wget is required to download the binary" >&2
    exit 1
  fi
}

os="$(detect_os)"
arch="$(detect_arch)"
asset="ld-gpt-check_${os}_${arch}"
if [ "$os" = "windows" ]; then
  asset="${asset}.exe"
fi

mkdir -p "$install_dir"
bin_path="${install_dir}/ld-gpt-check"
if [ "$os" = "windows" ]; then
  bin_path="${bin_path}.exe"
fi

url="https://github.com/${repo}/releases/latest/download/${asset}"
echo "Downloading ${asset}"
echo "Source: ${url}"
download "$url" "$bin_path"
chmod +x "$bin_path" 2>/dev/null || true

echo
echo "Binary: ${bin_path}"
"$bin_path" version

echo
echo "Checking built-in suites"
"$bin_path" run --no-remote-questions --list-suites

if [ "$run_test" -eq 0 ]; then
  echo
  echo "Downloaded and smoke-checked. Re-run with --run-test to execute one benchmark pass."
  exit 0
fi

if ! command -v codex >/dev/null 2>&1 && ! command -v codex.cmd >/dev/null 2>&1; then
  echo "Codex CLI was not found in PATH. Install and log in to Codex before running the benchmark." >&2
  exit 1
fi

run_args=(run --no-remote-questions -n 1 --timeout "$timeout" -r "$effort")
if [ -n "$model" ]; then
  run_args+=(-m "$model")
fi

echo
echo "Running one LD-gpt-check benchmark pass"
"$bin_path" "${run_args[@]}"
