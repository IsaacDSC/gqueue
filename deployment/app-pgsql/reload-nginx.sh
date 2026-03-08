#!/usr/bin/env bash
# Run from repo root or from deployment/app-pgsql.
# Reloads nginx config or restarts the container so changes to nginx.conf take effect.

set -e
cd "$(dirname "$0")"

PROFILE="${1:-gqueue}"

echo "Using profile: $PROFILE"
echo "Reloading nginx config..."
if docker compose --profile "$PROFILE" exec nginx nginx -s reload 2>/dev/null; then
  echo "Nginx reloaded."
else
  echo "Reload failed (container may not be running). Restarting nginx container..."
  docker compose --profile "$PROFILE" up -d --force-recreate nginx
  echo "Nginx restarted."
fi
