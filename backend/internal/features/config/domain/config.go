package domain

// AppConfig represents the application configuration.
type AppConfig struct {
	ProductContext      string                          `json:"product_context"`
	RolePrompts         map[string]string               `json:"role_prompts"`
	PhasePrompts        map[string]string               `json:"phase_prompts"`
	PhaseFormatExamples map[string][]PhaseFormatExample `json:"phase_format_examples"`
	ModelParams         ModelParams                     `json:"model_params"`
}

// ModelParams defines the parameters for the AI model.
type ModelParams struct {
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

type PhaseFormatExample struct {
	Role   string   `json:"role"`
	Prompt []string `json:"prompt"`
}
