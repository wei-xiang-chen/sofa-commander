package main

import (
	"log"
	"net/http"

	"sofa-commander/backend/internal/config"
	config_http "sofa-commander/backend/internal/features/config/presentation/http"
	"sofa-commander/backend/internal/features/refinement/application"
	"sofa-commander/backend/internal/features/refinement/infrastructure"
	refinement_http "sofa-commander/backend/internal/features/refinement/presentation/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Initialize OpenAI client
	openaiClient, err := infrastructure.NewOpenAIClient()
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	// Initialize services
	refinementService := application.NewRefinementService(openaiClient)
	appConfigService := config.NewAppConfigService("config/app_config.json")

	// Refinement API routes
	refineGroup := r.Group("/api/refine")
	{
		handler := refinement_http.NewRefinementHandler(refinementService, appConfigService)
		refineGroup.POST("/start", handler.StartRefinementHandler)
		refineGroup.POST("/submit_answers_and_continue", handler.SubmitAnswersAndContinueHandler)
		refineGroup.POST("/submit_answers_and_get_suggestions", handler.SubmitAnswersAndGetSuggestionsHandler)
		refineGroup.POST("/accept_suggestions", handler.AcceptSuggestionsHandler)
		refineGroup.POST("/finalize", handler.FinalizeHandler)
	}

	// Config API routes
	configGroup := r.Group("/api/config")
	{
		configGroup.GET("/app", config_http.NewAppConfigHandler(appConfigService).GetAppConfigHandler)
		configGroup.POST("/app", config_http.NewAppConfigHandler(appConfigService).SaveAppConfigHandler)
	}

	r.Run(":8080") // listen and serve on 0.0.0.0:8080
}
