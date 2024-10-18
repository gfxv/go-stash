package config

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

var (
	configPath  = flag.String("config", "", "path to config file")
	syncNodes   = flag.Bool("sync", false, "allow sync node request")
	announceNew = flag.Bool("announce-new", false, "allow announce new node request")
)

// Config holds the configuration settings for whole application.
// Including settings for env, grpc service and storage service.
// This struct is used to parse settings directly from yml file.
type Config struct {
	// Env defines environment in which app will run.
	// Acceptable values: prod, dev.
	Env string `yaml:"env" env:"STASH_ENV" env-default:"dev"`

	// GRPC configuration of grpc service.
	// See GRPCConfig for more details.
	GRPC GRPCConfig `yaml:"grpc"`

	// Storage configuration of storage service.
	// See StorageConfig for more details.
	Storage StorageConfig `yaml:"cas"`
}

// TODO: add description for config fields
// see github.com/ilyakaznacheev/cleanenv

// GRPCConfig holds the configuration settings for the gRPC server.
//
// The configuration can be populated from YAML file or environment variables
type GRPCConfig struct {
	// Port specifies the port on which the gRPC server will listen for incoming connections.
	// Default value is 5555.
	// Can be set through the `STASH_PORT` environment variable.
	Port int `yaml:"port" env:"STASH_PORT" env-default:"5555"`

	// Timeout defines the duration before a request is considered to have timed out.
	// The default value is 10 seconds
	// Can be set via the `STASH_TIMEOUT` environment variable.
	Timeout time.Duration `yaml:"timeout" env:"STASH_TIMEOUT" env-default:"10s"`

	// HealthCheckInterval sets the interval for health check pings
	// to be sent to the nodes in the system. This helps ensure that nodes
	// are responsive and can handle requests.
	// The default interval is 10 seconds
	// Can be set via the `STASH_HEALTH_CHECK_INTERVAL` environment variable.
	HealthCheckInterval time.Duration `yaml:"health-check-interval" env:"STASH_HEALTH_CHECK_INTERVAL" env-default:"10s""`

	// SyncNode identifies the specific node that should be synchronized with.
	// This field can be set through the `STASH_SYNC_NODE` environment variable
	// and may be left empty if synchronization is not needed.
	SyncNode string `yaml:"sync-node" env:"STASH_SYNC_NODE"`

	// Nodes is a list of nodes that the gRPC server can communicate with.
	// This field can be populated with multiple values passed to env. separated by a semicolon (`;`)
	Nodes []string `yaml:"nodes" env:"STASH_NODES" env-separator:";"`

	// AnnounceNewNode is a boolean flag that indicates whether the server should
	// announce the addition of a new node to the system.
	// If set to true, the server will broadcast the new node's presence to other nodes in the network.
	AnnounceNewNode bool
}

// StorageConfig holds the configuration settings for the storage system.
//
// This struct provides options for configuring the storage path, compression,
// and replication settings. Configuration values can be set through YAML file or environment variables
type StorageConfig struct {
	// Path specifies the directory path where storage data will be located.
	// The default path is `./stash/`. This can be configured using the `STASH_PATH` environment variable.
	Path string `yaml:"path" env:"STASH_PATH" env-default:"./stash/"`

	// CompressionLevel Defines the level of compression to be applied to the stored data.
	//
	// NOTE: Different levels of compression are not implemented yet
	//
	// The default is `0`, and this can be set via the `STASH_COMPRESSION_LEVEL` environment variable.
	CompressionLevel int `yaml:"compression-level" env:"STASH_COMPRESSION_LEVEL" env-default:"0"`

	// ReplicationFactor indicates the number of replicas for each piece of stored data.
	// A replication factor of `0` implies no replication, while higher values
	// provide redundancy for improved reliability and availability.
	// The default value is `0`
	// Can be set using the `STASH_REPLICATION_FACTOR` environment variable.
	ReplicationFactor int `yaml:"replication-factor" env:"STASH_REPLICATION_FACTOR" env-default:"0"`

	// AllowServerSideCompression is a boolean flag that determines whether
	// server-side compression is permitted. If set to true, the server will
	// apply compression when storing data, potentially saving disk space at
	// the cost of increased CPU usage during read/write operations.
	// The default is `false`
	// Can be configured using the `STASH_ALLOW_SERVER_SIDE_COMPRESSION` environment variable.
	AllowServerSideCompression bool `yaml:"allow-server-side-compression" env:"STASH_ALLOW_SERVER_SIDE_COMPRESSION" env-default:"false"` // TODO: <-- ???
}

func MustLoad() *Config {
	flag.Parse()

	configPath := parseConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + err.Error())
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("can not read config file: " + err.Error())
	}

	cfg.GRPC.AnnounceNewNode = *announceNew

	return &cfg
}

func parseConfigPath() string {
	if *configPath == "" {
		*configPath = os.Getenv("CONFIG_PATH")
	}
	return *configPath
}

func (c *Config) Validate(logger *slog.Logger) {
	if len(c.GRPC.Nodes) == 0 && c.GRPC.SyncNode == "" {
		logger.Warn("Nodes and SyncNode are not configured")
	}
}
