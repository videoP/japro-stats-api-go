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
| `check_password` | Verify a player's in-game password |
| `cosmetics` | All cosmetic unlock definitions from `cosmetics.cfg` |

All requests are POST with fields: `username`, `password`, `type`, `last_update`

### `check_password`

Additional POST fields: `player` (player name), `player_password` (their in-game password).

Returns `[[true]]` if the password matches, `[[false]]` if it doesn't or the player doesn't exist.

```
POST /update
username=apiuser&password=apipass&type=check_password&player=SomePlayer&player_password=theirpass
```

Response:
```json
[[true]]
```

### `cosmetics`

No additional fields. Reads `cosmetics_path` from config and returns all entries as an array of rows.

Each row: `[bit, coursename, style, ms_requirement]`

- `bit` — cosmetic unlock bit (integer)
- `coursename` — course name string
- `style` — style integer
- `ms_requirement` — time requirement in milliseconds (0 = no requirement)

```
POST /update
username=apiuser&password=apipass&type=cosmetics
```

Response:
```json
[[0,"racepack9 (snowboard)",1,60000],[1,"racepack7 (jump_green_pro)",1,0],...]
```

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

## Using alongside Apache / nginx

If you already have a web server handling static files, just remove or comment out all `[[static]]` blocks in the toml. japro-stats will run API-only on whatever port you configure, and your existing web server continues serving files as normal. Run japro-stats on a non-conflicting port (e.g. `:8080`) and proxy or point your client directly at it.

## Deployment

See [INSTALL.md](INSTALL.md) for full Linux install instructions.
