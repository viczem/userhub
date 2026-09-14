# UserHub Service

UserHub Service is the independently deployable runtime for one regional cell.
The current implementation provides typed configuration, PostgreSQL lifecycle,
HTTP safeguards, health reporting, graceful shutdown.
Account, profile, authentication, and delivery behavior will be added
through separate specifications.

The service is distributed as one `userhub` executable and container image.
There are no IAM or Identity compatibility executables.

## Commands

| Command               | Purpose                                                       |
| --------------------- | ------------------------------------------------------------- |
| `userhub start`       | Run the HTTP service                                          |
| `userhub migrate`     | Apply all pending embedded forward-only PostgreSQL migrations |
| `userhub healthcheck` | Probe the local `/health/ready` endpoint                      |

The production migration command does not support down migrations. Development
tasks can create migration pairs and roll back one version while authoring a
migration, but deployed schema changes remain forward-only.

## Development

Provide `DB_URL` through `services/userhub/.env` or the shell, then run:

```sh
docker compose up -d postgres pgbouncer
task userhub:migration-up
task userhub:dev
```

Component commands available from the repository root:

| Command                                          | Purpose                                                               |
| ------------------------------------------------ | --------------------------------------------------------------------- |
| `task userhub:build`                             | Build UserHub Service packages                                        |
| `task userhub:test`                              | Run UserHub Service tests                                             |
| `task userhub:cmd -- <arguments>`                | Run UserHub CLI with `services/userhub/.env` from the repository root |
| `task userhub:verify`                            | Run tests, module checks, and `go vet`                                |
| `task userhub:image`                             | Build `userhub/userhub:dev` from the repository root context          |
| `task userhub:migration-create NAME=add_example` | Create the next migration pair                                        |
| `task userhub:migration-up`                      | Apply pending development migrations                                  |
| `task userhub:migration-down`                    | Roll back one development migration                                   |

## Health And Shutdown

- `GET /health/live` reports process liveness without probing dependencies.
- `GET /health/ready` returns success only while the service accepts work and
  its selected runtime PostgreSQL endpoint responds.
- `SIGINT` and `SIGTERM` remove readiness, drain HTTP work within
  `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT`, and close PostgreSQL resources.

The scratch image contains the native readiness command:

```yaml
healthcheck:
    test: ["CMD", "/userhub", "healthcheck"]
```

## Deployment

PostgreSQL is the only durable database. `DB_URL` must be a direct PostgreSQL
endpoint used by migrations. When `DB_URL_POOL` is configured, all runtime
repository traffic uses it and does not fall back to `DB_URL` during an outage.

Run migrations successfully before starting or rolling out UserHub Service:

```sh
docker run --rm userhub/userhub:dev migrate
docker run --rm userhub/userhub:dev start
```

Each regional cell must use regional PostgreSQL, backup, logging, and
key-management infrastructure when claiming data residency. Multiple UserHub
replicas in one cell share PostgreSQL.

## Environment

Runtime configuration is environment-only. Invalid values fail command startup
with the variable name and constraint but without the supplied value.

| Variable                         | Required | Default      | Purpose and constraint                                                                                                                |
| -------------------------------- | -------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `APP_ENV`                        | no       | `production` | `production` or `development`                                                                                                         |
| `HTTP_ADDR`                      | no       | `:8080`      | TCP host and numeric port 1-65535                                                                                                     |
| `HTTP_MAX_HEADER_BYTES`          | no       | `16384`      | Maximum HTTP header bytes; at least `4096`                                                                                            |
| `HTTP_MAX_BODY_BYTES`            | no       | `65536`      | Maximum request body bytes; at least `8192`                                                                                           |
| `HTTP_WRITE_TIMEOUT`             | no       | `10s`        | HTTP write timeout; at least `1s`                                                                                                     |
| `HTTP_READ_TIMEOUT`              | no       | `5s`         | HTTP read timeout; at least `1s`                                                                                                      |
| `HTTP_READ_HEADER_TIMEOUT`       | no       | `2s`         | HTTP header timeout; at least `1s`                                                                                                    |
| `HTTP_IDLE_TIMEOUT`              | no       | `30s`        | HTTP idle timeout; at least `1s`                                                                                                      |
| `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT` | no       | `30s`        | Complete shutdown bound; at least `1s`                                                                                                |
| `DB_URL`                         | yes      | none         | Direct PostgreSQL connection URL for migrations and default runtime traffic                                                           |
| `DB_URL_POOL`                    | no       | empty        | Authoritative pooled runtime PostgreSQL URL                                                                                           |
| `KEYRING_HMAC`                   | yes      | none         | HMAC keyring; one positive active ID and zero or more negative retained IDs; each key is sensitive Base64URL-encoded 32-byte material |
| `KEYRING_ENCRYPTION`             | yes      | none         | AES-256-GCM keyring in the same format; key material must not be shared with `KEYRING_HMAC`                                           |
| `DB_MAX_OPEN_CONNS`              | no       | `20`         | Maximum connections per process; at least `1`                                                                                         |
| `DB_MIN_CONNS`                   | no       | `2`          | Minimum connections; nonnegative and no greater than maximum                                                                          |
| `DB_CONN_MAX_LIFETIME`           | no       | `0`          | Maximum connection lifetime; `0` means unlimited                                                                                      |
| `DB_CONN_MAX_IDLE_TIME`          | no       | `0`          | Maximum idle time; `0` means unlimited                                                                                                |

## Keyring Generation And Rotation

> [!NOTE]
> Rotate a key only after compromise, a required policy change, or expiration of
> its permitted lifetime—not for routine restarts or redeployments. Generate and
> store the new key independently, deploy it as the sole positive active ID, and
> retain previous keys with negative IDs until they are no longer needed for
> reads or recovery. HMAC and encryption keyrings rotate independently.

`KEYRING_HMAC` and `KEYRING_ENCRYPTION` contain independently generated,
32-byte keys encoded as padded Base64URL. Generate each key with Linux system
utilities:

```sh
head -c 32 /dev/urandom | basenc --base64url
```

Do not reuse key material between keyrings; store it in deployment secret
management, not Git or logs.

A keyring is a comma-separated list of entries:

```text
<signed-key-id>:<base64url-encoded-32-byte-key>[,...]
```

Exactly one positive ID is active for new values; negative IDs retain old keys
for reads. IDs are stored as positive values in `*_key_id` metadata. Key IDs
must not be `0`, `-32768`, or duplicate after removing their sign; material
must be padded Base64URL decoding to 32 bytes.

For example, this keyring has active key ID `1`:

```text
1:<active-key-material>
```

The second example rotates from key ID `1` to `2`: both keys remain available
for reads, while new values use `2`. Remove an old key only after no data,
lookup, backup, or recovery path requires it.

```text
-1:<previous-key-material>,2:<new-active-key-material>
```
