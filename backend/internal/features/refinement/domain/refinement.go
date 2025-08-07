package domain

import configdomain "sofa-commander/backend/internal/features/config/domain"

// TechStack defines the technology stack.
type TechStack struct {
	Frontend string `json:"frontend"`
	Backend  string `json:"backend"`
	Agent    string `json:"agent"`
}

// ModelParams defines the parameters for the AI model.
type ModelParams struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	Model       string  `json:"model"`
}

// RefinementRequest is the main request structure for starting a refinement process.
type RefinementRequest struct {
	InitialUserStory string `json:"initial_user_story"`
	TechStack        struct {
		Frontend string `json:"frontend"`
		Backend  string `json:"backend"`
		Agent    string `json:"agent"`
	} `json:"tech_stack"`
	ModelParams   ModelParams `json:"model_params"`
	SelectedRoles []string    `json:"selected_roles"`
}

// Question represents a question from a role.
type Question struct {
	Role   string   `json:"role"`
	Prompt []string `json:"prompt"`
	Answer string   `json:"answer,omitempty"` // PM's answer to the question
}

// Suggestion represents a suggestion from a role.
type Suggestion struct {
	Role   string   `json:"role"`
	Prompt []string `json:"prompt"`
}

// RefinementPhase defines the current phase of the refinement process.
type RefinementPhase string

const (
	PhaseQuestioning RefinementPhase = "QUESTIONING"
	PhaseSuggesting  RefinementPhase = "SUGGESTING"
	PhaseFinalizing  RefinementPhase = "FINALIZING"
)

// RefinementSession represents a full refinement session.
type RefinementSession struct {
	ID                     string                                       `json:"id"`
	ThreadID               string                                       `json:"thread_id"` // New: OpenAI Thread ID
	Request                RefinementRequest                            `json:"request"`
	UserStory              string                                       `json:"user_story"`
	RolePrompts            map[string]string                            `json:"role_prompts"` // Store role prompts for continued questioning
	PhasePrompts           map[string]string                            `json:"phase_prompts"`
	PhaseFormatExamples    map[string][]configdomain.PhaseFormatExample `json:"phase_format_examples"`
	Questions              []Question                                   `json:"questions,omitempty"`   // Stores questions during QUESTIONING phase
	Suggestions            []Suggestion                                 `json:"suggestions,omitempty"` // Stores suggestions during SUGGESTING phase
	History                []string                                     `json:"history,omitempty"`     // Stores conversation history
	Phase                  RefinementPhase                              `json:"phase"`
	AdditionalInfo         string                                       `json:"additional_info,omitempty"`         // 補充資訊
	ModificationSuggestion string                                       `json:"modification_suggestion,omitempty"` // 修改建議
}

// SubmitAnswersRequest is the request structure for submitting answers.
type SubmitAnswersRequest struct {
	SessionID      string            `json:"session_id"`
	Answers        map[string]string `json:"answers"`                   // Map of question_key (role_prompt) to answer
	AdditionalInfo string            `json:"additional_info,omitempty"` // 補充資訊
}

type AcceptSuggestionsRequest struct {
	SessionID           string       `json:"session_id"`
	AcceptedSuggestions []Suggestion `json:"accepted_suggestions"`
	NextPhase           string       `json:"next_phase"`
	AdditionalInfo      string       `json:"additional_info,omitempty"` // 補充資訊
}

type FinalizeRequest struct {
	SessionID              string            `json:"session_id"`
	CurrentPhase           string            `json:"current_phase"`
	CurrentAnswers         map[string]string `json:"current_answers,omitempty"`
	CurrentSuggestions     []string          `json:"current_suggestions,omitempty"`     // 只傳 key
	ModificationSuggestion string            `json:"modification_suggestion,omitempty"` // 修改建議
}
type FinalizeResponse struct {
	UserStory string   `json:"user_story"`
	AC        []string `json:"ac"`
	RawAI     string   `json:"raw_ai_response"`
}
