# Runp

![CI Linux Mac](https://github.com/enr/runp/workflows/CI%20Linux%20Mac/badge.svg)
![CI Windows](https://github.com/enr/runp/workflows/CI%20Windows/badge.svg)
[![Documentation](https://img.shields.io/badge/Website-Documentation-orange)](https://enr.github.io/runp/)
[![Download](https://img.shields.io/badge/Download-Last%20release-brightgreen)](https://github.com/enr/runp/releases/latest)

Runp is a process orchestration tool that extends beyond container management. 
In addition to running containers, Runp orchestrates host processes and SSH tunnels, enabling comprehensive development environment setup and management.

Designed to streamline the complete setup of development environments with a unified configuration approach.

## Usage

Define your system configuration in a Runpfile:

```yaml
name: Example
description: |
  Sample Runpfile to show runp functionalities
units:
  web:
    description: Web app
    # this process is running on host machine
    host:
      command: node app.js
      workdir: backend
      env:
        # inherit PATH from host system to find needed tools (e.g. node)
        PATH: $PATH
      await:
        # wait for the DB being available
        resource: tcp4://localhost:5432/
        timeout: 0h0m10s
  mail:
    description: Test mail server
    # this process is running in a container
    container:
      image: docker.io/mailhog/mailhog
      ports:
        - "8025:8025"
        - "1025:1025"
  db:
    description: Corporate DB
    # This process is reachable through SSH port forwarding
    ssh_tunnel:
      user: user
      auth:
        identity_file: ~/.ssh/id_rsa
      local:
        port: 5432
      jump:
        host: dev.host
        port: 22
      target:
        host: corporate.db
        port: 5432
```

Execute Runp:

```
runp up -f /path/to/Runpfile
```

For additional examples, see the [examples directory](examples/).  
For comprehensive documentation, visit the [official documentation](https://enr.github.io/runp/).


## Composing Runpfiles with `include:`

Large projects can split their configuration across multiple files and compose them with the `include:` key.
Each entry is a path **relative to the directory of the file that declares it**.
All units from included files are merged into the root Runpfile before startup.

```yaml
# Runpfile  (project root)
name: My Project
include:
  - infra/Runpfile      # defines units: db, redis
  - services/Runpfile   # defines units: api, worker
units:
  proxy:
    host:
      command: nginx -g "daemon off;"
```

```yaml
# infra/Runpfile
units:
  db:
    container:
      image: postgres:15
      ports: ["5432:5432"]
  redis:
    container:
      image: redis:7
      ports: ["6379:6379"]
```

After loading, `runp up` starts all five units (`db`, `redis`, `api`, `worker`, `proxy`) as if they were defined in a single file.

**Duplicate unit names** across any two files are an error — loading fails immediately with:

```
duplicate unit identifier: <name>
```

Rename the conflicting unit in one of the files before proceeding.

**Circular includes** are detected and reported with the full import chain, for example:

```
circular dependency detected: a.yml → b.yml → c.yml → b.yml
```

Includes can be nested to any depth as long as there are no cycles and no duplicate unit names.


## Podman Compatibility

### `volumes_from` on rootless Podman

When running Podman in rootless mode, `--volumes-from` can fail with a "container not found" error if the source container has not finished its startup sequence before the dependent container starts. This is a timing issue specific to rootless Podman; Docker is not affected.

**Workaround:** add an `await:` condition to the source unit so that runp waits for it to be ready before starting any dependent container.

```yaml
units:
  data:
    container:
      image: myapp/data:latest
      await:
        resource: tcp4://localhost:8080/
        timeout: 0h0m30s

  app:
    depends_on:
      - data
    container:
      image: myapp/app:latest
      volumes_from:
        - data   # resolved to runp-data at runtime
```

With `await:` on the `data` unit, runp blocks until the container is accepting connections before launching `app`, eliminating the race condition.

If the source container does not expose a TCP port you can await on, use a short fixed timeout:

```yaml
  data:
    container:
      image: myapp/data:latest
      await:
        resource: tcp4://localhost:9999/
        timeout: 0h0m5s
```

The 5-second wait is enough for Podman's rootless networking to register the container before `--volumes-from` is resolved.

## Development

Clone or download the repository.

Build the project (binaries will be created in `bin/`):

```
./.sdlc/build
```

or

```
.sdlc\build.cmd
```

Run code quality checks and tests:

```
./.sdlc/check
```

or

```
.sdlc\check.cmd
```


## License

Apache 2.0 - see LICENSE file.

Copyright 2020-TODAY runp contributors
