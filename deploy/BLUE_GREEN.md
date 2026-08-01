# Blue-green production deployment

Install `soulcourse@.service` under `/etc/systemd/system/` and `nginx-soulcourse.conf` under `/etc/nginx/conf.d/`, replacing the example hostname and TLS certificate paths. Create root-owned mode `0600` files:

```text
/etc/soulcourse/soulcourse.env
/etc/soulcourse/slot-blue.env   (HTTP_PORT=1309)
/etc/soulcourse/slot-green.env  (HTTP_PORT=13010)
```

Keep `DATABASE_URL` on the least-privileged runtime role. Supply `MIGRATION_DATABASE_URL` only to the release command. The deployment script loads `/etc/soulcourse/soulcourse.env` and the inactive slot's environment file before running preflight and migrations, matching the systemd unit. Environment files must be root-owned, mode `0600`, and shell-compatible. A release directory must be immutable and contain `soulcourse-linux-amd64`, `soulcourse-cleanup-uploads-linux-amd64`, frontend assets and `deploy/` scripts.

Build both Linux binaries from `backend/`:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../soulcourse-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ../soulcourse-cleanup-uploads-linux-amd64 ./cmd/cleanup-uploads
```

The application validates S3 bucket reachability with `HeadBucket`, but it does not
create or verify provider-specific bucket policies. Before the first public release,
configure and record these settings in the managed S3-compatible provider: bucket
versioning enabled, a 30-day noncurrent-version cleanup rule, and a 30-day cleanup
rule for incomplete multipart uploads. Public image keys must be readable through the
configured CDN; original policy files must remain private. The following commands
are examples only and must be run with the provider's real CLI and credentials:

```bash
aws s3api get-bucket-versioning --bucket "$S3_BUCKET" --endpoint-url "$S3_ENDPOINT"
aws s3api get-bucket-lifecycle-configuration --bucket "$S3_BUCKET" --endpoint-url "$S3_ENDPOINT"
aws s3api head-bucket --bucket "$S3_BUCKET" --endpoint-url "$S3_ENDPOINT"
```

Record the returned versioning and lifecycle JSON as release evidence. Also perform a
real presigned `PUT`, `POST /api/v1/uploads/images/:id/complete`, CDN `GET`, and an
actual registration and password-reset email delivery test. `HeadBucket` and SMTP
`AUTH/NOOP` only prove connectivity and credentials, not CDN policy, object lifecycle,
or mailbox delivery.

Run:

```bash
sudo --preserve-env=MIGRATION_DATABASE_URL \
  deploy/blue-green-deploy.sh /opt/soulcourse/releases/<release-id>
```

The script serializes deployments, applies migrations and the Guangdong data gate, starts the inactive slot, polls its direct `/readyz`, atomically switches the Nginx upstream, validates/reloads Nginx, verifies `SOULCOURSE_PUBLIC_HEALTH_URL` through the public HTTPS path, and drains the old slot. A direct readiness, Nginx, or public HTTPS failure leaves or restores the previous upstream. Keep the previous release directory until the next release has completed its observation window.

Install and enable the cleanup timer once on the host:

```bash
sudo install -m 0644 deploy/soulcourse-upload-cleanup.service deploy/soulcourse-upload-cleanup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now soulcourse-upload-cleanup.timer
sudo systemctl start soulcourse-upload-cleanup.service
sudo systemctl status soulcourse-upload-cleanup.service soulcourse-upload-cleanup.timer
```
