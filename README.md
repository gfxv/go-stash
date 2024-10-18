# Stash

**Stash** is a Distributed Content Addressable Storage (dCAS) system which features replication, automatic data re-distribution between nodes, built-in health control and which can run both locally and in cloud.

## Building

To build **Stash** from source:

1) If not installed, install **go 1.22 or newer** on your machine. You can get **go** from [the official website](https://go.dev/doc/install).
2) If not installed, install **Taskfile**. You can get **Taskfile** from [the official website](https://taskfile.dev/installation/).
3) Clone this repo: `git clone https://github.com/gfxv/go-stash.git`
4) In the `go-stash` directory run `task build`, which will result in a binary file named `stash` in `bin` directory.

## Usage

> **When using Stash it's highly recommended to use a VPN and to not expose Stash to the Internet.**

### Configuration

You can configure **Stash** both using the config file or by using environment variables. **Please note that environment variables override values specified in the config.**

The process works as follows: it parses the YML file, checks environment variables to override file values, and if neither is found, it uses the default value.

#### Sample Config File

```yml
env: "prod"
grpc:
  port: 5555
  timeout: "10s"
  health-check-interval: "10s"
  sync-node: "192.168.100.2:5656"
  nodes:
    - "192.168.100.5:5555"
    - "192.168.100.6:5555"
    - "192.168.100.7:5555"
cas:
  path: "/srv/data/stash"
  replication-factor: 0
  allow-server-side-compression: false
  compression-level: 1
```

#### Config Values

| Config | Environment | Default | Description |
| ------ | ----------- | ------- | ----------- |
| `env` | `STASH_ENV` | `dev` | Accepts `prod` or `dev`. Defines environment in which app will run. |
| `port` | `STASH_PORT` | `5555` | Defines the port where Stash will listen for connections. |
| `timeout` | `STASH_TIMEOUT` | `10s` | Defines the duration before a request is considered timed out. |
| `health-check-interval` | `STASH_HEALTH_CHECK_INTERVAL` | `10s` | Sets the interval for health check pings to be sent to the nodes in the system. |
| `sync-node` | `STASH_SYNC_NODE` | Empty | Defines a specific node to synchronize (retrieve addresses of other nodes connected to it) with. **Optional if `nodes` list is specified.** |
| `nodes` | `STASH_NODES` | Empty | List of nodes that the server can communicate with. When supplied via environment, the list is separated with semicolons (`0.0.0.0:5555;1.1.1.1:5555`). **Optional if `sync-node` is specified.** |
| `path` | `STASH_PATH` | `./stash/` | Path to a directory in which stored data will be located. |
| `replication-factor` | `STASH_REPLICATION_FACTOR` | `0` | Defines the replication factor (how much copies of the data to make) for Stash. `0` results in 1 copy (no replication), `1` results in 2 copies, etc.. |
| `allow-server-side-compression` | `STASH_ALLOW_SERVER_SIDE_COMPRESSION` | `false` | Accepts `true` or `false`. This flag determines whether server-side compression is permitted. |
| `compression-level` | `STASH_COMPRESSION_LEVEL` | `0` | Defines the level of compression to be applied to the stored data (up to `4`). |

#### Notes

- Using server-side compression comes with increased CPU usage and increased amount of read/write operations. Please note that with high load this can significantly harm performance.
- When creating a client to be used with **Stash**, implementing some form of compression before sending data to the storage is advisable to reduce disk space use without using server-side compression.

### Running

To run Stash, run the program as follows:

`./stash --config=./config.yml --sync=192.168.1.2:5555 --announce-new=true`

Several launch parameters are available:

| Parameter | Description |
| --------- | ----------- |
| `--config` | Specifies the path to a config file. Alternatively, path to the config can be specified in variable `STASH_CONFIG_PATH`. Useful if a single node is deployed in several configurations. |
| `--sync` | _Optional_. Defines a specific node to synchronize (retrieve addresses of other nodes connected to it) with, similarly to `sync-node` in config. |
| `--announce-new` | _Optional, `false` by default_. Accepts `true` or `false`. Defines whether or not the node should announce itself to other nodes to be added into their node lists.

**Please note:** Launch arguments override both config and environment variables.

### API (gRPC)

For more information, please see `./proto/stash.proto`.

## Planned Features
- [ ] Implementing authorization
- [ ] Making storing, modifying and deleting transactional
- [ ] Minimal CLI Client
- [ ] Support adding names to nodes
- [ ] Making nodes added via API persistent after a graceful shutdown
