package cmd

import (
	"fmt"
	"time"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

func newAPIClientWithTimeout(timeout int) (*client.ToneCloneClient, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return nil, fmt.Errorf("authentication required: %w", err)
	}
	return client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		time.Duration(timeout)*time.Second,
	), nil
}
