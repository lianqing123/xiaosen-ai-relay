#!/usr/bin/env bash
set -euo pipefail

KIRO_HOME="${KIRO_HOME:-/opt/kiro-gateway}"
SECRETS_DIR="$KIRO_HOME/secrets"
IMPORT_DIR="$SECRETS_DIR/imports"
CREDENTIALS_FILE="$SECRETS_DIR/credentials.json"
ACTIVATE_SCRIPT="$KIRO_HOME/activate_after_credentials.sh"

mode="replace"
activate_after=0
input_type=""
input_path=""
refresh_token=""
profile_arn=""
region="${KIRO_REGION:-us-east-1}"
api_region="${KIRO_API_REGION:-}"
show_only=0
auto_detect=0

usage() {
  cat <<'USAGE'
Usage:
  kiro_import_credentials.sh --json-file PATH [--append] [--activate]
  kiro_import_credentials.sh --sqlite-file PATH [--append] [--activate]
  kiro_import_credentials.sh --refresh-token TOKEN [--profile-arn ARN] [--region us-east-1] [--api-region us-east-1] [--append] [--activate]
  kiro_import_credentials.sh --auto [--append] [--activate]
  kiro_import_credentials.sh --show

Notes:
  - This script does not accept or store account passwords.
  - Log in with Kiro IDE or kiro-cli first, then import the generated JSON/SQLite/token.
  - Imported files are copied under /opt/kiro-gateway/secrets/imports and referenced from inside the container.
USAGE
}

die() {
  echo "ERROR: $*" >&2
  exit 1
}

need_python() {
  command -v python3 >/dev/null 2>&1 || die "python3 is required"
}

abs_path() {
  python3 - "$1" <<'PY'
import os, sys
print(os.path.abspath(os.path.expanduser(sys.argv[1])))
PY
}

ensure_dirs() {
  mkdir -p "$SECRETS_DIR" "$IMPORT_DIR"
  chmod 700 "$SECRETS_DIR" "$IMPORT_DIR"
}

interactive_if_needed() {
  if [ "$#" -gt 0 ]; then
    return 0
  fi

  usage
  echo
  echo "Choose import method:"
  echo "  1) Paste refresh token"
  echo "  2) Import Kiro IDE/AWS SSO JSON file"
  echo "  3) Import kiro-cli SQLite database"
  echo "  4) Auto-detect known paths on this server"
  echo "  0) Exit"
  read -r -p "Selection: " choice
  case "$choice" in
    1)
      input_type="refresh_token"
      read -r -s -p "Refresh token: " refresh_token
      echo
      read -r -p "Profile ARN (optional): " profile_arn
      ;;
    2)
      input_type="json"
      read -r -p "JSON file path: " input_path
      ;;
    3)
      input_type="sqlite"
      read -r -p "SQLite file path: " input_path
      ;;
    4)
      auto_detect=1
      ;;
    0)
      exit 0
      ;;
    *)
      die "unknown selection"
      ;;
  esac
  read -r -p "Append to existing credentials.json? [y/N]: " append_answer
  if [[ "$append_answer" =~ ^[Yy]$ ]]; then
    mode="append"
  fi
  read -r -p "Activate Kiro gateway after import? [y/N]: " activate_answer
  if [[ "$activate_answer" =~ ^[Yy]$ ]]; then
    activate_after=1
  fi
}

parse_args() {
  interactive_if_needed "$@"
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --json-file)
        input_type="json"
        input_path="${2:-}"
        shift 2
        ;;
      --sqlite-file)
        input_type="sqlite"
        input_path="${2:-}"
        shift 2
        ;;
      --refresh-token)
        input_type="refresh_token"
        refresh_token="${2:-}"
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
        mode="append"
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
      --show)
        show_only=1
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

auto_select() {
  local candidates=(
    "$IMPORT_DIR/kiro-auth-token.json"
    "$HOME/.aws/sso/cache/kiro-auth-token.json"
    "$IMPORT_DIR/data.sqlite3"
    "$HOME/.local/share/kiro-cli/data.sqlite3"
  )

  local candidate
  for candidate in "${candidates[@]}"; do
    if [ -f "$candidate" ]; then
      input_path="$candidate"
      case "$candidate" in
        *.sqlite3)
          input_type="sqlite"
          ;;
        *)
          input_type="json"
          ;;
      esac
      echo "Auto-selected $input_type credentials: $candidate"
      return 0
    fi
  done

  die "no known Kiro credentials found; provide --json-file, --sqlite-file, or --refresh-token"
}

copy_json_import() {
  local src_abs dest base
  src_abs="$(abs_path "$input_path")"
  [ -f "$src_abs" ] || die "JSON file not found: $src_abs"
  base="$(basename "$src_abs")"
  dest="$IMPORT_DIR/$(date +%Y%m%d%H%M%S)-$base"

  python3 - "$src_abs" "$dest" <<'PY'
import json, pathlib, sys

src = pathlib.Path(sys.argv[1])
dest = pathlib.Path(sys.argv[2])
data = json.loads(src.read_text(encoding="utf-8"))
if not isinstance(data, dict):
    raise SystemExit("JSON credentials must be an object")

client_hash = data.get("clientIdHash")
if client_hash and ("clientId" not in data or "clientSecret" not in data):
    companion = src.parent / f"{client_hash}.json"
    if companion.exists():
        companion_data = json.loads(companion.read_text(encoding="utf-8"))
        if "clientId" in companion_data and "clientId" not in data:
            data["clientId"] = companion_data["clientId"]
        if "clientSecret" in companion_data and "clientSecret" not in data:
            data["clientSecret"] = companion_data["clientSecret"]

if not any(k in data for k in ("refreshToken", "refresh_token", "clientId")):
    raise SystemExit("JSON does not look like a Kiro/AWS SSO credential file")

dest.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
PY

  chmod 600 "$dest"
  echo "/app/secrets/imports/$(basename "$dest")"
}

copy_sqlite_import() {
  local src_abs dest base
  src_abs="$(abs_path "$input_path")"
  [ -f "$src_abs" ] || die "SQLite file not found: $src_abs"
  base="$(basename "$src_abs")"
  dest="$IMPORT_DIR/$(date +%Y%m%d%H%M%S)-$base"

  python3 - "$src_abs" <<'PY'
import sqlite3, sys
path = sys.argv[1]
conn = sqlite3.connect(path)
try:
    cur = conn.cursor()
    cur.execute("SELECT name FROM sqlite_master WHERE type='table' AND name='auth_kv'")
    if not cur.fetchone():
        raise SystemExit("SQLite file does not contain auth_kv table")
finally:
    conn.close()
PY

  install -m 600 "$src_abs" "$dest"
  echo "/app/secrets/imports/$(basename "$dest")"
}

write_credentials() {
  local entry_type="$1"
  local entry_value="$2"

  python3 - "$CREDENTIALS_FILE" "$mode" "$entry_type" "$entry_value" "$profile_arn" "$region" "$api_region" <<'PY'
import hashlib, json, pathlib, sys

credentials_path = pathlib.Path(sys.argv[1])
mode = sys.argv[2]
entry_type = sys.argv[3]
entry_value = sys.argv[4]
profile_arn = sys.argv[5]
region = sys.argv[6]
api_region = sys.argv[7]

if mode == "append" and credentials_path.exists():
    try:
        credentials = json.loads(credentials_path.read_text(encoding="utf-8"))
    except Exception:
        credentials = []
    if not isinstance(credentials, list):
        credentials = []
else:
    credentials = []

if entry_type == "refresh_token":
    entry = {
        "type": "refresh_token",
        "refresh_token": entry_value,
        "region": region,
        "enabled": True,
        "comment": "Imported by kiro_import_credentials.sh"
    }
    if profile_arn:
        entry["profile_arn"] = profile_arn
    if api_region:
        entry["api_region"] = api_region
else:
    entry = {
        "type": entry_type,
        "path": entry_value,
        "enabled": True,
        "comment": "Imported by kiro_import_credentials.sh"
    }
    if profile_arn:
        entry["profile_arn"] = profile_arn
    if region:
        entry["region"] = region
    if api_region:
        entry["api_region"] = api_region

credentials.append(entry)
credentials_path.write_text(json.dumps(credentials, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
credentials_path.chmod(0o600)

sanitized = []
for item in credentials:
    clean = {k: v for k, v in item.items() if k != "refresh_token"}
    if "refresh_token" in item:
        clean["refresh_token_sha256"] = hashlib.sha256(item["refresh_token"].encode()).hexdigest()[:12]
    sanitized.append(clean)
print(json.dumps(sanitized, ensure_ascii=False, indent=2))
PY
}

show_credentials() {
  python3 - "$CREDENTIALS_FILE" <<'PY'
import hashlib, json, pathlib, sys
path = pathlib.Path(sys.argv[1])
if not path.exists():
    print("[]")
    raise SystemExit
data = json.loads(path.read_text(encoding="utf-8"))
out = []
for item in data:
    clean = {k: v for k, v in item.items() if k != "refresh_token"}
    if "refresh_token" in item:
        clean["refresh_token_sha256"] = hashlib.sha256(item["refresh_token"].encode()).hexdigest()[:12]
    out.append(clean)
print(json.dumps(out, ensure_ascii=False, indent=2))
PY
}

main() {
  need_python
  parse_args "$@"
  ensure_dirs

  if [ "$show_only" -eq 1 ]; then
    show_credentials
    exit 0
  fi

  if [ "$auto_detect" -eq 1 ]; then
    auto_select
  fi

  case "$input_type" in
    json)
      [ -n "$input_path" ] || die "--json-file requires a path"
      container_path="$(copy_json_import)"
      write_credentials "json" "$container_path"
      ;;
    sqlite)
      [ -n "$input_path" ] || die "--sqlite-file requires a path"
      container_path="$(copy_sqlite_import)"
      write_credentials "sqlite" "$container_path"
      ;;
    refresh_token)
      [ -n "$refresh_token" ] || die "--refresh-token requires a token"
      write_credentials "refresh_token" "$refresh_token"
      ;;
    *)
      usage
      die "choose --json-file, --sqlite-file, --refresh-token, --auto, or --show"
      ;;
  esac

  if [ "$activate_after" -eq 1 ]; then
    [ -x "$ACTIVATE_SCRIPT" ] || die "activate script not found or not executable: $ACTIVATE_SCRIPT"
    "$ACTIVATE_SCRIPT"
  else
    echo "Imported. To activate after checking credentials, run: $ACTIVATE_SCRIPT"
  fi
}

main "$@"
