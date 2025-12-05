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
