package config

import (
	"flag"
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

type GRPCConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
	Nodes   []string      `yaml:"nodes"`
}

type StorageConfig struct {
	Path             string `yaml:"path"`
	CompressionLevel int    `yaml:"compression-level"`
}

func MustLoad() *Config {
	configPath := parseConfigPath()
	if configPath == "" {
		panic("config path is empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + err.Error())
	}

	return &Config{}
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
