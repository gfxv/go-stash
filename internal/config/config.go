package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"log/slog"
	"os"
	"time"
)

var (
	configPath  = flag.String("config", "", "path to config file")
	syncNodes   = flag.Bool("sync", false, "allow sync node request")
	announceNew = flag.Bool("announce-new", false, "allow announce new node request")
)

type Config struct {
	Env     string        `yaml:"env" env:"STASH_ENV" env-default:"dev"`
	GRPC    GRPCConfig    `yaml:"grpc"`
	Storage StorageConfig `yaml:"cas"`
}

// TODO: add description for config fields
// see github.com/ilyakaznacheev/cleaner?tab=readme-ov-file#description

type GRPCConfig struct {
	Port                int           `yaml:"port" env:"STASH_PORT" env-default:"5555"`
	Timeout             time.Duration `yaml:"timeout" env:"STASH_TIMEOUT" env-default:"10s"`
	HealthCheckInterval time.Duration `yaml:"health-check-interval" env:"STASH_HEALTH_CHECK_INTERVAL" env-default:"10s""`
	SyncNode            string        `yaml:"sync-node" env:"STASH_SYNC_NODE"`
	Nodes               []string      `yaml:"nodes" env:"STASH_NODES" env-separator:";"`

	AnnounceNewNode bool
}

type StorageConfig struct {
	Path                       string `yaml:"path" env:"STASH_PATH" env-default:"./stash/"`
	CompressionLevel           int    `yaml:"compression-level" env:"STASH_COMPRESSION_LEVEL" env-default:"0"` // TODO: <-- ???
	ReplicationFactor          int    `yaml:"replication-factor" env:"STASH_REPLICATION_FACTOR" env-default:"0"`
	AllowServerSideCompression bool   `yaml:"allow-server-side-compression" env:"STASH_ALLOW_SERVER_SIDE_COMPRESSION" env-default:"false"` // TODO: <-- ???
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
