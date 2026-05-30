# japro-stats

A lightweight stats API and static file server for JKA japro mod servers. Single static binary, no runtime dependencies.

## Features

- Stats API endpoint (POST) with incremental sync via `last_update` timestamp
- Serves static files (demos, etc.) from configurable directories
- gzip compression
- Read-only SQLite access
- Optional TLS

## Query types

| type | Description |
|---|---|
| `races` | Race runs since `last_update` |
| `race_demos` | Unique course/style combos since `last_update` |
| `duels` | Duels since `last_update` |
| `accounts` | Accounts active since `last_update` |
| `teams` | All teams |
| `team_accounts` | All team memberships |

All requests are POST with fields: `username`, `password`, `type`, `last_update`

## Config

Copy `japro-stats.toml.example` to `japro-stats.toml` and edit:

```toml
port    = ":80"
db_path = "/opt/jkaserver/GameData/japro/data.db"
api_path = "/update"

[users]
user1 = "yourpassword"

[[static]]
url_path = "/demos"
fs_path  = "/opt/jkaserver/GameData/japro/demos"
```

## Building

```powershell
# Linux
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -buildvcs=false -o bin/japro-stats ./server

# Windows
go build -buildvcs=false -o bin/japro-stats.exe ./server
```

## Deployment

See [INSTALL.md](INSTALL.md) for full Linux install instructions.
