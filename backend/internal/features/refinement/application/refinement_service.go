package application

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	configdomain "sofa-commander/backend/internal/features/config/domain"
	"sofa-commander/backend/internal/features/refinement/domain"
	"sofa-commander/backend/internal/features/refinement/infrastructure"
)

// In-memory store for sessions (for demonstration purposes)
var sessions = make(map[string]*domain.RefinementSession)
var sessionsMutex sync.RWMutex

// RefinementService defines the interface for the refinement application service.
type RefinementService interface {
	StartSession(req *domain.RefinementRequest, productContext string, rolePrompts, phasePrompts map[string]string, phaseFormatExamples map[string][]configdomain.PhaseFormatExample) (*domain.RefinementSession, error)
	SubmitAnswersAndContinue(sessionID string, answers map[string]string, additionalInfo string, rolePrompts, phasePrompts map[string]string, phaseFormatExamples map[string][]configdomain.PhaseFormatExample) (*domain.RefinementSession, error)
	SubmitAnswersAndGetSuggestions(sessionID string, answers map[string]string, additionalInfo string, rolePrompts, phasePrompts map[string]string, phaseFormatExamples map[string][]configdomain.PhaseFormatExample) (*domain.RefinementSession, error)
	AcceptSuggestions(sessionID string, acceptedSuggestions []domain.Suggestion, nextPhase string, additionalInfo string) (*domain.RefinementSession, []domain.Suggestion, error)
	Finalize(sessionID string, currentPhase string, currentAnswers map[string]string, currentSuggestions []string, modificationSuggestion string) (string, []string, string, error)
}

// refinementService is the implementation of RefinementService.
type refinementService struct {
	openaiClient infrastructure.OpenAIClient
	assistantID  string // Store the assistant ID here
}

// NewRefinementService creates a new instance of refinementService.
func NewRefinementService(client infrastructure.OpenAIClient) RefinementService {
	return &refinementService{openaiClient: client}
}

// StartSession starts a new refinement session by fetching questions from all roles concurrently.
func (s *refinementService) StartSession(req *domain.RefinementRequest, productContext string, rolePrompts, phasePrompts map[string]string, phaseFormatExamples map[string][]configdomain.PhaseFormatExample) (*domain.RefinementSession, error) {
	log.Println("StartSession: Received request.")
	userStory := req.InitialUserStory

	// 1. Get or Create Assistant
	assistantName := "Refinement Assistant"
	assistantInstructionsTemplate := `You are a multi-role requirement refinement assistant. Your goal is to help a Product Manager refine a user story.\n\nProduct Context: %s\n\nCurrent User Story to Refine: "%s"\n\nIMPORTANT GUIDELINES:\n1. All your questions and suggestions must be directly related to this specific user story\n2. Focus on clarifying implementation details, edge cases, and factors that could impact the successful delivery of THIS user story\n3. Consider the product context deeply - understand the target users, core values, and business goals\n4. Ask specific, actionable questions that can be answered with concrete information\n5. Provide suggestions that are measurable, implementable, and aligned with the product vision\n6. Avoid generic or theoretical questions/suggestions\n\nRoles:\n%s\n%s\n格式範例：%s\n請勿加上任何說明、標題或條列，僅回傳JSON。`
	// 只針對 selectedRoles 組合角色角度
	selectedRoles := req.SelectedRoles
	rolePromptsString := ""
	for _, role := range selectedRoles {
		if prompt, ok := rolePrompts[role]; ok {
			rolePromptsString += fmt.Sprintf("- %s: %s\n", role, prompt)
		}
	}
	// 組合階段說明
	phaseDesc := ""
	if phasePrompts != nil {
		phaseDesc = ""
		if len(selectedRoles) > 0 {
			phaseDesc = phasePrompts["questioning"]
		}
	}
	// 組合格式範例
	formatExample := ""
	if phaseFormatExamples != nil {
		if arr, ok := phaseFormatExamples["questioning"]; ok {
			// 只取 selectedRoles 的範例
			var filtered []configdomain.PhaseFormatExample
			for _, ex := range arr {
				for _, role := range selectedRoles {
					if ex.Role == role {
						filtered = append(filtered, ex)
					}
				}
			}
			b, _ := json.Marshal(filtered)
			formatExample = string(b)
		}
	}
	assistantInstructions := fmt.Sprintf(assistantInstructionsTemplate, productContext, userStory, rolePromptsString, phaseDesc, formatExample)

	assistantID, err := s.openaiClient.GetOrCreateAssistant(assistantName, assistantInstructions, "o4-mini") // Hardcoding model for now
	if err != nil {
		return nil, fmt.Errorf("failed to get or create assistant: %w", err)
	}
	s.assistantID = assistantID // Store for later use

	// 2. Create Thread
	threadID, err := s.openaiClient.CreateThread()
	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	// 3. Add initial User Story message to thread
	if err := s.openaiClient.AddMessageToThread(threadID, assistantInstructions); err != nil {
		return nil, fmt.Errorf("failed to add initial message to thread: %w", err)
	}

	// Run Assistant to get initial questions
	if err := s.openaiClient.RunAssistant(threadID, assistantID); err != nil {
		return nil, fmt.Errorf("failed to run assistant for initial questions: %w", err)
	}

	// Get Assistant's response (initial questions)
	assistantMessages, err := s.openaiClient.GetAssistantResponse(threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant response for initial questions: %w", err)
	}

	var questions []domain.Question
	if len(assistantMessages) > 0 {
		latest := assistantMessages[len(assistantMessages)-1]
		if len(latest.Content) > 0 {
			rawJSON := latest.Content[0].Text.Value
			// Extract JSON string from markdown code block if present
			if strings.HasPrefix(rawJSON, "```json") && strings.HasSuffix(rawJSON, "```") {
				rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
				rawJSON = strings.TrimSuffix(rawJSON, "\n```")
			}
			fmt.Println("[DEBUG] AI raw response:", rawJSON)
			err = json.Unmarshal([]byte(rawJSON), &questions)
			if err != nil {
				return nil, fmt.Errorf("failed to parse initial questions from AI: %w, raw response: %s", err, rawJSON)
			}
		}
	}

	session := &domain.RefinementSession{
		ID:                  fmt.Sprintf("session-%d", len(sessions)+1), // Generate a simple unique ID
		ThreadID:            threadID,
		Request:             *req,
		UserStory:           userStory,
		RolePrompts:         rolePrompts, // Store role prompts
		PhasePrompts:        phasePrompts,
		PhaseFormatExamples: phaseFormatExamples,
		Questions:           questions,
		Phase:               domain.PhaseQuestioning,           // Set initial phase
		History:             []string{"[初始用戶故事] " + userStory}, // Keep history for our own reference/logging
	}

	sessionsMutex.Lock()
	sessions[session.ID] = session
	sessionsMutex.Unlock()

	log.Println("StartSession: Returning session.")
	return session, nil
}

// SubmitAnswersAndContinue updates the session with answers and generates new questions.
func (s *refinementService) SubmitAnswersAndContinue(sessionID string, answers map[string]string, additionalInfo string, rolePrompts, phasePrompts map[string]string, phaseFormatExamples map[string][]configdomain.PhaseFormatExample) (*domain.RefinementSession, error) {
	sessionsMutex.RLock()
	session, ok := sessions[sessionID]
	sessionsMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Update session with answers
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	userResponse := ""
	for i := range session.Questions {
		for _, p := range session.Questions[i].Prompt {
			key := session.Questions[i].Role + "_" + p
			if ans, found := answers[key]; found {
				session.Questions[i].Answer = ans
				userResponse += fmt.Sprintf("PM Answer to %s's question \"%s\": %s\n", session.Questions[i].Role, p, ans)
			}
		}
	}

	if strings.TrimSpace(userResponse) != "" {
		if err := s.openaiClient.AddMessageToThread(session.ThreadID, userResponse); err != nil {
			return nil, fmt.Errorf("failed to add user response to thread: %w", err)
		}
	}

	// 組合提問階段 prompt
	// 只針對 session.Request.SelectedRoles 組合角色角度
	selectedRoles := session.Request.SelectedRoles
	var rolePromptsString string
	for _, role := range selectedRoles {
		if prompt, ok := rolePrompts[role]; ok {
			rolePromptsString += fmt.Sprintf("- %s: %s\n", role, prompt)
		}
	}
	phaseDesc := ""
	if phasePrompts != nil {
		phaseDesc = ""
		if len(selectedRoles) > 0 {
			phaseDesc = phasePrompts["questioning"]
		}
	}
	formatExample := ""
	if phaseFormatExamples != nil {
		if arr, ok := phaseFormatExamples["questioning"]; ok {
			var filtered []configdomain.PhaseFormatExample
			for _, ex := range arr {
				for _, role := range selectedRoles {
					if ex.Role == role {
						filtered = append(filtered, ex)
					}
				}
			}
			b, _ := json.Marshal(filtered)
			formatExample = string(b)
		}
	}

	// 組合完整的指令，包含補充資訊
	instructionMessage := "基於當前的 User Story 和對話歷史，請根據下列角色角度：\n" + rolePromptsString + "\n" + phaseDesc + "\n格式範例：" + formatExample + "\n請勿加上任何說明、標題或條列，僅回傳 JSON 陣列。"

	// 如果有補充資訊，整合到指令中
	if strings.TrimSpace(additionalInfo) != "" {
		instructionMessage = "補充資訊：\n" + additionalInfo + "\n\n" + instructionMessage
	}
	if err := s.openaiClient.AddMessageToThread(session.ThreadID, instructionMessage); err != nil {
		return nil, fmt.Errorf("failed to add instruction message to thread: %w", err)
	}

	// Run Assistant to get new questions
	if err := s.openaiClient.RunAssistant(session.ThreadID, s.assistantID); err != nil {
		return nil, fmt.Errorf("failed to run assistant for new questions: %w", err)
	}

	// Get Assistant's response (new questions)
	assistantMessages, err := s.openaiClient.GetAssistantResponse(session.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant response for new questions: %w", err)
	}

	var newQuestions []domain.Question
	if len(assistantMessages) > 0 {
		latest := assistantMessages[len(assistantMessages)-1]
		if len(latest.Content) > 0 {
			rawJSON := latest.Content[0].Text.Value
			// Extract JSON string from markdown code block if present
			if strings.HasPrefix(rawJSON, "```json") && strings.HasSuffix(rawJSON, "```") {
				rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
				rawJSON = strings.TrimSuffix(rawJSON, "\n```")
			}
			fmt.Println("[DEBUG] AI raw response:", rawJSON)
			err = json.Unmarshal([]byte(rawJSON), &newQuestions)
			if err != nil {
				return nil, fmt.Errorf("failed to parse new questions from AI: %w, raw response: %s", err, rawJSON)
			}
		}
	}

	session.Questions = newQuestions // Replace old questions with new ones
	// Keep phase as QUESTIONING

	return session, nil
}

// SubmitAnswersAndGetSuggestions updates the session with answers and generates suggestions.
func (s *refinementService) SubmitAnswersAndGetSuggestions(sessionID string, answers map[string]string, additionalInfo string, rolePrompts, phasePrompts map[string]string, phaseFormatExamples map[string][]configdomain.PhaseFormatExample) (*domain.RefinementSession, error) {
	sessionsMutex.RLock()
	session, ok := sessions[sessionID]
	sessionsMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Update session with answers
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	userResponse := ""
	for i := range session.Questions {
		for _, p := range session.Questions[i].Prompt {
			key := session.Questions[i].Role + "_" + p
			if ans, found := answers[key]; found {
				session.Questions[i].Answer = ans
				userResponse += fmt.Sprintf("PM Answer to %s's question \"%s\": %s\n", session.Questions[i].Role, p, ans)
			}
		}
	}

	if strings.TrimSpace(userResponse) != "" {
		if err := s.openaiClient.AddMessageToThread(session.ThreadID, userResponse); err != nil {
			return nil, fmt.Errorf("failed to add user response to thread: %w", err)
		}
	}

	// 組合建議階段 prompt
	// 只針對 session.Request.SelectedRoles 組合角色角度
	selectedRoles := session.Request.SelectedRoles
	var rolePromptsString string
	for _, role := range selectedRoles {
		if prompt, ok := rolePrompts[role]; ok {
			rolePromptsString += fmt.Sprintf("- %s: %s\n", role, prompt)
		}
	}
	phaseDesc := ""
	if phasePrompts != nil {
		phaseDesc = ""
		if len(selectedRoles) > 0 {
			phaseDesc = phasePrompts["suggesting"]
		}
	}
	formatExample := ""
	if phaseFormatExamples != nil {
		if arr, ok := phaseFormatExamples["suggesting"]; ok {
			var filtered []configdomain.PhaseFormatExample
			for _, ex := range arr {
				for _, role := range selectedRoles {
					if ex.Role == role {
						filtered = append(filtered, ex)
					}
				}
			}
			b, _ := json.Marshal(filtered)
			formatExample = string(b)
		}
	}

	// 組合完整的指令，包含補充資訊
	instructionMessage := "基於當前的 User Story 和對話歷史，請根據下列角色角度：\n" + rolePromptsString + "\n" + phaseDesc + "\n格式範例：" + formatExample + "\n請勿再提出任何問題，也不要有多餘說明、標題或條列，僅回傳 JSON 陣列。"

	// 如果有補充資訊，整合到指令中
	if strings.TrimSpace(additionalInfo) != "" {
		instructionMessage = "補充資訊：\n" + additionalInfo + "\n\n" + instructionMessage
	}
	if err := s.openaiClient.AddMessageToThread(session.ThreadID, instructionMessage); err != nil {
		return nil, fmt.Errorf("failed to add instruction message to thread: %w", err)
	}

	// Run Assistant to get suggestions
	if err := s.openaiClient.RunAssistant(session.ThreadID, s.assistantID); err != nil {
		return nil, fmt.Errorf("failed to run assistant for suggestions: %w", err)
	}

	// Get Assistant's response (suggestions)
	assistantMessages, err := s.openaiClient.GetAssistantResponse(session.ThreadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assistant response for suggestions: %w", err)
	}

	var suggestions []domain.Suggestion
	if len(assistantMessages) > 0 {
		latest := assistantMessages[len(assistantMessages)-1]
		if len(latest.Content) > 0 {
			rawJSON := latest.Content[0].Text.Value
			// Extract JSON string from markdown code block if present
			if strings.HasPrefix(rawJSON, "```json") && strings.HasSuffix(rawJSON, "```") {
				rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
				rawJSON = strings.TrimSuffix(rawJSON, "\n```")
			}
			fmt.Println("[DEBUG] AI raw response:", rawJSON)
			err = json.Unmarshal([]byte(rawJSON), &suggestions)
			if err != nil {
				return nil, fmt.Errorf("failed to parse suggestions from AI: %w, raw response: %s", err, rawJSON)
			}
		}
	}

	session.Suggestions = suggestions
	session.Questions = nil                // Clear questions once suggestions are generated
	session.Phase = domain.PhaseSuggesting // Change phase to SUGGESTING

	return session, nil
}

// AcceptSuggestions accepts suggestions and starts a new refinement round.
func (s *refinementService) AcceptSuggestions(sessionID string, acceptedSuggestions []domain.Suggestion, nextPhase string, additionalInfo string) (*domain.RefinementSession, []domain.Suggestion, error) {
	sessionsMutex.RLock()
	session, ok := sessions[sessionID]
	sessionsMutex.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("session %s not found", sessionID)
	}

	// 將被採納的建議組合成新 context，送給 AI 產生新一輪問題
	acceptedText := "[採納建議] \n"
	if len(acceptedSuggestions) == 0 {
		acceptedText += "(本輪未勾選任何建議)\n"
	} else {
		for _, s := range acceptedSuggestions {
			for _, p := range s.Prompt {
				acceptedText += fmt.Sprintf("- %s: %s\n", s.Role, p)
			}
		}
	}

	// 這裡直接 append 建議內容到 thread
	if err := s.openaiClient.AddMessageToThread(session.ThreadID, acceptedText); err != nil {
		return nil, nil, fmt.Errorf("failed to add accepted suggestions to thread: %w", err)
	}

	// 根據 nextPhase 決定進入提問還是建議階段
	var phaseKey string
	var setQuestions bool
	if nextPhase == "suggesting" {
		phaseKey = "suggesting"
		setQuestions = false
	} else {
		phaseKey = "questioning"
		setQuestions = true
	}
	var rolePromptsString string
	for _, role := range session.Request.SelectedRoles {
		if prompt, ok := session.RolePrompts[role]; ok {
			rolePromptsString += fmt.Sprintf("- %s: %s\n", role, prompt)
		}
	}
	phaseDesc := ""
	if session.PhasePrompts != nil {
		phaseDesc = session.PhasePrompts[phaseKey]
	}
	formatExample := ""
	if arr, ok := session.PhaseFormatExamples[phaseKey]; ok {
		var filtered []configdomain.PhaseFormatExample
		for _, ex := range arr {
			for _, role := range session.Request.SelectedRoles {
				if ex.Role == role {
					filtered = append(filtered, ex)
				}
			}
		}
		b, _ := json.Marshal(filtered)
		formatExample = string(b)
	}

	// 組合完整的指令，包含補充資訊
	instructionMessage := "基於當前的 User Story 和對話歷史，請根據下列角色角度：\n" + rolePromptsString + "\n" + phaseDesc + "\n格式範例：" + formatExample + "\n請勿加上任何說明、標題或條列，僅回傳 JSON 陣列。"
	if strings.TrimSpace(instructionMessage) == "" {
		// fallback 根據 phaseKey 給預設 prompt
		if phaseKey == "suggesting" {
			instructionMessage = "基於當前的 User Story 和對話歷史，請給我下輪建議，僅回傳 JSON 陣列。"
		} else {
			instructionMessage = "基於當前的 User Story 和對話歷史，請給我下輪提問，僅回傳 JSON 陣列。"
		}
	}

	// 如果有補充資訊，整合到指令中
	if strings.TrimSpace(additionalInfo) != "" {
		instructionMessage = "補充資訊：\n" + additionalInfo + "\n\n" + instructionMessage
	}
	if err := s.openaiClient.AddMessageToThread(session.ThreadID, instructionMessage); err != nil {
		return nil, nil, fmt.Errorf("failed to add instruction message to thread: %w", err)
	}

	// Run Assistant to get new questions or suggestions
	if err := s.openaiClient.RunAssistant(session.ThreadID, s.assistantID); err != nil {
		return nil, nil, fmt.Errorf("failed to run assistant for new round: %w", err)
	}

	assistantMessages, err := s.openaiClient.GetAssistantResponse(session.ThreadID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get assistant response for new round: %w", err)
	}

	if setQuestions {
		var newQuestions []domain.Question
		if len(assistantMessages) > 0 {
			latest := assistantMessages[len(assistantMessages)-1]
			if len(latest.Content) > 0 {
				rawJSON := latest.Content[0].Text.Value
				if strings.HasPrefix(rawJSON, "```json") && strings.HasSuffix(rawJSON, "```") {
					rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
					rawJSON = strings.TrimSuffix(rawJSON, "\n```")
				}
				fmt.Println("[DEBUG] AI raw response:", rawJSON)
				err = json.Unmarshal([]byte(rawJSON), &newQuestions)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse new questions from AI: %w, raw response: %s", err, rawJSON)
				}
			}
		}
		sessionsMutex.Lock()
		session.Questions = newQuestions
		session.Suggestions = nil
		session.Phase = domain.PhaseQuestioning
		sessionsMutex.Unlock()
	} else {
		var newSuggestions []domain.Suggestion
		if len(assistantMessages) > 0 {
			latest := assistantMessages[len(assistantMessages)-1]
			if len(latest.Content) > 0 {
				rawJSON := latest.Content[0].Text.Value
				if strings.HasPrefix(rawJSON, "```json") && strings.HasSuffix(rawJSON, "```") {
					rawJSON = strings.TrimPrefix(rawJSON, "```json\n")
					rawJSON = strings.TrimSuffix(rawJSON, "\n```")
				}
				fmt.Println("[DEBUG] AI raw response:", rawJSON)
				err = json.Unmarshal([]byte(rawJSON), &newSuggestions)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse new suggestions from AI: %w, raw response: %s", err, rawJSON)
				}
			}
		}
		sessionsMutex.Lock()
		session.Questions = nil
		session.Suggestions = newSuggestions
		session.Phase = domain.PhaseSuggesting
		sessionsMutex.Unlock()
	}

	return session, acceptedSuggestions, nil
}

// Finalize 產生 user story + AC
func (s *refinementService) Finalize(sessionID string, currentPhase string, currentAnswers map[string]string, currentSuggestions []string, modificationSuggestion string) (string, []string, string, error) {
	sessionsMutex.RLock()
	session, ok := sessions[sessionID]
	sessionsMutex.RUnlock()
	if !ok {
		return "", nil, "", fmt.Errorf("session %s not found", sessionID)
	}

	// 1. 先將當前數據加入到 thread
	if currentPhase == "QUESTIONING" && len(currentAnswers) > 0 {
		// 將當前回答加入到 thread
		userResponse := ""
		for i := range session.Questions {
			for _, p := range session.Questions[i].Prompt {
				key := session.Questions[i].Role + "_" + p
				if ans, found := currentAnswers[key]; found {
					userResponse += fmt.Sprintf("PM Answer to %s's question \"%s\": %s\n", session.Questions[i].Role, p, ans)
				}
			}
		}
		if strings.TrimSpace(userResponse) != "" {
			if err := s.openaiClient.AddMessageToThread(session.ThreadID, userResponse); err != nil {
				return "", nil, "", fmt.Errorf("failed to add current answers to thread: %w", err)
			}
		}
	} else if currentPhase == "SUGGESTING" && len(currentSuggestions) > 0 {
		// 將當前採納的建議加入到 thread
		acceptedText := "[採納建議] \n"
		for _, suggestionKey := range currentSuggestions {
			// 從 session.Suggestions 中找到對應的建議
			for _, s := range session.Suggestions {
				for _, p := range s.Prompt {
					if s.Role+"_"+p == suggestionKey {
						acceptedText += fmt.Sprintf("- %s: %s\n", s.Role, p)
					}
				}
			}
		}
		if err := s.openaiClient.AddMessageToThread(session.ThreadID, acceptedText); err != nil {
			return "", nil, "", fmt.Errorf("failed to add current suggestions to thread: %w", err)
		}
	}

	// 如果有修改建議，加入到 thread
	if strings.TrimSpace(modificationSuggestion) != "" {
		message := "[修改建議]\n" + modificationSuggestion
		if err := s.openaiClient.AddMessageToThread(session.ThreadID, message); err != nil {
			return "", nil, "", fmt.Errorf("failed to add modification suggestion to thread: %w", err)
		}
	}

	// 組合 prompt - 明確要求 AI 基於對話歷史進行改進
	prompt := `你現在需要基於我們在這個 thread 中的完整對話歷史，重新撰寫一個改進版的用戶故事。

請仔細分析以下內容：
1. 原始用戶故事是什麼
2. 各角色提出了哪些問題
3. 產品經理如何回答這些問題
4. 各角色提供了哪些建議
5. 產品經理採納了哪些建議

基於這些對話內容，請：
- 整合所有有價值的資訊和需求
- 解決對話中提到的問題和疑慮
- 加入採納的建議內容
- 使新的用戶故事更加完整、具體和可執行
- 確保用戶故事符合產品背景中的核心價值和用戶需求
- 驗收標準要具體、可測量、可測試

重要要求：
1. 不要只是重複原始用戶故事，而是要進行實質性的改進和補充
2. 用戶故事應該包含明確的用戶角色、目標和價值
3. 驗收標準應該涵蓋功能完整性、用戶體驗、技術要求和業務價值
4. 考慮產品背景中的專注力管理、環保意識、社群參與等核心價值

請按照以下格式回傳：

【用戶故事】
改進後的用戶故事內容（必須基於對話歷史進行實質性改進）

【驗收標準】
1. 驗收標準1（具體、可測量）
2. 驗收標準2（具體、可測量）
3. 驗收標準3（具體、可測量）
4. 驗收標準4（具體、可測量）
5. 驗收標準5（具體、可測量）`
	if err := s.openaiClient.AddMessageToThread(session.ThreadID, prompt); err != nil {
		return "", nil, "", fmt.Errorf("failed to add finalize prompt to thread: %w", err)
	}
	if err := s.openaiClient.RunAssistant(session.ThreadID, s.assistantID); err != nil {
		return "", nil, "", fmt.Errorf("failed to run assistant for finalize: %w", err)
	}
	assistantMessages, err := s.openaiClient.GetAssistantResponse(session.ThreadID)
	if err != nil {
		return "", nil, "", fmt.Errorf("failed to get assistant response for finalize: %w", err)
	}
	if len(assistantMessages) == 0 || len(assistantMessages[len(assistantMessages)-1].Content) == 0 {
		return "", nil, "", fmt.Errorf("AI did not return any content")
	}
	raw := assistantMessages[len(assistantMessages)-1].Content[0].Text.Value

	// 解析純文字格式
	userStory := ""
	ac := []string{}

	// 尋找【用戶故事】和【驗收標準】標記
	userStoryStart := strings.Index(raw, "【用戶故事】")
	userStoryEnd := strings.Index(raw, "【驗收標準】")

	if userStoryStart != -1 && userStoryEnd != -1 {
		// 提取用戶故事
		userStory = strings.TrimSpace(raw[userStoryStart+len("【用戶故事】") : userStoryEnd])

		// 提取驗收標準
		acSection := raw[userStoryEnd+len("【驗收標準】"):]
		lines := strings.Split(acSection, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && (strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "2.") || strings.HasPrefix(line, "3.") || strings.HasPrefix(line, "4.") || strings.HasPrefix(line, "5.")) {
				// 移除數字前綴
				acItem := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(line, "1."), "2."), "3."), "4."), "5."))
				if acItem != "" {
					ac = append(ac, acItem)
				}
			}
		}
	} else {
		// fallback: 如果找不到標記，直接回傳原始內容作為用戶故事
		userStory = raw
	}

	return userStory, ac, raw, nil
}
