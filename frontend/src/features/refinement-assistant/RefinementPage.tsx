import React, { useState, useEffect } from 'react';

// Define TypeScript interfaces for our data structures
interface Question {
  role: string;
  prompt: string[];
  answer?: string; // Optional field for PM's answer
}

interface Suggestion {
  role: string;
  prompt: string[];
}

type RefinementPhase = "QUESTIONING" | "SUGGESTING" | "FINALIZING";

interface RefinementSession {
  id: string; // Changed from ID to id to match backend JSON tag
  user_story: string; // Changed from UserStory to user_story
  questions?: Question[]; // Now stores questions, optional
  suggestions?: Suggestion[]; // Stores suggestions, optional
  phase: RefinementPhase; // Current phase of the refinement process
}

// 新增 FinalizeResult 介面
interface FinalizeResult {
  user_story: string;
  ac: string[];
  raw_ai_response: string;
}

interface ModelParams {
  temperature: number;
  max_tokens: number;
  model: string;
}

interface AppConfig {
  product_context: string;
  role_prompts: { [key: string]: string };
  model_params: ModelParams;
}

// Modal component for error display
const ErrorModal: React.FC<{ message: string; onClose: () => void }> = ({ message, onClose }) => (
  <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.4)', zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
    <div style={{ background: 'white', padding: 24, borderRadius: 8, maxWidth: 500, width: '90%' }}>
      <h4>錯誤訊息</h4>
      <textarea style={{ width: '100%', minHeight: 120, fontFamily: 'monospace' }} value={message} readOnly />
      <div style={{ textAlign: 'right', marginTop: 16 }}>
        <button className="btn btn-secondary" onClick={onClose}>關閉</button>
      </div>
    </div>
  </div>
);

const RefinementPage: React.FC = () => {
  const [initialUserStory, setInitialUserStory] = useState('');
  const [session, setSession] = useState<RefinementSession | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [toastMessage, setToastMessage] = useState<string | null>(null);
  const [showToast, setShowToast] = useState(false);
  const [answers, setAnswers] = useState<{ [key: string]: string }>({}); // State to store answers
  const [selectedSuggestions, setSelectedSuggestions] = useState<string[]>([]); // State to store selected suggestion keys
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null); // State for full app config
  const [isSettingsOpen, setIsSettingsOpen] = useState(false); // State to control settings modal/section
  const [selectedRoles, setSelectedRoles] = useState<string[]>([]);
  const [previousResult, setPreviousResult] = useState<Suggestion[] | null>(null);
  const [finalizeResult, setFinalizeResult] = useState<FinalizeResult | null>(null); // 新增 finalizeResult state
  const [additionalInfo, setAdditionalInfo] = useState<string>("");
  const [modificationSuggestion, setModificationSuggestion] = useState<string>("");

  // Load app config on component mount
  useEffect(() => {
    const loadAppConfig = async () => {
      try {
        const res = await fetch('/api/config/app');
        if (!res.ok) {
          throw new Error(`Failed to load app config: ${res.status} ${res.statusText}`);
        }
        const data: AppConfig = await res.json();
        setAppConfig(data);
      } catch (e: any) {
        console.error("Error loading app config:", e);
        setToastMessage(`Error loading app config: ${e.message}`);
        setShowToast(true);
      }
    };
    loadAppConfig();
  }, []);

  // 只要進入 QUESTIONING 階段就自動清空 previousResult
  useEffect(() => {
    if (session && session.phase === "QUESTIONING") {
      setPreviousResult(null);
    }
  }, [session && session.phase]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setToastMessage(null);
    setShowToast(false);
    setIsLoading(true);

    if (!appConfig) {
      setToastMessage("App configuration not loaded. Please refresh the page.");
      setShowToast(true);
      setIsLoading(false);
      return;
    }

    const requestBody = {
      initial_user_story: initialUserStory,
      tech_stack: {
        frontend: "React + TypeScript",
        backend: "Go + Gin",
        agent: "OpenAI ChatGPT API (gpt-4-turbo)"
      },
      model_params: appConfig.model_params,
      selected_roles: selectedRoles,
    };

    try {
      const res = await fetch('/api/refine/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      if (!res.ok) {
        let errorMessage = `HTTP error! Status: ${res.status} ${res.statusText || ''}`;
        try {
          const errorData = await res.json();
          if (errorData && errorData.error) {
            errorMessage = errorData.error;
          } else if (errorData && errorData.message) {
            errorMessage = errorData.message;
          } else {
            errorMessage = JSON.stringify(errorData);
          }
        } catch (jsonError) {
          const textError = await res.text();
          errorMessage = `HTTP error! Status: ${res.status} ${res.statusText || ''}. Response: ${textError}`;
        }
        throw new Error(errorMessage);
      }

      const data: RefinementSession = await res.json();
      console.log("Received data from backend:", data);
      setSession(data);
      // Initialize answers state based on received questions
      const initialAnswers: { [key: string]: string } = {};
      data.questions?.forEach(q => {
        initialAnswers[q.role + '_' + q.prompt] = q.answer || '';
      });
      setAnswers(initialAnswers);

    } catch (e: any) {
      console.error("Frontend caught error:", e);
      setToastMessage(e.message);
      setShowToast(true);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAnswerChange = (role: string, prompt: string, value: string) => {
    setAnswers(prevAnswers => ({
      ...prevAnswers,
      [role + '_' + prompt]: value,
    }));
  };

  const handleSuggestionChange = (key: string) => {
    setSelectedSuggestions(prevSelected => {
      if (prevSelected.includes(key)) {
        return prevSelected.filter(item => item !== key);
      } else {
        return [...prevSelected, key];
      }
    });
  };

  const submitAnswers = async (endpoint: string) => {
    if (!session) return;

    setIsLoading(true);
    setToastMessage(null);
    setShowToast(false);

    const requestBody = {
      session_id: session.id,
      answers: answers,
      additional_info: additionalInfo,
    };

    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      if (!res.ok) {
        let errorMessage = `HTTP error! Status: ${res.status} ${res.statusText || ''}`;
        try {
          const errorData = await res.json();
          if (errorData && errorData.error) {
            errorMessage = errorData.error;
          } else if (errorData && errorData.message) {
            errorMessage = errorData.message;
          } else {
            errorMessage = JSON.stringify(errorData);
          }
        } catch (jsonError) {
          const textError = await res.text();
          errorMessage = `HTTP error! Status: ${res.status} ${res.statusText || ''}. Response: ${textError}`;
        }
        throw new Error(errorMessage);
      }

      const data: RefinementSession = await res.json();
      console.log("Received data from backend after submitting answers:", data);
      setSession(data);
      // Re-initialize answers state for new questions if any
      const newInitialAnswers: { [key: string]: string } = {};
      data.questions?.forEach(q => {
        newInitialAnswers[q.role + '_' + q.prompt] = q.answer || '';
      });
      setAnswers(newInitialAnswers);
      setSelectedSuggestions([]); // Clear selected suggestions for new phase
      setAdditionalInfo(""); // Clear additional info for new phase

    } catch (e: any) {
      console.error("Frontend caught error after submitting answers:", e);
      setToastMessage(e.message);
      setShowToast(true);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSubmitAnswersAndContinue = () => {
    submitAnswers('/api/refine/submit_answers_and_continue');
  };

  const handleSubmitAnswersAndGetSuggestions = () => {
    submitAnswers('/api/refine/submit_answers_and_get_suggestions');
  };



  // 新增：可指定 phase 的 accept suggestions
  const handleAcceptSuggestionsWithPhase = async (nextPhase: "questioning" | "suggesting") => {
    if (!session || !session.suggestions) return;
    setIsLoading(true);
    setToastMessage(null);
    setShowToast(false);

    // 將勾選的建議組成 array
    const accepted: Suggestion[] = [];
    session.suggestions.forEach(s => {
      s.prompt.forEach(p => {
        if (selectedSuggestions.includes(s.role + '_' + p)) {
          accepted.push({ role: s.role, prompt: [p] });
        }
      });
    });

    const requestBody = {
      session_id: session.id,
      accepted_suggestions: accepted,
      next_phase: nextPhase,
    };

    try {
      const res = await fetch('/api/refine/accept_suggestions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
      });
      if (!res.ok) {
        let errorMessage = `HTTP error! Status: ${res.status} ${res.statusText || ''}`;
        try {
          const errorData = await res.json();
          if (errorData && errorData.error) {
            errorMessage = errorData.error;
          } else if (errorData && errorData.message) {
            errorMessage = errorData.message;
          } else {
            errorMessage = JSON.stringify(errorData);
          }
        } catch (jsonError) {
          const textError = await res.text();
          errorMessage = `HTTP error! Status: ${res.status} ${res.statusText || ''}. Response: ${textError}`;
        }
        throw new Error(errorMessage);
      }
      const data = await res.json();
      console.log("[DEBUG] API 回傳 session.phase:", data.session.phase);
      console.log("[DEBUG] API 回傳 session:", data.session);
      setSession(data.session);
      // 只在 SUGGESTING 階段顯示 previous_result
      if (data.session && data.session.phase === "SUGGESTING") {
        setPreviousResult(data.previous_result);
      } else {
        setPreviousResult(null);
      }
      setSelectedSuggestions([]);
      setAdditionalInfo(""); // Clear additional info for new phase
    } catch (e: any) {
      setToastMessage(e.message);
      setShowToast(true);
    } finally {
      setIsLoading(false);
    }
  };

  // 新增 handleFinalize 函數
  const handleFinalize = async () => {
    if (!session) return;

    setIsLoading(true);
    try {
      // 收集當前狀態的數據
      const finalizeData = {
        session_id: session.id,
        current_phase: session.phase,
        current_answers: session.phase === "QUESTIONING" ? answers : {},
        current_suggestions: session.phase === "SUGGESTING" ? selectedSuggestions : [],
        modification_suggestion: modificationSuggestion
      };

      const res = await fetch('/api/refine/finalize', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(finalizeData),
      });

      if (!res.ok) {
        const errorText = await res.text();
        throw new Error(`Failed to finalize: ${res.status} ${res.statusText} - ${errorText}`);
      }

      const data: FinalizeResult = await res.json();
      setFinalizeResult(data);
      setModificationSuggestion(""); // Clear modification suggestion after successful finalize
    } catch (e: any) {
      setToastMessage(e.message);
      setShowToast(true);
    } finally {
      setIsLoading(false);
    }
  };



  const renderContent = () => {
    if (isLoading) {
      return <div className="spinner-border text-primary" role="status"><span className="visually-hidden">Loading...</span></div>;
    }
    if (!session) {
      return (
        <form onSubmit={handleSubmit}>
          <div className="mb-3">
            <label htmlFor="initialUserStory" className="form-label">初始用戶故事 (Initial User Story)</label>
            <textarea
              id="initialUserStory"
              className="form-control"
              rows={5}
              value={initialUserStory}
              onChange={(e) => setInitialUserStory(e.target.value)}
              required
            />
          </div>
          <div className="mb-3">
            <label className="form-label">參與角色（可複選）</label>
            <div className="d-flex flex-wrap gap-3">
              {appConfig && Object.keys(appConfig.role_prompts).map(role => (
                <div key={role} className="form-check">
                  <input
                    className="form-check-input"
                    type="checkbox"
                    id={`role-checkbox-${role}`}
                    checked={selectedRoles.includes(role)}
                    onChange={e => {
                      setSelectedRoles(prev =>
                        e.target.checked
                          ? [...prev, role]
                          : prev.filter(r => r !== role)
                      );
                    }}
                  />
                  <label className="form-check-label" htmlFor={`role-checkbox-${role}`}>{role}</label>
                </div>
              ))}
            </div>
          </div>
          <button type="submit" className="btn btn-primary" disabled={isLoading || selectedRoles.length === 0}>
            開始打磨
          </button>
        </form>
      );
    }

    // Render based on phase
    switch (session.phase) {
      case "QUESTIONING":
        // 依 role 分組
        const groupedQuestions = session.questions
          ? session.questions.reduce((acc, q) => {
            if (!acc[q.role]) acc[q.role] = [];
            acc[q.role].push(q);
            return acc;
          }, {} as { [role: string]: Question[] })
          : {};
        return (
          <div>
            <h4>目前的用戶故事：</h4>
            <p className="lead">{session.user_story}</p>
            <hr />
            <h4>各角色問題：</h4>
            {Object.entries(groupedQuestions).map(([role, questions]) => (
              <div key={role} className="mb-4 p-3 border rounded bg-light">
                <h5 className="mb-3">{role}</h5>
                {questions.map((question, idx) => (
                  <div key={idx} className="mb-2">
                    {question.prompt.map((p, i) => (
                      <div key={i} className="mb-1">
                        <div>{p}</div>
                        <textarea
                          className="form-control mt-1"
                          rows={2}
                          placeholder="請在此輸入您的回答..."
                          value={answers[question.role + '_' + p] || ''}
                          onChange={(e) => handleAnswerChange(question.role, p, e.target.value)}
                        />
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            ))}
            {/* 補充資訊區域 */}
            <div className="mt-4 p-3 border rounded bg-light">
              <h5>補充資訊</h5>
              <p className="text-muted mb-2">您可以在這裡補充任何額外的資訊，這些資訊會影響下一輪的提問或建議</p>
              <textarea
                className="form-control"
                rows={3}
                placeholder="請輸入補充資訊..."
                value={additionalInfo}
                onChange={(e) => setAdditionalInfo(e.target.value)}
              />
            </div>

            {session.questions && session.questions.length > 0 && (
              <div className="d-flex justify-content-between mt-3">
                <button
                  className="btn btn-info"
                  onClick={handleSubmitAnswersAndContinue}
                  disabled={isLoading}
                >
                  提交回答，繼續提問
                </button>
                <button
                  className="btn btn-success"
                  onClick={handleSubmitAnswersAndGetSuggestions}
                  disabled={isLoading}
                >
                  提交回答，獲取建議
                </button>
              </div>
            )}
            {/* 新增產生打磨結果按鈕 */}
            <div className="mt-3">
              <button
                className="btn btn-warning"
                onClick={handleFinalize}
                disabled={isLoading}
              >
                產生打磨結果
              </button>
            </div>
          </div>
        );
      case "SUGGESTING":
        // 依 role 分組
        const groupedSuggestions = session.suggestions
          ? session.suggestions.reduce((acc, s) => {
            if (!acc[s.role]) acc[s.role] = [];
            acc[s.role].push(s);
            return acc;
          }, {} as { [role: string]: Suggestion[] })
          : {};
        return (
          <div>
            <h4>目前的用戶故事：</h4>
            <p className="lead">{session.user_story}</p>
            <hr />
            <h4>各角色建議：</h4>
            {Object.entries(groupedSuggestions).map(([role, suggestions]) => (
              <div key={role} className="mb-4 p-3 border rounded bg-light">
                <h5 className="mb-3">{role}</h5>
                {suggestions.map((suggestion, idx) => (
                  <div key={idx} className="mb-2">
                    {suggestion.prompt.map((p, i) => (
                      <div key={i} className="form-check mb-1">
                        <input
                          className="form-check-input"
                          type="checkbox"
                          value={suggestion.role + '_' + p}
                          id={`suggestion-${role}-${idx}-${i}`}
                          onChange={() => handleSuggestionChange(suggestion.role + '_' + p)}
                          checked={selectedSuggestions.includes(suggestion.role + '_' + p)}
                        />
                        <label className="form-check-label" htmlFor={`suggestion-${role}-${idx}-${i}`}>
                          {p}
                        </label>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            ))}
            {/* 補充資訊區域 */}
            <div className="mt-4 p-3 border rounded bg-light">
              <h5>補充資訊</h5>
              <p className="text-muted mb-2">您可以在這裡補充任何額外的資訊，這些資訊會影響下一輪的建議</p>
              <textarea
                className="form-control"
                rows={3}
                placeholder="請輸入補充資訊..."
                value={additionalInfo}
                onChange={(e) => setAdditionalInfo(e.target.value)}
              />
            </div>

            {session.suggestions && session.suggestions.length > 0 && (
              <div className="d-flex gap-3 justify-content-end mt-3">
                <button
                  className="btn btn-primary"
                  onClick={() => handleAcceptSuggestionsWithPhase("questioning")}
                  disabled={isLoading}
                >
                  採納選定建議，進入下一輪打磨
                </button>
                <button
                  className="btn btn-success"
                  onClick={() => handleAcceptSuggestionsWithPhase("suggesting")}
                  disabled={isLoading}
                >
                  採納選定建議，直接獲取下一輪建議
                </button>
              </div>
            )}
            {/* 新增產生打磨結果按鈕 */}
            <div className="mt-3">
              <button
                className="btn btn-warning"
                onClick={handleFinalize}
                disabled={isLoading}
              >
                產生打磨結果
              </button>
            </div>
            {session.phase === "SUGGESTING" && previousResult && previousResult.length > 0 && (
              <div className="alert alert-info mt-4">
                <h5>上一輪已採納建議：</h5>
                <ul>
                  {previousResult.map((s, idx) => (
                    s.prompt.map((p, i) => (
                      <li key={s.role + '-' + idx + '-' + i}><strong>{s.role}：</strong>{p}</li>
                    ))
                  ))}
                </ul>
              </div>
            )}
          </div>
        );
      case "FINALIZING":
        return (
          <div>
            <h4>最終規格書：</h4>
            <p>此處將顯示最終的規格書內容。</p>
            {/* TODO: Display final specification */}
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <div className="container mt-5">
      {showToast && toastMessage && (
        <ErrorModal message={toastMessage} onClose={() => setShowToast(false)} />
      )}
      <div className="d-flex justify-content-between align-items-center mb-4">
        <h1 className="mb-0">sofa-commander</h1>
        <button
          className="btn btn-outline-secondary"
          onClick={() => setIsSettingsOpen(!isSettingsOpen)}
        >
          <i className="bi bi-gear"></i> 設定
        </button>
      </div>

      {isSettingsOpen && appConfig && (
        <div className="card mb-4">
          <div className="card-header">應用程式設定</div>
          <div className="card-body">
            <div className="mb-3">
              <label htmlFor="productContext" className="form-label">產品背景 (Product Context)</label>
              <textarea
                id="productContext"
                className="form-control"
                rows={5}
                value={appConfig.product_context}
                onChange={(e) => setAppConfig(prev => prev ? { ...prev, product_context: e.target.value } : null)}
              />
            </div>
            <hr />
            <h4>角色提示詞配置</h4>
            {Object.entries(appConfig.role_prompts).map(([role, prompt]) => (
              <div key={role} className="mb-3">
                <label htmlFor={`role-${role}`} className="form-label">{role} 提示詞</label>
                <textarea
                  id={`role-${role}`}
                  className="form-control"
                  rows={3}
                  value={prompt}
                  onChange={(e) => setAppConfig(prev => prev ? { ...prev, role_prompts: { ...prev.role_prompts, [role]: e.target.value } } : null)}
                />
              </div>
            ))}
            <div className="d-flex justify-content-end">
              <button
                className="btn btn-primary"
                onClick={async () => {
                  setIsLoading(true);
                  try {
                    const res = await fetch('/api/config/app', {
                      method: 'POST',
                      headers: {
                        'Content-Type': 'application/json',
                      },
                      body: JSON.stringify(appConfig),
                    });
                    if (!res.ok) {
                      throw new Error(`Failed to save app config: ${res.status} ${res.statusText}`);
                    }
                    alert('應用程式設定已保存！');
                    setIsSettingsOpen(false); // 儲存成功後自動關閉設定區塊
                  } catch (e: any) {
                    console.error("Error saving app config:", e);
                    setToastMessage(`Error saving app config: ${e.message}`);
                    setShowToast(true);
                  } finally {
                    setIsLoading(false);
                  }
                }}
                disabled={isLoading}
              >
                保存設定
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="row">
        <div className="col-md-6">
          <h2>輸入</h2>
          {renderContent()}
        </div>
        <div className="col-md-6">
          <h2>打磨結果</h2>
          {/* 顯示 finalizeResult */}
          {finalizeResult && (
            <div className="card">
              <div className="card-header">
                <h5 className="mb-0">最終用戶故事與驗收標準</h5>
              </div>
              <div className="card-body">
                <h6>用戶故事：</h6>
                <p className="mb-3">{finalizeResult.user_story}</p>

                <h6>驗收標準 (AC)：</h6>
                {finalizeResult.ac && finalizeResult.ac.length > 0 ? (
                  <ul>
                    {finalizeResult.ac.map((ac, index) => (
                      <li key={index}>{ac}</li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-muted">無驗收標準</p>
                )}

                <details className="mt-3">
                  <summary>查看原始 AI 回應</summary>
                  <pre className="mt-2 p-2 bg-light" style={{ fontSize: '0.8em', whiteSpace: 'pre-wrap' }}>
                    {finalizeResult.raw_ai_response}
                  </pre>
                </details>

                {/* 修改建議區域 */}
                <div className="mt-4 pt-3 border-top">
                  <h6>修改建議</h6>
                  <p className="text-muted mb-2">您可以對這個結果提出修改建議，然後重新產生打磨結果</p>
                  <textarea
                    className="form-control mb-2"
                    rows={3}
                    placeholder="請輸入修改建議..."
                    value={modificationSuggestion}
                    onChange={(e) => setModificationSuggestion(e.target.value)}
                  />
                  <div className="d-flex gap-2">
                    <button
                      className="btn btn-warning btn-sm"
                      onClick={handleFinalize}
                      disabled={isLoading}
                    >
                      重新產生打磨結果
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}
          {/* Content rendered by renderContent() based on phase */}
          {session && session.phase !== "QUESTIONING" && session.phase !== "SUGGESTING" && !finalizeResult && (
            <p className="lead">{session.user_story}</p>
          )}
        </div>
      </div>
    </div>
  );
};

export default RefinementPage;
