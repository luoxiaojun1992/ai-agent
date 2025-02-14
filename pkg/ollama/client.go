package ollama

type EmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbedResponse struct {
	Model           string      `json:"model"`
	Embeddings      [][]float32 `json:"embeddings"`
	TotalDuration   int64       `json:"total_duration"`
	LoadDuration    int64       `json:"load_duration"`
	PromptEvalCount int64       `json:"prompt_eval_count"`
}

type ChatRequest struct {
	Model    string     `json:"model"`
	Messages []*Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamResponse struct {
	Model         string   `json:"model"`
	CreatedAt     string   `json:"created_at"`
	Message       *Message `json:"message"`
	Done          bool     `json:"done"`
	TotalDuration int64    `json:"total_duration"`
}

type IClient interface {
	embeddingPrompt(req *EmbedRequest) (*EmbedResponse, error)
	talk(req *ChatRequest, callback func(content string) error) error
}

type Client struct {
	//todo
}

//todo
