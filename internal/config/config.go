package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP struct {
		Bind string `yaml:"bind"`
		Port int    `yaml:"port"`
		TLS  struct {
			Enabled bool   `yaml:"enabled"`
			Cert    string `yaml:"cert"`
			Key     string `yaml:"key"`
		} `yaml:"tls"`
	} `yaml:"http"`
	Auth struct {
		JWTPublicKeys []string `yaml:"jwt_public_keys"` // rutas a PEM
		Issuer        string   `yaml:"issuer"`
		Audience      string   `yaml:"audience"`
	} `yaml:"auth"`
	Logging struct {
		Level string `yaml:"level"`
		JSON  bool   `yaml:"json"`
	} `yaml:"logging"`
	ARI struct {
		URL      string `yaml:"url"` // wss://host:8089/ari/events?api_key=user:pass&app=echopbx
		App      string `yaml:"app"`
		APIKey   string `yaml:"api_key"`
		Insecure bool   `yaml:"insecure"`
		Fake     bool   `yaml:"fake_ari"`
	} `yaml:"ari"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.HTTP.Bind == "" {
		c.HTTP.Bind = "0.0.0.0"
	}
	if c.HTTP.Port == 0 {
		c.HTTP.Port = 8080
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	return &c, nil
}
