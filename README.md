# UserHub

UserHub is a reusable implementation of user management for applications. It
is composed of two independently deployable Go services connected by an
opaque, stable subject identifier.

> [!NOTE]
> UserHub is in early development. Service behavior and runtime configuration
> are still being specified.

## Services

| Service | Responsibility | Documentation |
| ------- | -------------- | ------------- |
| IAM | Accounts, login identifiers, credentials, authentication, general authorization, sessions, JWT issuance, and security audit | [IAM README](services/iam/README.md) |
| Identity | Personal and profile information, including contact details, names, profile attributes, and avatar data or references | [Identity README](services/identity/README.md) |

IAM creates the Identity record during registration, but authentication does
not depend on reading Identity. The services correlate records through an
opaque, stable subject identifier.

## Repository Layout

```text
services/iam/       IAM service and documentation
services/identity/  Identity service and documentation
shared/             Versioned protobuf contracts and generated Go bindings
docs/adr/           Architecture decision records
openspec/           Behavioral specifications and proposed changes
```

See the accepted [architecture decisions](docs/adr/README.md) for the rationale
behind the service boundaries and deployment model.

## Development

The workspace uses Go 1.27.0, Task 3.52.0, and Docker. Run repository-wide
commands from the repository root:

| Command | Description |
| ------- | ----------- |
| `task build` | Build both services |
| `task test` | Test all workspace modules |
| `task generate` | Regenerate protobuf and gRPC bindings |
| `task verify` | Run all repository checks |
| `task images` | Build both service images |

Service-specific setup, configuration, deployment, and commands are documented
in the [IAM](services/iam/README.md) and
[Identity](services/identity/README.md) READMEs.
