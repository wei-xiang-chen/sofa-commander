package application

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sofa-commander/backend/internal/features/config/domain"
)

// ConfigService defines the interface for config management.
type ConfigService interface {
	SaveConfig(config *domain.AppConfig) error
}

// configService is the implementation of ConfigService.
type configService struct {
	// In the future, this might have a repository dependency.
}

// NewConfigService creates a new instance of configService.
func NewConfigService() ConfigService {
	return &configService{}
}

// SaveConfig saves the application configuration to frontend/public/config.json.
func (s *configService) SaveConfig(config *domain.AppConfig) error {
	// Determine the absolute path to the frontend/public directory
	// This assumes the backend is run from the sofa-commander/backend directory
	// and frontend/public is at ../frontend/public relative to the backend.
	publicPath := filepath.Join("..", "frontend", "public", "config.json")

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = ioutil.WriteFile(publicPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config to file %s: %w", publicPath, err)
	}

	return nil
}
