# UserHub Directory

UserHub Directory is the optional, project-owned service for multi-country
UserHub installations. It will coordinate global login ownership and discover
trusted regional UserHub cells. A single-cell UserHub installation does not
need this service.

## Commands

| Command | Purpose |
| --- | --- |
| `userhub-directory start` | Run the directory HTTP runtime |
| `userhub-directory migrate` | Apply all pending embedded forward-only PostgreSQL migrations |
| `userhub-directory healthcheck` | Probe the local `/health/ready` endpoint |

The production migration command does not support down migrations. Development
tasks can create migration pairs and roll back one version while authoring a
migration, but deployed schema changes remain forward-only.

## Development

Provide `DB_URL` and `KEYRING_HMAC` through `services/userhub-directory/.env`
or the shell, then run:

```sh
docker compose up -d postgres pgbouncer
task userhub-directory:migration-up
task userhub-directory:dev
```

Component commands available from the repository root:

| Command                                                    | Purpose                                                                        |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `task userhub-directory:build`                             | Build UserHub Directory packages                                               |
| `task userhub-directory:test`                              | Run UserHub Directory tests                                                    |
| `task userhub-directory:cmd -- <arguments>`                | Run the Directory CLI with `services/userhub-directory/.env`                  |
| `task userhub-directory:dev`                               | Run the Directory with automatic reload                                        |
| `task userhub-directory:verify`                            | Run tests, module checks, and `go vet`                                         |
| `task userhub-directory:image`                             | Build `userhub/userhub-directory:dev` from the repository root context        |
| `task userhub-directory:migration-create NAME=add_example` | Create the next migration pair                                                 |
| `task userhub-directory:migration-up`                      | Apply pending development migrations                                           |
| `task userhub-directory:migration-down`                    | Roll back one development migration                                            |

## Health And Shutdown

- `GET /health/live` reports process liveness without probing dependencies.
- `GET /health/ready` returns success only while the service accepts work and
  its PostgreSQL endpoint responds.
- `SIGINT` and `SIGTERM` remove readiness, drain HTTP work within
  `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT`, and close PostgreSQL resources.

## Deployment

PostgreSQL is the only durable database. `DB_URL` must be a direct PostgreSQL
endpoint used by migrations. When `DB_URL_POOL` is configured, all runtime
repository traffic uses it and does not fall back to `DB_URL` during an outage.

Run migrations successfully before starting or rolling out UserHub Directory:

```sh
docker run --rm userhub/userhub-directory:dev migrate
docker run --rm userhub/userhub-directory:dev start
```

## Environment

Runtime configuration is environment-only. Invalid values fail command startup.

| Variable                         | Required | Default      | Purpose and constraint                                                                                                                |
| -------------------------------- | -------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| `APP_ENV`                        | no       | `production` | `production` or `development`                                                                                                         |
| `HTTP_ADDR`                      | no       | `:8080`      | TCP host and numeric port                                                                                                             |
| `HTTP_MAX_HEADER_BYTES`          | no       | `16384`      | Maximum HTTP header bytes; at least `4096`                                                                                            |
| `HTTP_MAX_BODY_BYTES`            | no       | `65536`      | Maximum request body bytes; at least `8192`                                                                                           |
| `HTTP_WRITE_TIMEOUT`             | no       | `10s`        | HTTP write timeout                                                                                                                    |
| `HTTP_READ_TIMEOUT`              | no       | `5s`         | HTTP read timeout                                                                                                                     |
| `HTTP_READ_HEADER_TIMEOUT`       | no       | `2s`         | HTTP header timeout                                                                                                                   |
| `HTTP_IDLE_TIMEOUT`              | no       | `30s`        | HTTP idle timeout                                                                                                                     |
| `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT` | no       | `30s`        | Complete shutdown bound                                                                                                               |
| `DB_URL`                         | yes      | none         | Direct PostgreSQL connection URL for migrations and default runtime traffic                                                           |
| `DB_URL_POOL`                    | no       | empty        | Authoritative pooled runtime PostgreSQL URL                                                                                           |
| `KEYRING_HMAC`                   | yes      | none         | HMAC keyring with one active positive ID and optional retained negative IDs; material is padded Base64URL-encoded 32-byte data       |
| `DB_MAX_OPEN_CONNS`              | no       | `20`         | Maximum connections per process; at least `1`                                                                                         |
| `DB_MIN_CONNS`                   | no       | `2`          | Minimum connections; nonnegative and no greater than maximum                                                                          |
| `DB_CONN_MAX_LIFETIME`           | no       | `0`          | Maximum connection lifetime; `0` means unlimited                                                                                      |
| `DB_CONN_MAX_IDLE_TIME`          | no       | `0`          | Maximum idle time; `0` means unlimited                                                                                                |

`KEYRING_HMAC` is a comma-separated list of
`<signed-key-id>:<base64url-encoded-32-byte-key>` entries. Exactly one positive
ID is active for new values; negative IDs retain old keys for reads. Do not
reuse or commit key material.
