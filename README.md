# UserHub

UserHub is a reusable implementation of user management for applications. Its
UserHub Service is one independently deployable regional runtime for account,
profile, authentication, and delivery capabilities that are specified and
implemented incrementally.

> [!NOTE]
> UserHub is in early development. Account and authentication behavior remains
> under active specification.

UserHub implements one self-service registration strategy: a numeric one-time
code (OTP) sent to an email address from any provider. The account is created
only after the code is confirmed. A password is optional and may add another
authentication factor, but never replaces the required email OTP. Phone numbers
and external accounts may be linked later, but do not provide separate
registration strategies. A confirmed email remains reserved to its original
account after an email change and cannot be used to register another account.

## Component

| Component         | Responsibility                                                                                                                                   | Documentation                                            |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| UserHub Service   | Regional HTTP runtime, PostgreSQL lifecycle, and foundation for user-management capabilities                                                     | [Service README](services/userhub/README.md)             |
| UserHub Directory | Optional, project-owned service for multi-country installations; coordinates global login ownership and discovers trusted regional UserHub cells | [Directory README](services/userhub-directory/README.md) |

One UserHub Service deployment is complete for a single region and requires no
global component. A project operating multiple regional cells across countries
can add the optional project-owned UserHub Directory. Different projects
continue to own independent installations, namespaces, secrets, and trust.

## Repository Layout

```text
services/userhub/            Regional UserHub Service and documentation
services/userhub-directory/  Optional multi-country UserHub Directory and documentation
```

Future inter-service protobuf contracts belong to the service that provides
them. Account domain models, configuration, and general utilities remain inside
UserHub Service.

## Development

The workspace uses Go 1.27.0, Task 3.52.0, and Docker. Run repository-wide
commands from the repository root:

| Command       | Description                                                             |
| ------------- | ----------------------------------------------------------------------- |
| `task build`  | Build UserHub Service and UserHub Directory                             |
| `task test`   | Test all workspace modules                                              |
| `task verify` | Check tests, module metadata, vet, workspace resolution, and boundaries |
| `task images` | Build UserHub Service and UserHub Directory images                      |

Service setup, configuration, migrations, and deployment procedures are
documented in the [UserHub Service README](services/userhub/README.md) and
[UserHub Directory README](services/userhub-directory/README.md).
