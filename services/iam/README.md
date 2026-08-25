# UserHub IAM

IAM (Identity and Access Management) is the security-focused UserHub service.
It owns accounts, protected login identifiers, credentials, authentication,
general authorization, sessions, JWT issuance, and security audit data.

> [!NOTE]
> UserHub is in early development. The current HTTP service exposes health
> endpoints while account and authentication behavior is being specified.

## Requirements

- Go 1.27.0
- Task 3.52.0
- PostgreSQL
- Docker, when building the service image

Run the commands below from the repository root unless noted otherwise.

## Local Development

Provide `DB_URL` through `services/iam/.env`, your shell environment, or a local
secret-management tool without placing its value in command-line arguments.
The IAM migration tasks automatically load `services/iam/.env`. Then apply
migrations and start the service:

```sh
task iam:migration-up
task iam:dev
```

The development task loads `services/iam/.env`, starts the service in
development mode, and rebuilds it after source files are saved. A build error
is printed without stopping the watcher; the next source change retries the
build. The service listens on `:8080` by default.

### Health Endpoints

| Endpoint            | Purpose              |
| ------------------- | -------------------- |
| `GET /health/live`  | Process liveness     |
| `GET /health/ready` | PostgreSQL readiness |

The `scratch` image includes an `iam healthcheck` command that converts the
local readiness response into a process exit status. Docker deployments can
invoke it from the running service container without adding `curl` or `wget`:

```yaml
healthcheck:
  test: ["CMD", "/iam", "healthcheck"]
  interval: 10s
  timeout: 4s
  retries: 5
  start_period: 5s
```

The command requests `GET /health/ready` through the configured local HTTP
port, has its own three-second deadline, and succeeds only for HTTP 200. Docker
reports this result as container health; it does not by itself restart an
unhealthy container or remove it from application traffic. Kubernetes should
continue to use native HTTP readiness and liveness probes.

## Development Commands

| Command           | Description                                      |
| ----------------- | ------------------------------------------------ |
| `task iam:build`  | Build all IAM packages                           |
| `task iam:test`   | Run IAM tests                                    |
| `task iam:dev`    | Run IAM and reload it after source changes       |
| `task iam:verify` | Run tests, module checks, and `go vet`           |
| `task iam:image`  | Build the `userhub/iam:dev` container image      |

## Database Migrations

IAM migrations are embedded PostgreSQL migrations. Each migration has an `up`
and a `down` SQL file.

For local development, create and run migrations with:

```sh
task iam:migration-create NAME=add_accounts
task iam:migration-up
task iam:migration-down
```

`migration-up` and `migration-down` read `DB_URL` from the environment and load
it from `services/iam/.env` when that file exists. This loading is scoped to the
IAM migration task, so it does not affect Identity tasks. Do not provide
credentials as Task variables or command-line arguments, because they may be
retained in shell history or exposed through the process list. The local `.env`
file is excluded from Git.

Migration names must contain lowercase letters, digits, and single underscores.
`migration-down` rolls back one migration at a time.

> [!IMPORTANT]
> Migration creation and rollback are development-only Taskfile operations. The
> production IAM command exposes only `iam migrate`, which applies pending
> migrations forward and does not support rollback.

Run production migrations as an explicit deployment step before starting the
service:

```sh
docker run --rm --env-file services/iam/.env userhub/iam:dev migrate
docker run --rm --env-file services/iam/.env userhub/iam:dev start
```

PostgreSQL is IAM's only supported durable database. IAM always maintains an
in-process pgxpool. Runtime traffic uses `DB_URL_POOL` when configured and
otherwise uses `DB_URL`. A configured pooled endpoint is authoritative: IAM
reports itself unready rather than bypassing PgBouncer through the direct URL.
Migrations always use `DB_URL`.

In Docker Compose, run migrations as a one-shot service before starting IAM:

```yaml
services:
  iam-migrate:
    image: userhub/iam:dev
    command: ["migrate"]
    environment:
      DB_URL: postgres://user:password@postgres:5432/iam?sslmode=disable

  iam:
    image: userhub/iam:dev
    command: ["start"]
    environment:
      DB_URL: postgres://user:password@postgres:5432/iam?sslmode=disable
    depends_on:
      iam-migrate:
        condition: service_completed_successfully
```

### Development PostgreSQL

The root `docker-compose.yml` provides PostgreSQL and PgBouncer for local
development:

```sh
docker compose up -d postgres pgbouncer
docker compose down
```

To verify migrations against a clean database, recreate an isolated database,
apply the migrations, inspect their state, and apply them again:

```sh
docker compose up -d postgres
docker compose exec postgres dropdb --if-exists -U userhub userhub_migration_test
docker compose exec postgres createdb -U userhub userhub_migration_test
DB_URL='postgres://userhub:userhub@localhost:5432/userhub_migration_test?sslmode=disable' go run ./services/iam/cmd migrate
docker compose exec postgres psql -U userhub -d userhub_migration_test -c 'TABLE schema_migrations;'
DB_URL='postgres://userhub:userhub@localhost:5432/userhub_migration_test?sslmode=disable' go run ./services/iam/cmd migrate
```

The second migration run verifies that an up-to-date schema exits successfully
without changes. Migration verification is manual; the IAM test suite does not
connect to PostgreSQL.

## Configuration

Configuration is read from environment variables.

| Name                             | Required | Default      | Purpose / Constraints                                                        |
| -------------------------------- | -------- | ------------ | ---------------------------------------------------------------------------- |
| `DB_URL`                         | yes      | -            | Direct PostgreSQL URL; used by migrations and as the runtime fallback         |
| `DB_URL_POOL`                    | no       | -            | PgBouncer URL; authoritative for runtime when configured                      |
| `APP_ENV`                        | no       | `production` | Application environment; `production` or `development`                       |
| `HTTP_ADDR`                      | no       | `:8080`      | HTTP listen address; for example, `127.0.0.1:9000`                            |
| `HTTP_MAX_HEADER_BYTES`          | no       | `16384`      | Maximum request header size; at least `4096` bytes                            |
| `HTTP_MAX_BODY_BYTES`            | no       | `65536`      | Maximum buffered request body size; at least `8192` bytes                     |
| `HTTP_WRITE_TIMEOUT`             | no       | `10s`        | HTTP response write timeout; at least `1s`                                    |
| `HTTP_READ_TIMEOUT`              | no       | `5s`         | HTTP request read timeout; at least `1s`                                      |
| `HTTP_READ_HEADER_TIMEOUT`       | no       | `2s`         | HTTP header read timeout; at least `1s`                                       |
| `HTTP_IDLE_TIMEOUT`              | no       | `30s`        | Keep-alive idle timeout; at least `1s`                                        |
| `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT` | no       | `30s`        | Graceful shutdown timeout; at least `1s`                                      |
| `DB_MAX_OPEN_CONNS`              | no       | `20`         | Maximum pgxpool connections per IAM process; at least `1`                     |
| `DB_MIN_CONNS`                   | no       | `2`          | Minimum pgxpool connections; cannot exceed `DB_MAX_OPEN_CONNS`                |
| `DB_CONN_MAX_LIFETIME`           | no       | `0`          | Maximum connection lifetime; `0` is unlimited                                |
| `DB_CONN_MAX_IDLE_TIME`          | no       | `0`          | Maximum connection idle time; `0` is unlimited                               |

Example direct development URL:

```text
postgres://userhub:userhub@localhost:5432/userhub?sslmode=disable
```

See the repository-level [`README.md`](../../README.md) for deployment context
and service boundaries.
