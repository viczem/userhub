# UserHub

UserHub is a reusable implementation of user management for applications. It
is composed of two independently deployable Go services connected by an
opaque, stable subject identifier.

> [!NOTE]
> UserHub is in early development. Service behavior and runtime configuration
> are still being specified.

## IAM

IAM owns security-sensitive account data and operations, including login
identifiers, credentials, authentication, general authorization, sessions,
JWT issuance, and security audit.

### Environment Variables

| Name | Required | Purpose | Example / Notes |
| --- | --- | --- | --- |
| `APP_ENV` | no | Application runtime environment | Defaults to `production`; allowed values: `production`, `development` |
| `HTTP_ADDR` | no | HTTP listen address | Defaults to `:8080`; example: `127.0.0.1:9000` |
| `HTTP_MAX_HEADER_BYTES` | no | Maximum HTTP request header size | Defaults to `16384`; minimum `4096` bytes |
| `HTTP_MAX_BODY_BYTES` | no | Maximum buffered HTTP request body size | Defaults to `65536`; minimum `8192` bytes; oversized bodies receive `413` |
| `HTTP_WRITE_TIMEOUT` | no | Maximum time for writing an HTTP response | Defaults to `10s`; minimum `1s` |
| `HTTP_READ_TIMEOUT` | no | Maximum time for reading an HTTP request | Defaults to `5s`; minimum `1s` |
| `HTTP_READ_HEADER_TIMEOUT` | no | Maximum time for reading HTTP request headers | Defaults to `2s`; minimum `1s` |
| `HTTP_IDLE_TIMEOUT` | no | Maximum keep-alive idle time | Defaults to `30s`; minimum `1s` |
| `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT` | no | Maximum graceful HTTP shutdown time | Defaults to `30s`; minimum `1s` |

## Identity

Identity owns personal and profile information, including contact details,
names, profile attributes, and avatar data or references. Authentication does
not depend on reading Identity.

### Environment Variables

| Name | Required | Purpose | Example / Notes |
| --- | --- | --- | --- |
| `APP_ENV` | no | Application runtime environment | Defaults to `production`; allowed values: `production`, `development` |
| `HTTP_ADDR` | no | HTTP listen address | Defaults to `:8080`; example: `127.0.0.1:9000` |

## Repository Layout

```text
services/iam/       IAM service
services/identity/  Identity service
shared/             Versioned protobuf contracts and generated Go bindings
docs/adr/           Architecture decision records
openspec/           Behavioral specifications and proposed changes
```

## Development

The workspace uses Go 1.26.5, Task 3.52.0, and Docker.

```sh
task build     # Build both services
task test      # Test all workspace modules
task generate  # Regenerate protobuf and gRPC bindings
task verify    # Run all repository checks
task images    # Build both service images
```

### Integration Test Databases

Each supported database has an isolated Docker Compose environment for
integration tests. The containers listen only on the local loopback interface
and use the `userhub_test` database with the `userhub` / `userhub_test`
credentials.

| Database | Compose file | Host port | Start task | Stop task |
| --- | --- | --- | --- | --- |
| PostgreSQL 17 | `docker/integration/compose.postgres.yaml` | `54329` | `task integration-up-postgres` | `task integration-down-postgres` |
| MySQL 8.4 | `docker/integration/compose.mysql.yaml` | `33069` | `task integration-up-mysql` | `task integration-down-mysql` |
| MariaDB 11.4 | `docker/integration/compose.mariadb.yaml` | `33079` | `task integration-up-mariadb` | `task integration-down-mariadb` |

The stop tasks remove database volumes so each integration-test run can begin
with an empty database.
