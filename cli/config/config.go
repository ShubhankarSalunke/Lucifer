package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Token     string `json:"token"`
	ServerURL string `json:"base_url"`
	AWS       struct {
		AccessKey  string `json:"access_key"`
		SecretKey  string `json:"secret_key"`
		Region     string `json:"region"`
		RoleArn    string `json:"role_arn"`
		ExternalId string `json:"external_id"`
	} `json:"aws_config"`
}

func getConfigPath() string {
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".lucifer", "cli", "config.json")
	}
	curr, _ := os.Getwd()
	return filepath.Join(curr, ".lucifer", "cli", "config.json")
}


func LoadConfig() Config {
	path := getConfigPath()
	file, err := os.ReadFile(path)
	if err != nil {
		// Migration Fallback: Try the old path (~/.lucifer/config.json)
		home, _ := os.UserHomeDir()
		oldPath := filepath.Join(home, ".lucifer", "config.json")
		file, err = os.ReadFile(oldPath)
		if err != nil {
			return Config{}
		}
	}

	var cfg Config
	json.Unmarshal(file, &cfg)

	return cfg
}


func SaveConfig(cfg Config) error {

	path := getConfigPath()

	// ensure directory exists
	os.MkdirAll(filepath.Dir(path), os.ModePerm)

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func GetServerURL() string {
	envURL := os.Getenv("CHAOS_SERVER_URL")
	if envURL != "" {
		return envURL
	}

	cfg := LoadConfig()
	if cfg.ServerURL != "" {
		return cfg.ServerURL
	}
	cfg.ServerURL = "http://localhost:8000"
	// cfg.ServerURL = "https://kzvijk5asj.execute-api.us-east-1.amazonaws.com/"
	return cfg.ServerURL

}



func GetToken() string {
	envToken := os.Getenv("CHAOS_TOKEN")
	if envToken != "" {
		return envToken
	}
	cfg := LoadConfig()
	return cfg.Token
}
