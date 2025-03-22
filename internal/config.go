package internal

import (
	"encoding/json"
	"log/slog"
	"os"
)

type Discord struct {
	Token   string `json:"token"`
	GuildID string `json:"guild"`
}

type EmbedMessage struct {
	ChannelId string
	MessageId string
}

type Config struct {
	Discord      *Discord      `json:"discord"`
	EmbedMessage *EmbedMessage `json:"embed_message"`

	path string
}

func (c *Config) Save() error {
	config, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, config, 0655)
}

func NewConfig(path string, logger *slog.Logger) (*Config, error) {
	config, err := readConfig(path, logger)
	if err != nil {
		return config, err
	}

	return config, config.Save()
}

func readConfig(path string, logger *slog.Logger) (*Config, error) {
	var config Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Info("create-config")
		config = Config{}
	} else {
		logger.Info("read-existing-config")
		c, err := os.ReadFile(path)
		if err != nil {
			return &Config{}, err
		}
		err = json.Unmarshal(c, &config)
		if err != nil {
			return &Config{}, err
		}
	}
	config.path = path
	return &config, nil
}
