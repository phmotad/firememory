package contract

type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Permissions struct {
	AllowWrite           bool `json:"allow_write"`
	RequiresConfirmation bool `json:"requires_confirmation"`
}

type Thresholds struct {
	TopK                int     `json:"top_k"`
	SimilarityThreshold float64 `json:"similarity_threshold"`
	BudgetTokens        int     `json:"budget_tokens"`
}

type Options struct {
	IncludeGraph bool `json:"include_graph"`
	IncludeTrace bool `json:"include_trace"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ExternalRequest struct {
	Version   string         `json:"version"`
	RequestID string         `json:"request_id"`
	Language  string         `json:"language,omitempty"`
	Actor     Actor          `json:"actor"`
	Operation string         `json:"operation"`
	Brain     string         `json:"brain"`
	Input     map[string]any `json:"input"`
}

type ExternalResponse struct {
	OK        bool           `json:"ok"`
	RequestID string         `json:"request_id"`
	Operation string         `json:"operation,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
	Trace     map[string]any `json:"trace,omitempty"`
	Rejected  bool           `json:"rejected,omitempty"`
	Error     *Error         `json:"error,omitempty"`
}

type OperationRequest struct {
	Version     string         `json:"version"`
	RequestID   string         `json:"request_id"`
	Language    string         `json:"language"`
	Actor       Actor          `json:"actor"`
	Operation   string         `json:"operation"`
	Intent      string         `json:"intent"`
	Brain       string         `json:"brain"`
	Scope       string         `json:"scope"`
	Input       map[string]any `json:"input"`
	Permissions *Permissions   `json:"permissions"`
	Thresholds  *Thresholds    `json:"thresholds,omitempty"`
	Options     *Options       `json:"options,omitempty"`
}

type OperationResponse = ExternalResponse

type InternalRequest = OperationRequest
