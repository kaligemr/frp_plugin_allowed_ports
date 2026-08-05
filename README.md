# frps_allowed_ports

[English](https://github.com/kaligemr/frp_plugin_allowed_ports/blob/dev/README_en.md)

frp 服务端插件，用于为 [frp](https://github.com/fatedier/frp) 的特定用户限制可用端口、子域名和自定义域名。

### Fork 历史

本项目历经多次 fork 维护：

| 阶段 | 仓库 | 状态 |
|------|------|------|
| 原始作者 | [Parmicciano/frp_plugin_allowed_ports](https://github.com/Parmicciano/frp_plugin_allowed_ports) | 已停止维护 |
| 第二次维护 | [gainskills/frp_plugin_allowed_ports](https://github.com/gainskills/frp_plugin_allowed_ports) | 已停止维护 |
| **当前维护** | [**kaligemr/frp_plugin_allowed_ports**](https://github.com/kaligemr/frp_plugin_allowed_ports) | **正在维护** |

注意：正在维护不代表持续维护，此仓库随时可能停止维护

### 功能特性

* 通过配置文件对用户进行端口、子域名和自定义域名的校验。
* 新增端口范围支持，如 1-65535、8000-9000。
* 新增 `all` 关键字，允许用户使用任意域名、任意端口、任意协议。
* 新增域名通配符支持：`*.domain.com`（一级子域）、`**.domain.com`（所有层级子域）。
* 新增配置文件`#`注释支持
* stcp 类型不做校验（因为它通过 `sk` 进行认证）。

依赖 [fp-multiuser](https://github.com/gofrp/fp-multiuser)。

### 下载

从 [Release](https://github.com/kaligemr/frp_plugin_allowed_ports/releases) 页面下载预编译二进制文件。

### 要求

frp 版本 >= v0.42.0。

更低版本可能也能运行，但未经过测试。
注意：此内容由原仓库fork而来，没有经过测试。

### 使用方法

支持 `tcp`、`udp`、`http`（含 custom_domains 和 subdomains）类型的校验。

**1. 创建 `ports` 配置文件**，每行一条记录，格式为 `用户名=规则`：

```
user1=65535
user2=80
user2=525
user1=6980

# 子域名前缀（http/https），对应 frps 的 subdomain_host 字段
user2=subdomain
user1=subdomain2

# 自定义域名（精确匹配或通配符）
user1=service.example.com
user6=*.example.com
user7=**.example.org

user3=all
user4=8000-9000
user5=1-65535
```

规则类型说明：

| 规则 | 含义 | 适用类型 |
|------|------|---------|
| `80` | 精确匹配单个端口 | tcp/udp |
| `8000-9000` | 匹配 8000 到 9000 之间的所有端口 | tcp/udp |
| `all` | 等价于 `1-65535`，允许使用任意域名、任意端口、任意协议 | tcp/udp/http/https |
| `subdomain`(示例) | 精确匹配子域名，需同subdomain_host使用 | http/https |
| `service.example.com`(示例) | 精确匹配自定义域名 | http/https |
| `*.example.com` | 单层通配符：匹配 `a.example.com`，但不包括 `example.com` 本身或 `a.b.example.com` | http/https |
| `**.example.org` | 多层通配符：匹配 `a.example.org`、`a.b.c.example.org`，但不匹配 `example.org` 本身 | http/https |

> 同一用户可写多行，每行一条规则，多条规则取并集。
> 以 `#` 开头的行会被视为注释；空白行则会被忽略。

**2. 运行 frps 和插件：**

```bash
./frps -c ./frps.ini
./frp_plugin_allowed_ports -p ./ports
```

**3. 在 frps 配置中注册插件：**

```toml
bindAddr = "0.0.0.0"
bindPort = 7000
vhostHttpPort = 8080

# ports配置文件中subdomain对应的一级域名
# 例：ports中配置了user1=example，frps.toml中配置了subdomain_host = example.com
# 代表允许user1用户使用example.example.com域名
subdomain_host = example.com

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

**4. frpc 客户端配置：**

`user` 字段为必填项。

```toml
# frpc.toml
serverAddr = x.x.x.x
serverPort = 7000
user = "user1"
metadatas.token = "xxxxxx"

[[proxies]]
name = "ssh"
type = tcp
localPort = 22
remotePort = 6000
```

