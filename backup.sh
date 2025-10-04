#!/usr/bin/env bash
set -euo pipefail

SRC_DIR="${1:-${SRC_DIR:-/path/to/source}}"
DEST_DIR="${2:-${DEST_DIR:-$HOME/Backups}}"
RETAIN="${RETAIN_BACKUPS:-5}"

if [[ ! -d "$SRC_DIR" ]]; then
  echo "No folder: $SRC_DIR" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"
LOG_DIR="${LOG_DIR:-$DEST_DIR/logs}"
mkdir -p "$LOG_DIR"

TS="$(date +%Y%m%d_%H%M%S)"
HOST="$(hostname -s)"
BASE="$(basename "$SRC_DIR")"
ARCHIVE="$DEST_DIR/${BASE}_${HOST}_${TS}.tar.gz"
LOG_FILE="$LOG_DIR/backup_$(date +%Y-%m-%d).log"

{
  echo "[$(date '+%F %T')] Start backup"
  echo "Source: $SRC_DIR"
  echo "Target: $ARCHIVE"

  tar -czf "$ARCHIVE" -C "$(dirname "$SRC_DIR")" "$(basename "$SRC_DIR")"

  if command -v shasum >/dev/null 2>&1; then
    SUM="$(shasum -a 256 "$ARCHIVE" | awk '{print $1}')"
    echo "SHA256: $SUM"
  fi

  echo "[$(date '+%F %T')] Archive created: $ARCHIVE"

  KEEP=$((RETAIN+1))
  TO_DELETE=$(ls -1t "$DEST_DIR"/"${BASE}_${HOST}"_*.tar.gz 2>/dev/null | tail -n +$KEEP || true)
  if [[ -n "${TO_DELETE:-}" ]]; then
    echo "$TO_DELETE" | xargs -I {} rm -f "{}"
    echo "[$(date '+%F %T')] Old backups pruned (kept last $RETAIN)."
  else
    echo "[$(date '+%F %T')] Nothing to prune."
  fi

  echo "[$(date '+%F %T')] Done."
} | tee -a "$LOG_FILE"