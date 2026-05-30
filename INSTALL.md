# japro-stats — Install Guide (Linux)

## Files

| File | Purpose |
|---|---|
| `bin/japro-stats` | Linux binary (deploy this) |
| `japro-stats.toml.example` | Copy to `japro-stats.toml` and edit |

---

## 1. Upload files to the server

```bash
scp bin/japro-stats japro-stats.toml user@yourserver:/opt/japro-stats/
```

---

## 2. Set ownership and permissions

As root on the server:

```bash
chown user:user /opt/japro-stats/japro-stats /opt/japro-stats/japro-stats.toml
chmod 744 /opt/japro-stats/japro-stats
chmod 600 /opt/japro-stats/japro-stats.toml
```

If binding to port 80, grant the binary permission to use low ports (run once as root, repeat after every binary update):

```bash
setcap 'cap_net_bind_service=+ep' /opt/japro-stats/japro-stats
```

---

## 3. Edit config

```bash
nano /opt/japro-stats/japro-stats.toml
```

Set `db_path`, `port`, `api_path`, users, and any static directories.

---

## 4. Test run

```bash
/opt/japro-stats/japro-stats /opt/japro-stats/japro-stats.toml
```

Should print `listening on :80` (or your configured port). `Ctrl+C` to stop.

---

## 5. Autostart via crontab

```bash
crontab -e
```

Add:

```
@reboot /opt/japro-stats/japro-stats /opt/japro-stats/japro-stats.toml
```

---

## 6. Updating

Replace the binary and re-run setcap:

```bash
scp bin/japro-stats user@yourserver:/opt/japro-stats/japro-stats
```

Then as root on the server:

```bash
chown user:user /opt/japro-stats/japro-stats
chmod 744 /opt/japro-stats/japro-stats
setcap 'cap_net_bind_service=+ep' /opt/japro-stats/japro-stats
```

Then restart the process (kill and let crontab bring it back on next reboot, or start it manually).

---

## Troubleshooting

### `Permission denied` when running the binary

The binary isn't executable. Fix it:

```bash
chmod 744 /opt/japro-stats/japro-stats
```

### `bind: permission denied` on port 80

Non-root processes can't bind to ports below 1024 by default. Run as root:

```bash
setcap 'cap_net_bind_service=+ep' /opt/japro-stats/japro-stats
```

This must be repeated every time the binary is replaced.

### Connection times out / can't reach the server

Check your firewall:

```bash
iptables -L -n | grep <port>
```

To open a port:

```bash
iptables -I INPUT -p tcp --dport 80 -j ACCEPT
```

### `config: open japro-stats.toml: no such file or directory`

Always pass the full path to the config — crontab and other launchers don't run from the install directory:

```bash
/opt/japro-stats/japro-stats /opt/japro-stats/japro-stats.toml
```

### Binary works but DB errors on startup

Check `db_path` in your toml points to the correct absolute path of `data.db`. Verify it exists:

```bash
ls -la /path/to/data.db
```
