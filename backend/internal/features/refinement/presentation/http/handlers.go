package http

import (
	"log"
	"net/http"

	"sofa-commander/backend/internal/config"
	"sofa-commander/backend/internal/features/refinement/application"
	"sofa-commander/backend/internal/features/refinement/domain"

	"github.com/gin-gonic/gin"
)

// RefinementHandler holds the refinement service and app config service.
type RefinementHandler struct {
	refinementService application.RefinementService
	appConfigService  config.AppConfigService
}

// NewRefinementHandler creates a new RefinementHandler.
func NewRefinementHandler(refinementService application.RefinementService, appConfigService config.AppConfigService) *RefinementHandler {
	return &RefinementHandler{
		refinementService: refinementService,
		appConfigService:  appConfigService,
	}
}

// StartRefinementHandler handles the request to start a new refinement process.
func (h *RefinementHandler) StartRefinementHandler(c *gin.Context) {
	var req domain.RefinementRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load app config to get product context and role prompts
	appConfig, err := h.appConfigService.LoadAppConfig()
	if err != nil {
		log.Println("[ERROR] Failed to load app config:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load app config: " + err.Error()})
		return
	}

	// Start a new session
	session, err := h.refinementService.StartSession(&req, appConfig.ProductContext, appConfig.RolePrompts, appConfig.PhasePrompts, appConfig.PhaseFormatExamples)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start refinement session: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// SubmitAnswersAndContinueHandler handles the request to submit answers and continue questioning.
func (h *RefinementHandler) SubmitAnswersAndContinueHandler(c *gin.Context) {
	var req domain.SubmitAnswersRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load app config for question prompts
	appConfig, err := h.appConfigService.LoadAppConfig()
	if err != nil {
		log.Println("[ERROR] Failed to load app config:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load app config: " + err.Error()})
		return
	}

	// Submit answers and continue
	session, err := h.refinementService.SubmitAnswersAndContinue(req.SessionID, req.Answers, req.AdditionalInfo, appConfig.RolePrompts, appConfig.PhasePrompts, appConfig.PhaseFormatExamples)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit answers and continue: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// SubmitAnswersAndGetSuggestionsHandler handles the request to submit answers and get suggestions.
func (h *RefinementHandler) SubmitAnswersAndGetSuggestionsHandler(c *gin.Context) {
	var req domain.SubmitAnswersRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Load app config for suggestion prompts
	appConfig, err := h.appConfigService.LoadAppConfig()
	if err != nil {
		log.Println("[ERROR] Failed to load app config:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load app config: " + err.Error()})
		return
	}

	// Submit answers and get suggestions
	session, err := h.refinementService.SubmitAnswersAndGetSuggestions(req.SessionID, req.Answers, req.AdditionalInfo, appConfig.RolePrompts, appConfig.PhasePrompts, appConfig.PhaseFormatExamples)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit answers and get suggestions: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, session)
}

// AcceptSuggestionsHandler handles accepting suggestions and starting a new refinement round.
func (h *RefinementHandler) AcceptSuggestionsHandler(c *gin.Context) {
	var req domain.AcceptSuggestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session, prevResult, err := h.refinementService.AcceptSuggestions(req.SessionID, req.AcceptedSuggestions, req.NextPhase, req.AdditionalInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to accept suggestions: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"session": session, "previous_result": prevResult})
}

// FinalizeHandler handles generating the final user story and AC.
func (h *RefinementHandler) FinalizeHandler(c *gin.Context) {
	var req domain.FinalizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userStory, ac, rawAI, err := h.refinementService.Finalize(req.SessionID, req.CurrentPhase, req.CurrentAnswers, req.CurrentSuggestions, req.ModificationSuggestion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain.FinalizeResponse{UserStory: userStory, AC: ac, RawAI: rawAI})
}
