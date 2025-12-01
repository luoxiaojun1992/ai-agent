package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	ai_agent "github.com/luoxiaojun1992/ai-agent"
	mcpClient "github.com/luoxiaojun1992/ai-agent/pkg/mcp"
	skillSet "github.com/luoxiaojun1992/ai-agent/skill/impl"
	directory_reader "github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/directory"
	file_reader "github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/file"
	time_skill "github.com/luoxiaojun1992/ai-agent/skill/impl/time"
)

type Server struct {
	agent              *ai_agent.AgentDouble
	router             *gin.Engine
	config             *Config
	ctx                context.Context
	cancel             context.CancelFunc
	mcpWebSearchClient *mcpClient.Client
	mcpContext7Client  *mcpClient.Client
}

type Config struct {
	Port           string
	CORSOrigins    []string
	AgentConfig    *ai_agent.Config
	AgentCharacter string
	AgentRole      string
}

func addToolSampleMemories(ad *ai_agent.AgentDouble) {
	ad.AddUserMemory("Please tell me what's the weather like today", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_web_search","context":{"name":"search","arguments":{"query":"what's the weather like today"}}}</tool>`, nil).
		AddUserMemory("What's the weather like today", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_web_search","context":{"name":"search","arguments":{"query":"what's the weather like today"}}}</tool>`, nil).
		AddUserMemory("What's AI", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_web_search","context":{"name":"search","arguments":{"query":"What's AI"}}}</tool>`, nil).
		AddUserMemory("AI", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_web_search","context":{"name":"search","arguments":{"query":"AI"}}}</tool>`, nil).
		AddUserMemory("weather", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_web_search","context":{"name":"search","arguments":{"query":"weather"}}}</tool>`, nil).
		AddUserMemory("search weather", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_web_search","context":{"name":"search","arguments":{"query":"weather"}}}</tool>`, nil).
		AddUserMemory("sleep", nil).
		AddAssistantMemory(`<tool>{"function":"sleep","context":{"duration":"1s"}}}</tool>`, nil).
		AddUserMemory("how to use mongodb", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_context_7","context":{"name":"resolve-library-id","arguments":{"libraryName":"mongodb"}}}</tool>`, nil).
		AddToolMemory("/mongodb/docs", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_context_7","context":{"name":"get-library-docs","arguments":{"context7CompatibleLibraryID":"/mongodb/docs"}}}</tool>`, nil).
		AddUserMemory("how to use next.js", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_context_7","context":{"name":"resolve-library-id","arguments":{"libraryName":"next.js"}}}</tool>`, nil).
		AddToolMemory("/vercel/next.js", nil).
		AddAssistantMemory(`<tool>{"function":"mcp_context_7","context":{"name":"get-library-docs","arguments":{"context7CompatibleLibraryID":"/vercel/next.js"}}}</tool>`, nil)
}

func NewServer() (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Load configuration
	config := &Config{
		Port:        getEnv("PORT", "8080"),
		CORSOrigins: []string{"*"}, // Default to allow all origins
		AgentConfig: &ai_agent.Config{
			ChatModel:             getEnv("CHAT_MODEL", "qwen3:0.6b"),
			EmbeddingModel:        getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
			SupervisorModel:       getEnv("SUPERVISOR_MODEL", "qwen3:0.6b"),
			ModelTemperature:      getFloat32Env("MODEL_TEMPERATURE", 0.1),
			SupervisorSwitch:      getBoolEnv("SUPERVISOR_SWITCH", false),
			OllamaHost:            getEnv("OLLAMA_HOST", "http://ollama:11434"),
			MilvusHost:            getEnv("MILVUS_HOST", "milvus:19530"),
			MilvusCollection:      getEnv("MILVUS_COLLECTION", "ai_agent_memory"),
			HttpTimeout:           30 * time.Second,
			HttpAllowRedirects:    true,
			HttpMaxRedirects:      5,
			ChatModelContextLimit: 1000000,
			AgentMode:             ai_agent.AgentMode(getEnv("AGENT_MODE", string(ai_agent.AgentModeChat))),
			AgentLoopDuration:     1 * time.Second,
		},
		AgentCharacter: getEnv("AGENT_CHARACTER", "I am a helpful AI assistant."),
		AgentRole:      getEnv("AGENT_ROLE", "AI Assistant"),
	}

	mcpWebSearchClient, err := mcpClient.NewClient(&mcpClient.Config{
		Host:       getEnv("MCP_WEB_SEARCH_HOST", "http://mcp-web-search:3000"),
		ClientType: mcpClient.ClientTypeSSE,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	if err := mcpWebSearchClient.Initialize(ctx); err != nil {
		cancel()
		return nil, err
	}

	mcpContext7Client, err := mcpClient.NewClient(&mcpClient.Config{
		Host:       getEnv("MCP_CONTEXT_7_CLIENT_HOST", "http://mcp-context-7:8080"),
		ClientType: mcpClient.ClientTypeStream,
	})
	if err != nil {
		cancel()
		return nil, err
	}
	if err := mcpContext7Client.Initialize(ctx); err != nil {
		cancel()
		return nil, err
	}

	// Create agent with skills
	agent, err := ai_agent.NewAgentDouble(ctx,
		func(option *ai_agent.AgentDoubleOption) {
			option.SetConfig(config.AgentConfig)
			option.SetCharacter(config.AgentCharacter)
			option.SetRole(config.AgentRole)

			// Add filesystem skills
			option.AddSkill("file_reader", &file_reader.Reader{RootDir: "/tmp/agent"})
			option.AddSkill("file_writer", &file_reader.Writer{RootDir: "/tmp/agent"})
			option.AddSkill("file_remover", &file_reader.Remover{RootDir: "/tmp/agent"})
			option.AddSkill("directory_reader", &directory_reader.Reader{RootDir: "/tmp/agent"})
			option.AddSkill("directory_writer", &directory_reader.Writer{RootDir: "/tmp/agent"})
			option.AddSkill("directory_remover", &directory_reader.Remover{RootDir: "/tmp/agent"})

			// Add MCP skills
			option.AddSkill("mcp_web_search", &skillSet.MCP{MCPClient: mcpWebSearchClient})
			option.AddSkill("mcp_context_7", &skillSet.MCP{MCPClient: mcpContext7Client})

			// Add time skills
			option.AddSkill("sleep", &time_skill.Sleep{})
		},
	)

	if err != nil {
		cancel()
		return nil, err
	}

	// Initialize memory
	agent.InitMemory()
	addToolSampleMemories(agent)

	// Setup Gin router
	router := gin.Default()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "X-Stream-Mode"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	return &Server{
		agent:              agent,
		router:             router,
		config:             config,
		ctx:                ctx,
		cancel:             cancel,
		mcpWebSearchClient: mcpWebSearchClient,
		mcpContext7Client:  mcpContext7Client,
	}, nil
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", s.healthHandler)

	// Agent status
	s.router.GET("/status", s.statusHandler)

	// Chat with agent
	s.router.POST("/chat", s.chatHandler)

	// Execute skill
	s.router.POST("/skill", s.skillHandler)

	// Configuration
	s.router.GET("/config", s.getConfigHandler)
	s.router.PUT("/config", s.updateConfigHandler)

	// Memory operations
	s.router.GET("/memory", s.getMemoryHandler)
	s.router.DELETE("/memory", s.clearMemoryHandler)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) statusHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "running",
		"character": s.agent.GetDescription(),
		"timestamp": time.Now().Unix(),
	})
}

type ChatRequest struct {
	Message     string                 `json:"message"`
	AgentConfig map[string]interface{} `json:"agentConfig,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
}

func (s *Server) chatHandler(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	if req.Message == "" {
		c.JSON(400, gin.H{"error": "Message is required"})
		return
	}

	// Check if stream mode is requested
	if req.Stream {
		s.handleStreamChat(c, req.Message)
		return
	}

	// Original blocking mode
	// Create a channel to collect the response
	responseChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in agent response: %v", r)
			}
		}()

		var response strings.Builder
		err := s.agent.ListenAndWatch(c.Request.Context(), req.Message, nil, func(resp string) error {
			response.WriteString(resp)
			return nil
		})

		if err != nil {
			log.Println("Error during agent response", err)
			errChan <- err
		} else {
			responseChan <- response.String()
		}
	}()

	select {
	case response := <-responseChan:
		c.JSON(200, gin.H{
			"response":  response,
			"timestamp": time.Now().Unix(),
		})
	case err := <-errChan:
		c.JSON(500, gin.H{"error": err.Error()})
	case <-time.After(600 * time.Second):
		c.JSON(504, gin.H{"error": "Request timeout"})
	}
}

func (s *Server) handleStreamChat(c *gin.Context, message string) {
	// Set headers for SSE (Server-Sent Events)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no") // Disable proxy buffering

	// Create channels for streaming response
	streamChan := make(chan string, 100)
	errChan := make(chan error, 1)
	doneChan := make(chan bool, 1)

	// Start goroutine to handle agent response
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in agent response: %v", r)
			}
			// Close channel when done
			close(streamChan)
		}()

		// Use a for loop to continuously process callbacks
		// until ListenAndWatch completes
		err := s.agent.ListenAndWatch(c.Request.Context(), message, nil, func(resp string) error {
			// Send each chunk to the stream channel
			select {
			case streamChan <- resp:
				// Successfully sent chunk
			case <-doneChan:
				// Early termination requested
				return nil
			}
			return nil
		})

		if err != nil {
			log.Println("Error during agent response", err)
			errChan <- err
		}

		// Signal completion
		time.Sleep(time.Second)
		doneChan <- true
	}()

	// Stream response to client
	c.Stream(func(w io.Writer) bool {
		for {
			select {
			case chunk, ok := <-streamChan:
				if !ok {
					// Channel closed, send completion event
					c.SSEvent("complete", map[string]interface{}{
						"done":      true,
						"timestamp": time.Now().Unix(),
					})
					return false
				}

				// Send chunk as SSE event
				c.SSEvent("message", map[string]interface{}{
					"content":   chunk,
					"timestamp": time.Now().Unix(),
				})

				// Flush to ensure immediate delivery
				c.Writer.Flush()

			case err := <-errChan:
				// Send error event
				c.SSEvent("error", map[string]interface{}{
					"error":     err.Error(),
					"timestamp": time.Now().Unix(),
				})
				return false

			case <-doneChan:
				// Send completion event
				c.SSEvent("complete", map[string]interface{}{
					"done":      true,
					"timestamp": time.Now().Unix(),
				})
				return false

			case <-time.After(600 * time.Second):
				// Send timeout event
				c.SSEvent("error", map[string]interface{}{
					"error":     "Request timeout",
					"timestamp": time.Now().Unix(),
				})
				return false
			}
		}
	})
}

type SkillRequest struct {
	SkillName  string                 `json:"skillName"`
	Parameters map[string]interface{} `json:"parameters"`
}

func (s *Server) skillHandler(c *gin.Context) {
	var req SkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}

	if req.SkillName == "" {
		c.JSON(400, gin.H{"error": "Skill name is required"})
		return
	}

	// Execute skill
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		err := s.agent.Command(c.Request.Context(), req.SkillName, req.Parameters, func(output interface{}) (interface{}, error) {
			resultChan <- output
			return output, nil
		})

		if err != nil {
			errChan <- err
		}
	}()

	select {
	case result := <-resultChan:
		c.JSON(200, gin.H{
			"result": result,
			"skill":  req.SkillName,
		})
	case err := <-errChan:
		c.JSON(500, gin.H{"error": err.Error()})
	case <-time.After(600 * time.Second):
		c.JSON(504, gin.H{"error": "Request timeout"})
	}
}

func (s *Server) getConfigHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"chatModel":       s.config.AgentConfig.ChatModel,
		"embeddingModel":  s.config.AgentConfig.EmbeddingModel,
		"supervisorModel": s.config.AgentConfig.SupervisorModel,
		"agentMode":       s.config.AgentConfig.AgentMode,
		"character":       s.config.AgentCharacter,
		"role":            s.config.AgentRole,
	})
}

func (s *Server) updateConfigHandler(c *gin.Context) {
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": "Invalid config format"})
		return
	}

	// Update configuration (simplified for demo)
	if chatModel, ok := config["chatModel"].(string); ok {
		s.config.AgentConfig.ChatModel = chatModel
	}
	if embeddingModel, ok := config["embeddingModel"].(string); ok {
		s.config.AgentConfig.EmbeddingModel = embeddingModel
	}
	if agentMode, ok := config["agentMode"].(string); ok {
		s.config.AgentConfig.AgentMode = ai_agent.AgentMode(agentMode)
	}

	c.JSON(200, gin.H{
		"message": "Configuration updated",
		"config":  config,
	})
}

func (s *Server) getMemoryHandler(c *gin.Context) {
	memory := s.agent.MemorySnapshot()
	c.JSON(200, gin.H{
		"contexts": memory.Contexts,
		"length":   len(memory.Contexts),
	})
}

func (s *Server) clearMemoryHandler(c *gin.Context) {
	s.agent.ResetMemory()
	addToolSampleMemories(s.agent)
	c.JSON(200, gin.H{
		"message": "Memory cleared successfully",
	})
}

func (s *Server) Start() error {
	s.setupRoutes()

	// Start server
	server := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	go func() {
		log.Printf("Server starting on port %s", s.config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	select {
	case <-ctx.Done():
		log.Println("Server shutdown successfully")
	case <-time.After(30 * time.Second):
		log.Println("Timed out during stopping server")
	}

	log.Println("Server exited")

	s.agent.Agent.Close()
	s.mcpWebSearchClient.Close()
	s.mcpContext7Client.Close()
	s.cancel()

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getFloat32Env(key string, defaultValue float32) float32 {
	if value := os.Getenv(key); value != "" {
		valueFloat, err := strconv.ParseFloat(value, 32)
		if err != nil {
			log.Fatal("Error parsing environment variable", key, ":", err)
		}
		return float32(valueFloat)
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		valueBool, err := strconv.ParseBool(value)
		if err != nil {
			log.Fatal("Error parsing environment variable", key, ":", err)
		}
		return valueBool
	}
	return defaultValue
}

func main() {
	log.Println("Initializing AI Agent Service...")

	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file, using environment variables or defaults")
	}

	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}

	log.Println("Starting server")
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
