---
title: "Deployment"
permalink: "/docs/deployment/"
weight: 10
---

Ways to run Agentary in production or on a server.

## Binary (GitHub Releases)

1. Download the archive for your OS/arch from [GitHub Releases](https://github.com/ankittk/agentary/releases) (e.g. `agentary_1.0.0_linux_amd64.tar.gz`).
2. Extract and place the `agentary` binary in your PATH (e.g. `/usr/local/bin`).
3. Run:
   ```bash
   agentary start --foreground
   ```
   Or run as a daemon (no `--foreground`) so it backgrounds and writes a PID file under the home directory.

**Environment:**

| Variable | Description |
|----------|-------------|
| `AGENTARY_HOME` | Data directory (default: `~/.agentary`). |
| `DATABASE_URL` | Optional; for PostgreSQL (e.g. `postgres://user:pass@host/db`). |

**Flags:** Use `--port` to change the HTTP port (default 3548), `--db-driver=postgres` and `--db-url` (or `DATABASE_URL`) for Postgres. See [Configuration]({{< ref "docs/configuration" >}}).

---

## Docker

Images are published to GitHub Container Registry:

```bash
docker pull ghcr.io/ankittk/agentary:latest
docker run -d -p 3548:3548 -v agentary-data:/data ghcr.io/ankittk/agentary:latest
```

Use a specific version tag (e.g. `v1.0.0`) for production. The image runs `agentary start --foreground` with `--home /data`. The volume mount **must** be `/data` (or set `AGENTARY_HOME` inside the container to match your volume path).

---

## Summary

| Method | Use case |
|--------|----------|
| Binary + `--foreground` | Quick run or scripting. |
| Binary (daemon) | No systemd; PID file under `AGENTARY_HOME`. |
| Docker | Isolated run with a single volume for data. |
