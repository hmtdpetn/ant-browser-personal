#!/usr/bin/env bash
set -euo pipefail

target_skills_dir=""
config_file=""
browser_profile_name="ant-chrome"
base_url="${ANT_CHROME_BASE_URL:-http://127.0.0.1:19876}"
api_header="${ANT_CHROME_API_HEADER:-X-Ant-Api-Key}"
api_key="${ANT_CHROME_API_KEY:-}"
color="#0F766E"
set_default_profile="false"
dry_run="false"

print_help() {
  cat <<'EOF'
Usage: install_ant_chrome_openclaw.sh [options]

Options:
  --target-skills-dir PATH   OpenClaw skills directory
  --config-file PATH         OpenClaw config path (for example openclaw.json)
  --browser-profile NAME     Browser profile name to create/update (default: ant-chrome)
  --base-url URL             Ant Browser LaunchServer base URL
  --api-header NAME          API auth header name
  --api-key VALUE            API key written into the skill entry
  --color VALUE              Browser profile color (default: #0F766E)
  --set-default-profile      Set browser.defaultProfile to the selected profile
  --dry-run                  Print detected paths without writing files
  --help                     Show this help
EOF
}

resolve_existing_path() {
  for candidate in "$@"; do
    if [[ -n "$candidate" && -e "$candidate" ]]; then
      printf '%s\n' "$(cd "$(dirname "$candidate")" && pwd)/$(basename "$candidate")"
      return 0
    fi
  done
  return 1
}

detect_skills_dir() {
  local xdg_config_home="${XDG_CONFIG_HOME:-$HOME/.config}"
  resolve_existing_path \
    "${OPENCLAW_SKILLS_DIR:-}" \
    "${OPENCLAW_HOME:-}/skills" \
    "$HOME/.openclaw/skills" \
    "$xdg_config_home/openclaw/skills" \
    "$HOME/.config/openclaw/skills" || true
}

detect_config_file() {
  local xdg_config_home="${XDG_CONFIG_HOME:-$HOME/.config}"
  resolve_existing_path \
    "${OPENCLAW_CONFIG:-}" \
    "${OPENCLAW_HOME:-}/openclaw.json" \
    "${OPENCLAW_HOME:-}/config.json" \
    "$HOME/.openclaw/openclaw.json" \
    "$HOME/.openclaw/config.json" \
    "$xdg_config_home/openclaw/openclaw.json" \
    "$xdg_config_home/openclaw/config.json" \
    "$HOME/.config/openclaw/openclaw.json" \
    "$HOME/.config/openclaw/config.json" || true
}

can_run_python() {
  local cmd=("$@")
  "${cmd[@]}" -V >/dev/null 2>&1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target-skills-dir)
      target_skills_dir="${2:-}"
      shift 2
      ;;
    --config-file)
      config_file="${2:-}"
      shift 2
      ;;
    --browser-profile)
      browser_profile_name="${2:-}"
      shift 2
      ;;
    --base-url)
      base_url="${2:-}"
      shift 2
      ;;
    --api-header)
      api_header="${2:-}"
      shift 2
      ;;
    --api-key)
      api_key="${2:-}"
      shift 2
      ;;
    --color)
      color="${2:-}"
      shift 2
      ;;
    --set-default-profile)
      set_default_profile="true"
      shift
      ;;
    --dry-run)
      dry_run="true"
      shift
      ;;
    --help|-h)
      print_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      print_help >&2
      exit 1
      ;;
  esac
done

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source_skill_root="$(cd "$script_dir/.." && pwd)"
skill_name="ant-chrome-openclaw"

if [[ -z "$target_skills_dir" ]]; then
  target_skills_dir="$(detect_skills_dir)"
fi

if [[ -z "$target_skills_dir" ]]; then
  echo "target skills dir is required because no existing OpenClaw skills directory was detected" >&2
  exit 1
fi

if [[ -z "$config_file" ]]; then
  config_file="$(detect_config_file)"
fi

skill_destination="$target_skills_dir/$skill_name"

if [[ "$dry_run" == "true" ]]; then
  echo "[dry-run] source skill: $source_skill_root"
  echo "[dry-run] target skills dir: $target_skills_dir"
  echo "[dry-run] install destination: $skill_destination"
  if [[ -n "$config_file" ]]; then
    echo "[dry-run] config file to update: $config_file"
  else
    echo "[dry-run] config file: not set; only files would be installed"
  fi
  exit 0
fi

mkdir -p "$target_skills_dir"

backup_path=""
if [[ -e "$skill_destination" ]]; then
  backup_path="${skill_destination}.backup-$(date +%Y%m%d%H%M%S)"
  mv "$skill_destination" "$backup_path"
fi

cp -R "$source_skill_root" "$skill_destination"

config_updated="false"
if [[ -n "$config_file" ]]; then
  python_cmd=()
  if command -v python3 >/dev/null 2>&1 && can_run_python python3; then
    python_cmd=(python3)
  elif command -v python >/dev/null 2>&1 && can_run_python python; then
    python_cmd=(python)
  elif command -v py >/dev/null 2>&1 && can_run_python py -3; then
    python_cmd=(py -3)
  elif command -v py.exe >/dev/null 2>&1 && can_run_python py.exe -3; then
    python_cmd=(py.exe -3)
  else
    echo "python3, python, or py -3 is required to update the OpenClaw config file" >&2
    exit 1
  fi

  export INSTALL_CONFIG_FILE="$config_file"
  export INSTALL_BROWSER_PROFILE="$browser_profile_name"
  export INSTALL_BASE_URL="$base_url"
  export INSTALL_API_HEADER="$api_header"
  export INSTALL_API_KEY="$api_key"
  export INSTALL_COLOR="$color"
  export INSTALL_SET_DEFAULT_PROFILE="$set_default_profile"

  "${python_cmd[@]}" - <<'PY'
import json
import os
from pathlib import Path

config_file = Path(os.environ["INSTALL_CONFIG_FILE"])
browser_profile = os.environ["INSTALL_BROWSER_PROFILE"]
base_url = os.environ["INSTALL_BASE_URL"]
api_header = os.environ["INSTALL_API_HEADER"]
api_key = os.environ["INSTALL_API_KEY"]
color = os.environ["INSTALL_COLOR"]
set_default_profile = os.environ["INSTALL_SET_DEFAULT_PROFILE"] == "true"
skill_name = "ant-chrome-openclaw"

data = {}
if config_file.exists():
    raw = config_file.read_text(encoding="utf-8").strip()
    if raw:
        data = json.loads(raw)

browser = data.setdefault("browser", {})
browser["enabled"] = True
profiles = browser.setdefault("profiles", {})
profile = profiles.setdefault(browser_profile, {})
profile["cdpUrl"] = base_url
profile.setdefault("color", color)
if set_default_profile or not str(browser.get("defaultProfile", "")).strip():
    browser["defaultProfile"] = browser_profile

skills = data.setdefault("skills", {})
entries = skills.setdefault("entries", {})
skill_entry = entries.setdefault(skill_name, {})
skill_entry["enabled"] = True
env_map = skill_entry.setdefault("env", {})
env_map["ANT_CHROME_BASE_URL"] = base_url
env_map["ANT_CHROME_API_HEADER"] = api_header
if api_key.strip():
    skill_entry["apiKey"] = api_key

config_file.parent.mkdir(parents=True, exist_ok=True)
config_file.write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

  config_updated="true"
fi

echo "Installed skill to: $skill_destination"
if [[ -n "$backup_path" ]]; then
  echo "Backed up previous skill to: $backup_path"
fi
if [[ "$config_updated" == "true" ]]; then
  echo "Updated OpenClaw config: $config_file"
else
  echo "No config file updated. Merge openclaw.config.sample.json manually or rerun with --config-file."
fi
echo "Browser profile name: $browser_profile_name"
echo "Base URL: $base_url"
