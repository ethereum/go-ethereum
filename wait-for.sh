#!/bin/sh

log() {
  echo "[wait-for] [`date +'%Y%m%d%H%M%S'`] $@"
}

usage() {
  echo "Usage: `basename "$0"` [--timeout=15] <HOST>:<PORT> [<HOST_2>:<PORT_2>]"
}

unknown_arg() {
  log "[ERROR] unknown argument: '$@'"
  usage
  exit 2
}

wait_for() {
  timeout=$1 && host=$2 && port=$3
  log "wait '$host':'$port' up to '$timeout'"
  for i in `seq $timeout` ; do
    if nc -z "$host" "$port" > /dev/null 2>&1 ; then
      log "wait finish '$host:$port' [`expr $(date +%s) - $START`] seconds"
      exit 0
    fi
    log "wait attempt '${host}:${port}' [$i]"
    sleep 1
  done
  log "[ERROR] wait timeout of '$timeout' on '$host:$port'"
  exit 1
}

trap 'kill $(jobs -p) &>/dev/null' EXIT

START=$(date +%s)
timeout=15
pids=""
for i in $@ ; do
  case $i in
    --timeout=*) timeout="${i#*=}" ;;
    -t=*) timeout="${i#*=}" ;;
    *:* )
      wait_for "$timeout" "${i%%:*}" "${i##*:}" &
      pids="$pids $!"
    ;;
    *) unknown_arg "$i" ;;
  esac
done

status=0
for pid in $pids; do
  if ! wait $pid ; then
    status=1
  fi
done

log "wait done with status=$status"
exit $status
