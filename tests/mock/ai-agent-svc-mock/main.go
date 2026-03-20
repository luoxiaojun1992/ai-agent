package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type chatRequest struct {
	Message     string                 `json:"message"`
	AgentConfig map[string]interface{} `json:"agentConfig,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
}

type skillRequest struct {
	SkillName  string                 `json:"skillName"`
	Parameters map[string]interface{} `json:"parameters"`
}

type state struct {
	mu      sync.RWMutex
	config  map[string]string
	memory  []string
	counter int64
}

func newState() *state {
	return &state{
		config: map[string]string{
			"chatModel":       "mock-chat-model",
			"embeddingModel":  "mock-embedding-model",
			"supervisorModel": "mock-supervisor-model",
			"agentMode":       "chat",
			"character":       "I am a mock ai-agent-svc.",
			"role":            "Mock Assistant",
		},
		memory: []string{"mock memory context 1", "mock memory context 2"},
	}
}

func (s *state) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/status", s.statusHandler)
	mux.HandleFunc("/chat", s.chatHandler)
	mux.HandleFunc("/skill", s.skillHandler)
	mux.HandleFunc("/config", s.configHandler)
	mux.HandleFunc("/memory", s.memoryHandler)
	return mux
}

func (s *state) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func (s *state) statusHandler(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	character := s.config["character"]
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "running",
		"character": character,
		"timestamp": time.Now().Unix(),
	})
}

func (s *state) chatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
		return
	}

	if req.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Message is required"})
		return
	}

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
			return
		}

		chunk := fmt.Sprintf("event: message\ndata: {\"content\":\"mock stream response: %s\",\"timestamp\":%d}\n\n", req.Message, time.Now().Unix())
		_, _ = w.Write([]byte(chunk))
		flusher.Flush()
		_, _ = w.Write([]byte(fmt.Sprintf("event: complete\ndata: {\"done\":true,\"timestamp\":%d}\n\n", time.Now().Unix())))
		flusher.Flush()
		return
	}

	s.mu.Lock()
	s.counter++
	s.memory = append(s.memory, req.Message)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"response":  "mock response: " + req.Message,
		"timestamp": time.Now().Unix(),
	})
}

func (s *state) skillHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req skillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
		return
	}
	if req.SkillName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Skill name is required"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"result": map[string]interface{}{
			"mock":       true,
			"skillName":  req.SkillName,
			"parameters": req.Parameters,
		},
		"skill": req.SkillName,
	})
}

func (s *state) configHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		defer s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"chatModel":       s.config["chatModel"],
			"embeddingModel":  s.config["embeddingModel"],
			"supervisorModel": s.config["supervisorModel"],
			"agentMode":       s.config["agentMode"],
			"character":       s.config["character"],
			"role":            s.config["role"],
		})
	case http.MethodPut:
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid config format"})
			return
		}
		s.mu.Lock()
		for k, v := range req {
			if str, ok := v.(string); ok {
				s.config[k] = str
			}
		}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "Configuration updated",
			"config":  req,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func (s *state) memoryHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.mu.RLock()
		contexts := append([]string(nil), s.memory...)
		s.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"contexts": contexts,
			"length":   len(contexts),
		})
	case http.MethodDelete:
		s.mu.Lock()
		s.memory = []string{}
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Memory cleared successfully",
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func main() {
	port := getEnv("MOCK_AI_AGENT_SVC_PORT", "18080")
	addr := ":" + port
	log.Printf("mock ai-agent-svc listening on %s", addr)
	if err := http.ListenAndServe(addr, newState().routes()); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
