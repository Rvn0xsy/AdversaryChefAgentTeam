#!/bin/bash
# Update nuclei-templates from official repo + merge custom templates.
# Run BEFORE building kali-mcp Docker image.
# Usage: ./scripts/update-nuclei-templates.sh

set -euo pipefail

DATA_DIR="$(cd "$(dirname "$0")/../data" && pwd)"
OFFICIAL_DIR="$DATA_DIR/nuclei-templates"
CUSTOM_DIR="$DATA_DIR/nuclei-templates-custom"
OFFICIAL_REPO="https://github.com/projectdiscovery/nuclei-templates.git"

if [ -d "$OFFICIAL_DIR/.git" ]; then
    echo "[1/3] Pulling latest official templates..."
    git -C "$OFFICIAL_DIR" pull --ff-only origin main
else
    echo "[1/3] Cloning official templates (~500MB, one-time)..."
    git clone --depth 1 "$OFFICIAL_REPO" "$OFFICIAL_DIR"
fi

if [ -d "$CUSTOM_DIR" ] && [ "$(ls -A "$CUSTOM_DIR" 2>/dev/null)" ]; then
    echo "[2/3] Merging custom templates..."
    cp -r "$CUSTOM_DIR"/* "$OFFICIAL_DIR"/
    echo "       Custom templates applied."
else
    echo "[2/3] No custom templates to merge."
fi

count=$(find "$OFFICIAL_DIR" -name "*.yaml" -o -name "*.yml" 2>/dev/null | wc -l)
echo "[3/3] Done. $count templates available in $OFFICIAL_DIR"
