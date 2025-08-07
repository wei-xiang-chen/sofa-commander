# Sofa Commander - 多角色需求打磨助理：規劃與目標

## 專案總目標
協助產品經理 (PM) 從多種專業視角（產品、設計、前端、後端、使用者與行銷）反覆討論、篩選，最終產出新版 User Story 與詳細需求文件。

## 技術棧 (Tech Stack)
*   **前端 (Front-end)**: React + TypeScript
*   **後端 (Back-end)**: Go + Gin
*   **AI Agent**: OpenAI Assistants API (利用 Persistent Threads 進行對話管理)

## 核心工作流程 (已調整與確認)

1.  **階段一：釐清與提問 (Questioning & Clarification)**
    *   PM 上傳 `initial_user_story` 和 `product_context`。
    *   AI 扮演各個角色（目前僅限 ProductManager），根據輸入內容向 PM 提出**釐清問題**。
    *   PM 回答這些問題。
    *   此階段可**遞迴進行**：PM 可選擇「提交回答，繼續提問」，直到所有疑問被釐清。

2.  **階段二：建議 (Suggesting)**
    *   當 PM 認為問題已釐清，選擇「提交回答，獲取建議」。
    *   AI 根據完整的對話歷史，提供**結構化（JSON 格式）的建議列表**。
    *   PM 審閱建議，並**勾選採納**的項目。

3.  **階段三：最終化 (Finalizing)**
    *   PM 選擇「採納選定建議，進入下一輪打磨」或「完成討論，產出最終規格書」。
    *   **若選擇「下一輪打磨」**：系統根據採納的建議，彙整出「本輪優化後的 User Story」，並以此作為基礎，回到階段一，再次向 AI 提出釐清問題。
    *   **若選擇「產出最終規格書」**：AI 根據所有對話歷史和最終 User Story，生成一份**詳細的 Markdown 格式規格文件**。

## 目前開發進度與重點

*   **後端 Assistants API 整合 (進行中)**：
    *   已將 AI 互動從 Chat Completions API 遷移至 Assistants API。
    *   已實作 `GetOrCreateAssistant`、`CreateThread`、`AddMessageToThread`、`RunAssistant`、`GetAssistantResponse` 等核心 Assistants API 呼叫。
    *   已在 `StartSession`、`SubmitAnswersAndContinue`、`SubmitAnswersAndGetSuggestions` 中整合 Assistants API 邏輯。
    *   **重要更新**：**後端所有 linter 和編譯錯誤已由您解決，感謝您的協助！** `openai_client.go` 現在應該是正確的。

*   **後端配置管理**：
    *   已建立 `/api/config/save` API，可將配置（產品背景、角色提示詞、模型參數）寫入 `frontend/public/config.json`。

*   **前端介面**：
    *   初始輸入表單 (`product_context`, `initial_user_story`)。
    *   根據 `session.phase` 動態渲染介面：
        *   `QUESTIONING` 階段：顯示 AI 提出的問題，並提供文字輸入框供 PM 回答。
        *   `SUGGESTING` 階段：顯示 AI 提供的建議，並提供核取方塊供 PM 勾選。
    *   「設定」按鈕及模態視窗，用於修改 `product_context`、`role_prompts` 和 `model_params`。
    *   應用啟動時從 `config.json` 載入配置。

## 下一步重點

*   **實作「採納選定建議，進入下一輪打磨」的後端邏輯**：根據採納的建議，生成新的 User Story，並將其作為新的上下文傳遞給 AI，重新進入提問階段。
*   **實作「完成討論，產出最終規格書」的後端邏輯**：呼叫 AI 生成最終的規格文件。