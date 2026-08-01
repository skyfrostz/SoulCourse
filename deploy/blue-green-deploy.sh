#!/usr/bin/env bash
set -euo pipefail

RELEASE_DIR="${1:-}"
[[ -n "$RELEASE_DIR" && -d "$RELEASE_DIR" ]] || {
  printf 'usage: %s /opt/soulcourse/releases/<release-id>\n' "$0" >&2
  exit 2
}

INSTALL_ROOT="${SOULCOURSE_INSTALL_ROOT:-/opt/soulcourse}"
UPSTREAM_FILE="${SOULCOURSE_NGINX_UPSTREAM:-/etc/nginx/soulcourse-active-upstream.conf}"
LOCK_FILE="${SOULCOURSE_DEPLOY_LOCK:-/run/lock/soulcourse-deploy.lock}"
SLOT_ENV_DIR="${SOULCOURSE_SLOT_ENV_DIR:-/etc/soulcourse}"
RELEASE_MIGRATION_DATABASE_URL="${MIGRATION_DATABASE_URL:-}"
SYSTEMCTL_BIN="${SYSTEMCTL_BIN:-systemctl}"
NGINX_BIN="${NGINX_BIN:-nginx}"
CURL_BIN="${CURL_BIN:-curl}"
RUNUSER_BIN="${RUNUSER_BIN:-runuser}"
PREFLIGHT_USER="${SOULCOURSE_PREFLIGHT_USER:-soulcourse}"
HEALTH_ATTEMPTS="${SOULCOURSE_HEALTH_ATTEMPTS:-30}"
HEALTH_INTERVAL="${SOULCOURSE_HEALTH_INTERVAL:-2}"
DRAIN_SECONDS="${SOULCOURSE_DRAIN_SECONDS:-10}"

exec 9>"$LOCK_FILE"
flock -n 9 || { printf 'another SoulCourse deployment is running\n' >&2; exit 1; }

active_slot=""
if [[ -f "$UPSTREAM_FILE" ]]; then
  active_slot="$(sed -n 's/^# slot: \(blue\|green\)$/\1/p' "$UPSTREAM_FILE" | head -n 1)"
fi
case "$active_slot" in
  blue) inactive_slot="green"; inactive_port="13010" ;;
  green) inactive_slot="blue"; inactive_port="1309" ;;
  *) active_slot=""; inactive_slot="blue"; inactive_port="1309" ;;
esac

slot_env="$SLOT_ENV_DIR/slot-$inactive_slot.env"
[[ -f "$slot_env" ]] || {
  printf 'missing %s\n' "$slot_env" >&2
  exit 1
}
shared_env="$SLOT_ENV_DIR/soulcourse.env"
[[ -f "$shared_env" ]] || {
  printf 'missing %s\n' "$shared_env" >&2
  exit 1
}

# Load the same environment files as the systemd slot unit. This keeps
# preflight checks and migrations aligned with the configuration that the new
# application process will receive; slot values intentionally override shared
# values.
set -a
# shellcheck disable=SC1090
. "$shared_env"
# shellcheck disable=SC1090
. "$slot_env"
set +a

# The elevated migration role is release-scoped. Never accept a stale value
# from a long-lived runtime environment file.
MIGRATION_DATABASE_URL="$RELEASE_MIGRATION_DATABASE_URL"

configured_port="$(sed -n 's/^HTTP_PORT=//p' "$slot_env" | tail -n 1)"
[[ "$configured_port" == "$inactive_port" ]] || {
  printf 'slot %s must define HTTP_PORT=%s\n' "$inactive_slot" "$inactive_port" >&2
  exit 1
}

[[ -n "${DATABASE_URL:-}" ]] || { printf 'DATABASE_URL is required\n' >&2; exit 1; }
[[ -n "${MIGRATION_DATABASE_URL:-}" ]] || { printf 'MIGRATION_DATABASE_URL is required\n' >&2; exit 1; }
[[ "${APP_ENV:-}" == "production" || "${APP_ENV:-}" == "prod" ]] || {
  printf 'APP_ENV must be production\n' >&2
  exit 1
}
[[ -n "${GUANGDONG_DATA_YEAR:-}" ]] || {
  printf 'GUANGDONG_DATA_YEAR is required\n' >&2
  exit 1
}
[[ "${SOULCOURSE_PUBLIC_HEALTH_URL:-}" == https://* ]] || {
  printf 'SOULCOURSE_PUBLIC_HEALTH_URL must be an https URL to /readyz\n' >&2
  exit 1
}
[[ -x "$RELEASE_DIR/deploy/migrate-production.sh" ]] || {
  printf 'release is missing executable deploy/migrate-production.sh\n' >&2
  exit 1
}
[[ -x "$RELEASE_DIR/deploy/preflight.sh" ]] || {
  printf 'release is missing executable deploy/preflight.sh\n' >&2
  exit 1
}
[[ -x "$RELEASE_DIR/soulcourse-cleanup-uploads-linux-amd64" ]] || {
  printf 'release is missing executable soulcourse-cleanup-uploads-linux-amd64\n' >&2
  exit 1
}

# Validate the exact artifact and runtime configuration before migrations can
# affect the active slot. Run as the service account so the non-root check and
# release-file permissions match the eventual systemd process.
if [[ "$(id -u)" -eq 0 ]]; then
  SOULCOURSE_BINARY="$RELEASE_DIR/soulcourse-linux-amd64" \
    "$RUNUSER_BIN" --preserve-environment -u "$PREFLIGHT_USER" -- "$RELEASE_DIR/deploy/preflight.sh"
else
  SOULCOURSE_BINARY="$RELEASE_DIR/soulcourse-linux-amd64" \
    "$RELEASE_DIR/deploy/preflight.sh"
fi

DATABASE_URL="$MIGRATION_DATABASE_URL" "$RELEASE_DIR/deploy/migrate-production.sh"
unset MIGRATION_DATABASE_URL

install -d -m 0755 "$INSTALL_ROOT/bin"
install -m 0755 "$RELEASE_DIR/soulcourse-cleanup-uploads-linux-amd64" "$INSTALL_ROOT/bin/soulcourse-cleanup-uploads"

slot_link="$INSTALL_ROOT/$inactive_slot"
temporary_link="$INSTALL_ROOT/.${inactive_slot}.$$.new"
ln -s "$RELEASE_DIR" "$temporary_link"
mv -Tf "$temporary_link" "$slot_link"

"$SYSTEMCTL_BIN" daemon-reload
"$SYSTEMCTL_BIN" restart "soulcourse@$inactive_slot.service"

healthy=false
for ((attempt = 1; attempt <= HEALTH_ATTEMPTS; attempt++)); do
  if "$CURL_BIN" --fail --silent --show-error --max-time 3 "http://127.0.0.1:$inactive_port/readyz" >/dev/null; then
    healthy=true
    break
  fi
  sleep "$HEALTH_INTERVAL"
done
if [[ "$healthy" != true ]]; then
  "$SYSTEMCTL_BIN" stop "soulcourse@$inactive_slot.service" || true
  printf 'new %s slot failed readiness; active slot was not changed\n' "$inactive_slot" >&2
  exit 1
fi

previous_upstream="$(mktemp)"
next_upstream="$(mktemp)"
trap 'rm -f "$previous_upstream" "$next_upstream"' EXIT
if [[ -f "$UPSTREAM_FILE" ]]; then
  cp "$UPSTREAM_FILE" "$previous_upstream"
fi
cat >"$next_upstream" <<EOF
# slot: $inactive_slot
upstream soulcourse_active {
    server 127.0.0.1:$inactive_port;
    keepalive 32;
}
EOF

install -m 0644 "$next_upstream" "$UPSTREAM_FILE"
if ! "$NGINX_BIN" -t || ! "$NGINX_BIN" -s reload; then
	if [[ -s "$previous_upstream" ]]; then
		install -m 0644 "$previous_upstream" "$UPSTREAM_FILE"
		"$NGINX_BIN" -t && "$NGINX_BIN" -s reload || true
  fi
  "$SYSTEMCTL_BIN" stop "soulcourse@$inactive_slot.service" || true
  printf 'nginx switch failed; previous upstream restored\n' >&2
  exit 1
fi

if ! "$CURL_BIN" --fail --silent --show-error --max-time 10 "$SOULCOURSE_PUBLIC_HEALTH_URL" >/dev/null; then
	if [[ -s "$previous_upstream" ]]; then
		install -m 0644 "$previous_upstream" "$UPSTREAM_FILE"
		"$NGINX_BIN" -t && "$NGINX_BIN" -s reload || true
  fi
  "$SYSTEMCTL_BIN" stop "soulcourse@$inactive_slot.service" || true
  printf 'external HTTPS readiness failed; previous upstream restored when available\n' >&2
  exit 1
fi

sleep "$DRAIN_SECONDS"
if [[ -n "$active_slot" ]]; then
  "$SYSTEMCTL_BIN" stop "soulcourse@$active_slot.service"
fi
printf 'SoulCourse release activated in %s slot on port %s\n' "$inactive_slot" "$inactive_port"
