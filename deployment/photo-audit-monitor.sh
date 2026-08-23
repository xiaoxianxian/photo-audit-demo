#!/bin/bash
# photo-audit-monitor — Process guardian for Photo Audit Platform backend.
# Monitors the audit-server process and auto-restarts on crash.
# Usage: sudo ./photo-audit-monitor start|stop|status|logs

set -euo pipefail

PID_FILE="/var/run/photo-audit.pid"
LOG_DIR="/var/log/photo-audit"
SERVER_BIN="/opt/photo-audit/audit-server"
ENV_FILE="/opt/photo-audit/.env"
MAX_RESTARTS=5
RESTART_WINDOW=300  # 5 minutes

mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_DIR/monitor.log"
}

is_running() {
    if [ -f "$PID_FILE" ]; then
        local pid
        pid=$(cat "$PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

start() {
    if is_running; then
        local pid
        pid=$(cat "$PID_FILE")
        log "Server already running (PID $pid)"
        return 0
    fi

    if [ ! -x "$SERVER_BIN" ]; then
        log "ERROR: Server binary not found at $SERVER_BIN"
        exit 1
    fi

    log "Starting photo-audit server..."
    if [ -f "$ENV_FILE" ]; then
        set -a; source "$ENV_FILE"; set +a
    fi
    nohup "$SERVER_BIN" >> "$LOG_DIR/server.log" 2>&1 &
    echo $! > "$PID_FILE"
    log "Server started (PID $(cat "$PID_FILE"))"
}

stop() {
    if ! is_running; then
        log "Server not running"
        rm -f "$PID_FILE"
        return 0
    fi

    local pid
    pid=$(cat "$PID_FILE")
    log "Stopping server (PID $pid)..."
    kill "$pid" 2>/dev/null || true
    sleep 2
    if kill -0 "$pid" 2>/dev/null; then
        log "Force killing server (PID $pid)..."
        kill -9 "$pid" 2>/dev/null || true
    fi
    rm -f "$PID_FILE"
    log "Server stopped"
}

status() {
    if is_running; then
        local pid
        pid=$(cat "$PID_FILE")
        echo "Server running (PID $pid)"
        echo "Log: $LOG_DIR/server.log"
    else
        echo "Server not running"
        rm -f "$PID_FILE"
    fi
}

logs() {
    if [ -f "$LOG_DIR/server.log" ]; then
        tail -f "$LOG_DIR/server.log"
    else
        log "No server log found at $LOG_DIR/server.log"
    fi
}

# Main loop for daemon mode
guard() {
    local restart_count=0
    local last_restart_time=0

    log "Guardian started (monitoring $SERVER_BIN)"

    while true; do
        if ! is_running; then
            # Reset counter if window expired
            local now
            now=$(date +%s)
            if [ $((now - last_restart_time)) -gt $RESTART_WINDOW ]; then
                restart_count=0
            fi

            if [ $restart_count -ge $MAX_RESTARTS ]; then
                log "ERROR: Max restarts ($MAX_RESTARTS) reached in $RESTART_WINDOW seconds. Giving up."
                exit 1
            fi

            restart_count=$((restart_count + 1))
            last_restart_time=$(date +%s)
            log "Attempt $restart_count/$MAX_RESTARTS: starting server..."
            start
            sleep 5  # cooldown between restarts
        else
            sleep 10
        fi
    done
}

case "${1:-guard}" in
    start)   start ;;
    stop)    stop ;;
    status)  status ;;
    logs)    logs ;;
    guard)   guard ;;
    *)
        echo "Usage: $0 {start|stop|status|logs|guard}"
        exit 1
        ;;
esac
