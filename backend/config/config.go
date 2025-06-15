package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	LLM      LLMConfig      `yaml:"llm"`
	Pipeline PipelineConfig `yaml:"pipeline"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LLMConfig struct {
	Provider string            `yaml:"provider"`
	Endpoint string            `yaml:"endpoint"`
	Model    string            `yaml:"model"`
	Options  map[string]string `yaml:"options"`
}

type PipelineConfig struct {
	ConfigPath string `yaml:"config_path"`
	DataPath   string `yaml:"data_path"`
}

func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port: "8080",
			Host: "0.0.0.0",
		},
		Database: DatabaseConfig{
			Path: "./data/badger",
		},
		LLM: LLMConfig{
			Provider: "openai",
			Endpoint: "http://localhost:11434",
			Model:    "qwen3-30b-a3b-mlx",
		},
		Pipeline: PipelineConfig{
			ConfigPath: "./configs/pipeline.yaml",
			DataPath:   "./data/pipeline",
		},
	}

	// Try to load from config file
	configPath := os.Getenv("CURATOR_CONFIG")
	if configPath == "" {
		configPath = "./configs/curator.yaml"
	}

	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	// Create directories if they don't exist
	os.MkdirAll(filepath.Dir(config.Database.Path), 0755)
	os.MkdirAll(config.Pipeline.DataPath, 0755)
	os.MkdirAll(filepath.Dir(config.Pipeline.ConfigPath), 0755)

	return config, nil
}
