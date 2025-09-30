package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Env        string        `yaml:"env"`
	Storage    StorageConfig `yaml:"storage"`
	HTTPServer `yaml:"http_server"`
}

type StorageConfig struct {
	Type     string         `yaml:"type"`
	SQLite   SQLiteConfig   `yaml:"sqlite"`
	Postgres PostgresConfig `yaml:"postgres"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type HTTPServer struct {
	Address     string        `yaml:"address"`
	User        string        `yaml:"user"`
	Password    string        `yaml:"password"`
	Timeout     time.Duration `yaml:"timeout"`
	IdleTimeout time.Duration `yaml:"idle_timeout"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/docker.yaml"
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	yamlFile, err := os.ReadFile(configPath)
	if err != nil {
		panic("failed to read config file: " + err.Error())
	}

	var cfg Config

	if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
		panic("failed to parse config: " + err.Error())
	}

	return &cfg
}
