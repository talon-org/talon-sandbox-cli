# talon-sandbox CLI

CLI for the **talon-sandbox** platform — create, manage, and inspect AI agent sandboxes.

Binary names: `talon-sandbox` (full) and `tsb` (short alias, via symlink or `os.Args[0]` detection).

## Install

```sh
go install x.xgit.pro/dark/talon-sandbox-cli@latest

# Optional symlink for short alias
ln -sf $(which talon-sandbox) /usr/local/bin/tsb
```

## Environment variables

| Variable | Purpose |
|---|---|
| `TALON_SANDBOX_SERVER` | Server URL (overrides `--server` and config context) |
| `TALON_SANDBOX_API_KEY` | API key (overrides config/OS keyring) |
| `TALON_SANDBOX_CONTEXT` | Config context name (overrides `--context`) |
| `TALON_SANDBOX_CONFIG` | Config file path (overrides `--config`) |

Config file: `~/.config/talon-sandbox/config.yaml` (XDG).

## Quick start

```sh
# Store your API key
tsb login --server https://api.talon.example.com --api-key ask_xxxxx

# Create a sandbox and wait for it
tsb create --image talon-alpine --resources cpu=2,memory=4GiB --wait running

# Run a command
tsb run sb-abc123 "echo hello"

# Open an interactive terminal
tsb pty sb-abc123
```

## Command reference

### Auth

```
tsb login [--server URL] [--api-key KEY] [-u USERNAME]
tsb logout
tsb whoami [-o table|json]
```

### Context management

```
tsb context list
tsb context use <name>
tsb context create <name> [--server URL]
tsb context delete <name>
```

### Sandbox lifecycle

```
tsb create [--image IMAGE] [--resources cpu=N,memory=NiB] [--network allowlist|open|sealed]
           [--idle-timeout DURATION] [--ttl DURATION] [--wait running]
           [--spawn CMD] [--expose PORT] [--print-url]
           [-o table|json|id]

tsb list [-o table|json|id]
tsb get <id> [-o table|json|id]
tsb rm <id> [--force]
tsb pause <id>
tsb resume <id>
```

### Process management

```
tsb run <id> <cmd>                  # sync exec; exit code propagated
tsb spawn <id> <cmd>                # async; prints process ID
tsb logs <id> <pid> [--follow] [--tail N]
tsb kill <id> <pid>
```

### Networking

```
tsb expose <id> <port> [--sign] [--ttl DURATION] [--subdomain NAME]
tsb unexpose <id> <port>
tsb exposed <id> [-o table|json]
```

### Terminal

```
tsb pty <id> [--cmd "/bin/bash"]   # raw PTY; SIGWINCH forwarded
```

### File transfer

```
tsb cp <id>:<remote> <local>        # download
tsb cp <local> <id>:<remote>        # upload
```

### Environment variables

```
tsb env get <id> <KEY>
tsb env set <id> KEY=VALUE [KEY2=VALUE2 ...]
tsb env list <id> [-o table|json]
tsb env rm <id> <KEY>
```

### Version

```
tsb version
```

## Notes

- `tsb create --wait running` blocks via `Sandbox.waitForState()` (client-side polling, 500ms interval).
- `tsb create --spawn CMD --expose PORT --print-url` chains: create → wait running → spawn → expose → print URL.
- `tsb run` propagates the remote process exit code (`os.Exit`) for pipeline composition.
- `tsb expose` returns `ErrNotImplemented` if the server endpoint is not yet available — surfaced as a warning, not a hard error.
- `tsb pty` requires a real TTY on stdin; exits with an error otherwise.

## Links

- Spec 49 (v2 brand design)
- Go SDK: `x.xgit.pro/dark/talon-sandbox-sdk-go`
