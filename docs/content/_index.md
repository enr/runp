---
title: "runp"
---

Describe your system in a `Runpfile` and start it as a whole: host processes,
containers and SSH tunnels run in parallel, each with its own working directory,
environment and startup conditions.

```yaml
name: Example
units:
  web:
    host:
      command: node app.js
      workdir: backend
      await:
        resource: tcp4://localhost:5432/
  mail:
    container:
      image: docker.io/mailhog/mailhog
      ports:
        - "8025:8025"
```

```bash
# Start all units in ./Runpfile
runp up

# List units and validate the Runpfile
runp ls
runp validate
```

Then head to the [documentation](docs/) for the Runpfile format and the full
command reference.
