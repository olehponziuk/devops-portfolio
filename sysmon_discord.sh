#!/usr/bin/env bash
set -euo pipefail

DISCORD_WEBHOOK="${DISCORD_WEBHOOK:-}"
CPU_WARN="${CPU_WARN:-80}"               
RAM_WARN="${RAM_WARN:-90}"               
DISK_WARN="${DISK_WARN:-90}"           
DISK_PATH="${DISK_PATH:-/}"              
ALERT_COOLDOWN_SEC="${ALERT_COOLDOWN_SEC:-1800}"
STATE_DIR="${STATE_DIR:-$HOME/.local/var/sysmon}"
mkdir -p "$STATE_DIR"
LAST_SENT_FILE="$STATE_DIR/last_alert.ts"

if [[ -z "${DISCORD_WEBHOOK}" ]]; then
  echo "DISCORD_WEBHOOK not set" >&2
  exit 1
fi

now_ts() { date +%s; }

within_cooldown() {
  local now last diff
  now="$(now_ts)"
  if [[ -f "$LAST_SENT_FILE" ]]; then
    last="$(cat "$LAST_SENT_FILE" || echo 0)"
  else
    last=0
  fi
  diff=$(( now - last ))
  [[ "$diff" -lt "$ALERT_COOLDOWN_SEC" ]]
}

mark_sent() { now_ts > "$LAST_SENT_FILE"; }

send_alert() {
  local title="$1" body="$2"
  curl -sS -X POST -H 'Content-Type: application/json' \
    -d "$(cat <<JSON
{
  "embeds": [{
    "title": "$title",
    "description": "$body",
    "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  }]
}
JSON
)" "$DISCORD_WEBHOOK" >/dev/null
}

get_cpu_usage_percent() {
  local idle
  idle=$(top -l 1 | grep "CPU usage" | awk -F'[, ]+' '{for(i=1;i<=NF;i++){if($i ~ /idle/){print $(i-1);}}}' | tr -d '%')
  if [[ -z "$idle" ]]; then echo 0; return; fi
  awk -v idle="$idle" 'BEGIN{printf "%.0f", (100 - idle)}'
}

get_ram_usage_percent() {
  local pagesize free speculative active inactive wired compressed total used
  pagesize=$(sysctl -n hw.pagesize)
  eval "$(vm_stat | awk '
    /free/        {gsub("\\.","",$3); print "free=" $3}
    /speculative/ {gsub("\\.","",$3); print "speculative=" $3}
    /active/      {gsub("\\.","",$3); print "active=" $3}
    /inactive/    {gsub("\\.","",$3); print "inactive=" $3}
    /wired down/  {gsub("\\.","",$4); print "wired=" $4}
    /compressed/  {gsub("\\.","",$3); print "compressed=" $3}
  ')"
  total=$(( free + speculative + active + inactive + wired + compressed ))
  used=$(( total - free - speculative ))
  awk -v u="$used" -v t="$total" 'BEGIN{ if(t==0){print 0}else{ printf "%.0f", (u*100.0/t) } }'
}

get_disk_usage_percent() {
  df -H "$DISK_PATH" | awk 'NR==2 {gsub("%","",$5); print $5}'
}

main() {
  local cpu ram disk alerts=""
  cpu="$(get_cpu_usage_percent || echo 0)"
  ram="$(get_ram_usage_percent || echo 0)"
  disk="$(get_disk_usage_percent || echo 0)"

  if (( cpu >= CPU_WARN )); then
    alerts+="• CPU ≥ ${CPU_WARN}% (now: ${cpu}%)\n"
  fi
  if (( ram >= RAM_WARN )); then
    alerts+="• RAM ≥ ${RAM_WARN}% (now: ${ram}%)\n"
  fi
  if (( disk >= DISK_WARN )); then
    alerts+="• Disk ${DISK_PATH} ≥ ${DISK_WARN}% (now: ${disk}%)\n"
  fi

  if [[ -n "$alerts" ]]; then
    if within_cooldown; then
      echo "Within cooldown — not sending alert."
      exit 0
    fi
    local host ts
    host="$(hostname -s)"
    ts="$(date '+%F %T')"
    send_alert " System Alert on ${host}" "time: ${ts}\n${alerts}"
    mark_sent
    echo "Sent Discord alert."
  else
    echo "OK: CPU=${cpu}% RAM=${ram}% DISK(${DISK_PATH})=${disk}%"
  fi
}

main "$@"