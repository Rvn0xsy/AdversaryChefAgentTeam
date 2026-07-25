#!/bin/bash
# Validate all prompt files have only known placeholders
# {{TOOLS_*}} are auto-validated (expand-tools.sh handles them)
# Usage: ./scripts/validate-prompts.sh

set -euo pipefail
PROMPTS_DIR="$(cd "$(dirname "$0")/../prompts" && pwd)"

KNOWN="{{WORKSPACE}} {{MCP_ASSET_URL}} {{MCP_KALI_URL}} {{MCP_MYTHIC_URL}}"

for f in "$PROMPTS_DIR"/*.md; do
    base=$(basename "$f")
    unknown=$(grep -Eo '\{\{[A-Z_]+\}\}' "$f" 2>/dev/null | sort -u | while read -r p; do
        if [[ "$p" =~ ^\{\{TOOLS_[A-Z_]+\}\}$ ]]; then
            continue
        fi
        if ! echo "$KNOWN" | grep -qF "$p"; then
            echo "  $p"
        fi
    done)
    if [ -n "$unknown" ]; then
        echo "x $base: unknown placeholders:$unknown"
    else
        echo "ok $base"
    fi
done
