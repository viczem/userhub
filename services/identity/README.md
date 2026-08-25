# UserHub Identity

Identity is the profile-focused UserHub service. It owns personal and profile
information, including contact details, names, profile attributes, and avatar
data or references. Authentication does not depend on reading Identity.

> [!NOTE]
> UserHub is in early development. The current HTTP service exposes a liveness
> endpoint while profile behavior is being specified.

## Requirements

- Go 1.27.0
- Task 3.52.0
- Docker, when building the service image

Run the commands below from the repository root unless noted otherwise.

## Local Development

Start the service from its module directory:

```sh
cd services/identity
go run ./cmd start
```

The service listens on `:8080` by default.

### Health Endpoints

| Endpoint | Purpose |
| -------- | ------- |
| `GET /health/live` | Process liveness |

## Development Commands

| Command | Description |
| ------- | ----------- |
| `task identity:build` | Build all Identity packages |
| `task identity:test` | Run Identity tests |
| `task identity:verify` | Run tests, module checks, and `go vet` |
| `task identity:image` | Build the `userhub/identity:dev` container image |

## Configuration

Configuration is read from environment variables.

| Name                             | Required | Default      | Purpose / Constraints                                      |
| -------------------------------- | -------- | ------------ | ---------------------------------------------------------- |
| `APP_ENV`                        | no       | `production` | Application environment; `production` or `development`     |
| `HTTP_ADDR`                      | no       | `:8080`      | HTTP listen address; for example, `127.0.0.1:9000`          |
| `HTTP_MAX_HEADER_BYTES`          | no       | `16384`      | Maximum request header size; at least `4096` bytes          |
| `HTTP_MAX_BODY_BYTES`            | no       | `65536`      | Maximum buffered request body size; at least `8192` bytes   |
| `HTTP_WRITE_TIMEOUT`             | no       | `10s`        | HTTP response write timeout; at least `1s`                  |
| `HTTP_READ_TIMEOUT`              | no       | `5s`         | HTTP request read timeout; at least `1s`                    |
| `HTTP_READ_HEADER_TIMEOUT`       | no       | `2s`         | HTTP header read timeout; at least `1s`                     |
| `HTTP_IDLE_TIMEOUT`              | no       | `30s`        | Keep-alive idle timeout; at least `1s`                      |
| `HTTP_GRACEFUL_SHUTDOWN_TIMEOUT` | no       | `30s`        | Graceful shutdown timeout; at least `1s`                    |

See the repository-level [`README.md`](../../README.md) for project context and
service boundaries.
