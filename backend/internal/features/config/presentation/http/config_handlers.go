package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sofa-commander/backend/internal/config"
	"sofa-commander/backend/internal/features/config/domain"
)

// AppConfigHandler holds the app config service.
type AppConfigHandler struct {
	appConfigService config.AppConfigService
}

// NewAppConfigHandler creates a new AppConfigHandler.
func NewAppConfigHandler(appConfigService config.AppConfigService) *AppConfigHandler {
	return &AppConfigHandler{
		appConfigService: appConfigService,
	}
}

// GetAppConfigHandler handles fetching the application configuration.
func (h *AppConfigHandler) GetAppConfigHandler(c *gin.Context) {
	appConfig, err := h.appConfigService.LoadAppConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load app config: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, appConfig)
}

// SaveAppConfigHandler handles saving the application configuration.
func (h *AppConfigHandler) SaveAppConfigHandler(c *gin.Context) {
	var appConfig domain.AppConfig
	if err := c.ShouldBindJSON(&appConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.appConfigService.SaveAppConfig(&appConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save app config: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "App config saved successfully"})
}
