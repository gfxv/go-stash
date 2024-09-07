package config

import (
	"flag"
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

var (
	syncNodes = flag.Bool("sync", false, "")
)

type Config struct {
	GRPC    GRPCConfig    `yaml:"grpc"`
	Storage StorageConfig `yaml:"cas"`
}

// TODO: add description for config fields
// see github.com/ilyakaznacheev/cleaner?tab=readme-ov-file#description

type GRPCConfig struct {
	Port     int           `yaml:"port" env:"PORT" env-default:"5555"`
	Timeout  time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"10s"`
	SyncNode string        `yaml:"sync-node" env:"SYNC_NODE"`
	Nodes    []string      `yaml:"nodes" env:"NODES" env-separator:";"`
}

type StorageConfig struct {
	Path                       string `yaml:"path" env:"STASH_PATH" env-default:"./stash/"`
	CompressionLevel           int    `yaml:"compression-level" env:"STASH_COMPRESSION_LEVEL" env-default:"0"` // TODO: <-- ???
	AllowServerSideCompression bool   `yaml:"allow-server-side-compression" env:"ALLOW_SERVER_SIDE_COMPRESSION" env-default:"false"`
}

func MustLoad() *Config {
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

	return &cfg
}

func parseConfigPath() string {
	path := ""

	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}
	return path
}
