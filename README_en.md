# frps_allowed_ports

[中文](https://github.com/kaligemr/frp_plugin_allowed_ports/blob/dev/README.md)

A frp server plugin to restrict allowed ports, subdomains and custom domains for specific users of [frp](https://github.com/fatedier/frp).

### Fork History

This project has been maintained through several forks:

| Stage | Repository | Status |
|-------|-----------|--------|
| Original author | [Parmicciano/frp_plugin_allowed_ports](https://github.com/Parmicciano/frp_plugin_allowed_ports) | No longer maintained |
| Second maintainer | [gainskills/frp_plugin_allowed_ports](https://github.com/gainskills/frp_plugin_allowed_ports) | No longer maintained |
| **Current maintainer** | [**kaligemr/frp_plugin_allowed_ports**](https://github.com/kaligemr/frp_plugin_allowed_ports) | **Active** |

Note: "Active" does not mean continuously maintained; this repository may stop being maintained at any time.

### Features

* Validate ports, subdomains and custom domains per user via a config file.
* Port range support, e.g. `1-65535`, `8000-9000`.
* `all` keyword — allows any domain, any port, and any protocol.
* Domain wildcard support: `*.domain.com` (single-level subdomain) and `**.domain.com` (any level of subdomain).
* `#` comments in the config file.
* Skips validation for stcp type as it authenticates via `sk`.

Depends on [fp-multiuser](https://github.com/gofrp/fp-multiuser).

### Download

Download prebuilt binaries from the [Release](https://github.com/kaligemr/frp_plugin_allowed_ports/releases) page.

### Requirements

frp version >= v0.42.0.

The plugin may work with older versions but has not been tested.
Note: this project is forked from the original repository and has not been fully tested.

### Usage

Works with `tcp`, `udp`, and `http` (including custom_domains and subdomains) proxy types.

**1. Create a `ports` config file** with one rule per line in the format `username=rule`:

```
user1=65535
user2=80
user2=525
user1=6980

# Subdomain prefixes (http/https) — correspond to frps's subdomain_host setting
user2=subdomain
user1=subdomain2

# Custom domains (exact match or wildcard)
user1=service.example.com
user6=*.example.com
user7=**.example.org

# Port ranges and all
user3=all
user4=8000-9000
user5=1-65535
```

Rule types:

| Rule | Meaning | Applies to |
|------|---------|------------|
| `80` | Exact match for a single port | tcp/udp |
| `8000-9000` | Matches any port from 8000 to 9000 inclusive | tcp/udp |
| `all` | Equivalent to `1-65535`; allows any domain, any port, and any protocol | tcp/udp/http/https |
| `subdomain` (example) | Exact match for a subdomain prefix; used together with `subdomain_host` | http/https |
| `service.example.com` (example) | Exact match for a custom domain | http/https |
| `*.example.com` | Single-level wildcard: matches `a.example.com`, but not `example.com` itself nor `a.b.example.com` | http/https |
| `**.example.org` | Multi-level wildcard: matches `a.example.org`, `a.b.c.example.org`, but not `example.org` itself | http/https |

> A user can have multiple lines; all rules are combined (OR logic).
> Lines starting with `#` are treated as comments; blank lines are ignored.

**2. Run frps and the plugin:**

```bash
./frps -c ./frps.toml
./frp_plugin_allowed_ports -p ./ports
```

**3. Register the plugin in frps config:**

```toml
bindAddr = "0.0.0.0"
bindPort = 7000
vhostHttpPort = 8080

# The base domain used to resolve subdomain prefixes in the ports file.
# Example: if the ports file has "user1=example" and frps.toml sets
# subdomain_host = "example.com", then user1 is allowed to use example.example.com.
subdomain_host = "example.com"

[[httpPlugins]]
name = "multiuser"
addr = "127.0.0.1:7200"
path = "/handler"
ops = ["Login"]

[[httpPlugins]]
name = "allowed-ports"
addr = "127.0.0.1:7201"
path = "/handler"
ops = ["NewProxy"]
```

**4. frpc client config:**

The `user` field is required.

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
user = "user1"
metadatas.token = "xxxxxx"

[[proxies]]
name = "ssh"
type = "tcp"
localPort = 22
remotePort = 6000
```
