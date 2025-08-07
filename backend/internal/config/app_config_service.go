package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sofa-commander/backend/internal/features/config/domain"
)

// AppConfigService defines the interface for application configuration management.
type AppConfigService interface {
	LoadAppConfig() (*domain.AppConfig, error)
	SaveAppConfig(config *domain.AppConfig) error
}

// appConfigService is the implementation of AppConfigService.
type appConfigService struct {
	configPath string
}

// NewAppConfigService creates a new instance of appConfigService.
func NewAppConfigService(configPath string) AppConfigService {
	return &appConfigService{configPath: configPath}
}

// LoadAppConfig loads the application configuration from the configured JSON file.
func (s *appConfigService) LoadAppConfig() (*domain.AppConfig, error) {
	fmt.Println("[DEBUG] LoadAppConfig called, configPath:", s.configPath)
	absPath, err := filepath.Abs(s.configPath)
	fmt.Println("[DEBUG] Absolute config path:", absPath, "err:", err)
	if err != nil {
		fmt.Println("[ERROR] Failed to get absolute path:", err)
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", s.configPath, err)
	}

	data, err := ioutil.ReadFile(absPath)
	fmt.Println("[DEBUG] ReadFile result, bytes:", len(data), "err:", err)
	if err != nil {
		fmt.Println("[ERROR] Failed to read app config file:", err)
		return nil, fmt.Errorf("failed to read app config file %s: %w", absPath, err)
	}

	var appConfig domain.AppConfig
	err = json.Unmarshal(data, &appConfig)
	fmt.Println("[DEBUG] Unmarshal result, err:", err)
	if err != nil {
		fmt.Println("[ERROR] Failed to unmarshal app config:", err)
		return nil, fmt.Errorf("failed to unmarshal app config from %s: %w", absPath, err)
	}

	fmt.Println("[DEBUG] LoadAppConfig success")
	return &appConfig, nil
}

// SaveAppConfig saves the application configuration to the configured JSON file.
func (s *appConfigService) SaveAppConfig(appConfig *domain.AppConfig) error {
	absPath, err := filepath.Abs(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", s.configPath, err)
	}

	data, err := json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal app config: %w", err)
	}

	err = ioutil.WriteFile(absPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write app config to file %s: %w", absPath, err)
	}

	return nil
}
