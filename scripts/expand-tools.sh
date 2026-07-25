#!/bin/bash
# Auto-discover {{TOOLS_*}} placeholders and expand from prompts/_tools/*.md
# Usage: ./scripts/expand-tools.sh <prompt-file>
# Works on macOS (bash 3.2+) and Linux.

set -euo pipefail

PROMPTS_DIR="$(cd "$(dirname "$0")/../prompts" && pwd)"

input=$(cat "$1")

# Find all {{TOOLS_XXX}} placeholders
placeholders=$(echo "$input" | grep -Eo '\{\{TOOLS_[A-Z_]+\}\}' | sort -u)

for ph in $placeholders; do
    name=$(echo "$ph" | sed 's/{{TOOLS_//;s/}}//' | tr '[:upper:]' '[:lower:]')
    tool_file="$PROMPTS_DIR/_tools/${name}.md"
    if [ -f "$tool_file" ]; then
        tool_content=$(cat "$tool_file")
        input="${input//$ph/$tool_content}"
    else
        echo "WARNING: _tools/${name}.md not found, leaving $ph as-is" >&2
    fi
done

echo "$input"
