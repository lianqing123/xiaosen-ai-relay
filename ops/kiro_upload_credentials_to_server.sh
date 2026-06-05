#!/usr/bin/env bash
set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-23.169.169.107}"
REMOTE_USER="${REMOTE_USER:-root}"
REMOTE_KEY="${REMOTE_KEY:-$HOME/.ssh/taipei}"
JUMP_HOST="${JUMP_HOST:-23.189.248.127}"
JUMP_USER="${JUMP_USER:-root}"
JUMP_PORT="${JUMP_PORT:-49222}"
JUMP_KEY="${JUMP_KEY:-$HOME/.ssh/taipei_rsa}"
REMOTE_IMPORT_DIR="${REMOTE_IMPORT_DIR:-/opt/kiro-gateway/secrets/imports}"
REMOTE_IMPORT_SCRIPT="${REMOTE_IMPORT_SCRIPT:-/opt/kiro-gateway/import_kiro_credentials.sh}"

input_path=""
input_type=""
refresh_token=""
activate_after=0
append_mode=0
profile_arn=""
region=""
api_region=""
auto_detect=0

ssh_base=(
  -o "ProxyCommand=ssh -p $JUMP_PORT -i $JUMP_KEY -o IdentitiesOnly=yes -o BatchMode=yes -o StrictHostKeyChecking=accept-new $JUMP_USER@$JUMP_HOST nc %h %p"
  -o BatchMode=yes
  -o IdentitiesOnly=yes
  -o StrictHostKeyChecking=accept-new
  -o ConnectTimeout=30
  -o ServerAliveInterval=10
  -o ServerAliveCountMax=3
  -i "$REMOTE_KEY"
)

usage() {
  cat <<'USAGE'
Usage:
  kiro_upload_credentials_to_server.sh --file PATH [--activate]
  kiro_upload_credentials_to_server.sh --refresh-token TOKEN [--profile-arn ARN] [--region us-east-1] [--api-region us-east-1] [--activate]
  kiro_upload_credentials_to_server.sh --auto [--activate]

This helper uploads local Kiro IDE / kiro-cli credentials to the server and then
runs /opt/kiro-gateway/import_kiro_credentials.sh remotely.
USAGE
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

remote() {
  ssh "${ssh_base[@]}" "$REMOTE_USER@$REMOTE_HOST" "$@"
}

remote_scp() {
  scp "${ssh_base[@]}" "$1" "$REMOTE_USER@$REMOTE_HOST:$2"
}

detect_type() {
  local path="$1"
  case "$path" in
    *.sqlite|*.sqlite3|*.db)
      echo "sqlite"
      ;;
    *)
      echo "json"
      ;;
  esac
}

auto_select() {
  local candidates=(
    "$HOME/.aws/sso/cache/kiro-auth-token.json"
    "$HOME/.local/share/kiro-cli/data.sqlite3"
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    if [ -f "$candidate" ]; then
      input_path="$candidate"
      input_type="$(detect_type "$candidate")"
      echo "Auto-selected $input_type credentials: $candidate"
      return 0
    fi
  done

  die "no local Kiro credentials found; log in with Kiro IDE or kiro-cli first"
}

upload_json_companion_if_needed() {
  local src="$1"
  python3 - "$src" <<'PY'
import json, pathlib, sys
src = pathlib.Path(sys.argv[1]).expanduser()
try:
    data = json.loads(src.read_text(encoding="utf-8"))
except Exception:
    raise SystemExit
client_hash = data.get("clientIdHash") if isinstance(data, dict) else None
if not client_hash:
    raise SystemExit
companion = src.parent / f"{client_hash}.json"
if companion.exists():
    print(str(companion))
PY
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --file)
        input_path="${2:-}"
        input_type="$(detect_type "$input_path")"
        shift 2
        ;;
      --refresh-token)
        refresh_token="${2:-}"
        input_type="refresh_token"
        shift 2
        ;;
      --profile-arn)
        profile_arn="${2:-}"
        shift 2
        ;;
      --region)
        region="${2:-}"
        shift 2
        ;;
      --api-region)
        api_region="${2:-}"
        shift 2
        ;;
      --append)
        append_mode=1
        shift
        ;;
      --activate)
        activate_after=1
        shift
        ;;
      --auto)
        auto_detect=1
        shift
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        die "unknown argument: $1"
        ;;
    esac
  done
}

main() {
  parse_args "$@"

  if [ "$auto_detect" -eq 1 ]; then
    auto_select
  fi

  remote "mkdir -p '$REMOTE_IMPORT_DIR' && chmod 700 '$REMOTE_IMPORT_DIR'"

  common_flags=()
  if [ "$append_mode" -eq 1 ]; then
    common_flags+=(--append)
  fi
  if [ "$activate_after" -eq 1 ]; then
    common_flags+=(--activate)
  fi
  if [ -n "$profile_arn" ]; then
    common_flags+=(--profile-arn "$profile_arn")
  fi
  if [ -n "$region" ]; then
    common_flags+=(--region "$region")
  fi
  if [ -n "$api_region" ]; then
    common_flags+=(--api-region "$api_region")
  fi

  case "$input_type" in
    json|sqlite)
      [ -n "$input_path" ] || die "--file requires a path"
      [ -f "$input_path" ] || die "file not found: $input_path"
      base="$(basename "$input_path")"
      remote_path="$REMOTE_IMPORT_DIR/$base"
      remote_scp "$input_path" "$remote_path"

      if [ "$input_type" = "json" ]; then
        companion="$(upload_json_companion_if_needed "$input_path" || true)"
        if [ -n "$companion" ] && [ -f "$companion" ]; then
          remote_scp "$companion" "$REMOTE_IMPORT_DIR/$(basename "$companion")"
        fi
        remote "$REMOTE_IMPORT_SCRIPT" --json-file "$remote_path" "${common_flags[@]}"
      else
        remote "$REMOTE_IMPORT_SCRIPT" --sqlite-file "$remote_path" "${common_flags[@]}"
      fi
      ;;
    refresh_token)
      [ -n "$refresh_token" ] || die "--refresh-token requires a token"
      remote "$REMOTE_IMPORT_SCRIPT" --refresh-token "$refresh_token" "${common_flags[@]}"
      ;;
    *)
      usage
      die "choose --file, --refresh-token, or --auto"
      ;;
  esac
}

main "$@"
