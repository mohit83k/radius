#!/bin/bash

set -e

SECRET="testing123"
HOST="radius-controlplane:1813"

USERS=("user1" "user2" "user3")
TYPES=("Start" "Stop" "Interim-Update")

generate_payload() {
  local user=$1
  local type=$2
  cat <<EOF
User-Name = "$user"
Acct-Status-Type = $type
Acct-Session-Id = "sess-${user}-${type}"
NAS-IP-Address = 192.168.1.1
NAS-Port = 0
Calling-Station-Id = "caller"
Called-Station-Id = "callee"
Framed-IP-Address = 10.0.0.100
EOF
}

echo "Starting concurrent RADIUS Accounting tests..."

for user in "${USERS[@]}"; do
  for type in "${TYPES[@]}"; do
    (
      echo "[+] Sending $type for $user"
      generate_payload "$user" "$type" | radclient -x "$HOST" acct "$SECRET"
    ) &
  done
done

wait
echo "✅ All concurrent requests sent."
