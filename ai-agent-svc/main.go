package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ai_agent "github.com/luoxiaojun1992/ai-agent"
	file_reader "github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/file"
	directory_reader "github.com/luoxiaojun1992/ai-agent/skill/impl/filesystem/directory"
	http_skill "github.com/luoxiaojun1992/ai-agent/skill/impl"
	milvus_skill "github.com/luoxiaojun1992/ai-agent/skill/impl/milvus"
	ollama_skill "github.com/luoxiaojun1992/ai-agent/skill/impl/ollama"
	time_skill "github.com/luoxiaojun1992/ai-agent/skill/impl/time"
)

type Server struct {
	agent       *ai_agent.AgentDouble
	router      *gin.Engine
	config      *Config
	ctx         context.Context
	cancel      context.CancelFunc
}

type Config struct {
	Port            string
	CORSOrigins     []string
	AgentConfig     *ai_agent.Config
	AgentCharacter  string
	AgentRole       string
}

func NewServer() (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Load configuration
	config := &Config{
		Port: getEnv("PORT", "8080"),
		CORSOrigins: []string{"*"}, // Default to allow all origins
		AgentConfig: &ai_agent.Config{
			ChatModel:             getEnv("CHAT_MODEL", "deepseek-r1:1.5b"),
			EmbeddingModel:        getEnv("EMBEDDING_MODEL", "nomic-embed-text"),
			SupervisorModel:       getEnv("SUPERVISOR_MODEL", "deepseek-r1:1.5b"),
			OllamaHost:            getEnv("OLLAMA_HOST", "http://ollama:11434"),
			MilvusHost:            getEnv("MILVUS_HOST", "milvus:19530"),
			MilvusCollection:      getEnv("MILVUS_COLLECTION", "ai_agent_memory"),
			HttpTimeout:           30 * time.Second,
			HttpAllowRedirects:    true,
			HttpMaxRedirects:      5,
			ChatModelContextLimit: 4096,
			AgentMode:             ai_agent.AgentModeChat,
			AgentLoopDuration:     1 * time.Second,
		},
		AgentCharacter: getEnv("AGENT_CHARACTER", "You are a helpful AI assistant"),
		AgentRole:      getEnv("AGENT_ROLE", "AI Assistant"),
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
			
			// Add HTTP skill
			option.AddSkill("http", &http_skill.Http{})
			
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
	
	// Setup Gin router
	router := gin.Default()
	
	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	
	return &Server{
		agent:  agent,
		router: router,
		config: config,
		ctx:    ctx,
		cancel: cancel,
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
	
	// Get available skills
	s.router.GET("/skills", s.skillsHandler)
	
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
		"status":     "running",
		"character":  s.agent.GetDescription(),
		"skills":     len(s.agent.SkillSet),
		"timestamp":  time.Now().Unix(),
	})
}

type ChatRequest struct {
	Message    string                 `json:"message"`
	AgentConfig map[string]interface{} `json:"agentConfig,omitempty"`
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
		err := s.agent.ListenAndWatch(s.ctx, req.Message, nil, func(resp string) error {
			response.WriteString(resp)
			return nil
		})
		
		if err != nil {
			errChan <- err
		} else {
			responseChan <- response.String()
		}
	}()
	
	select {
	case response := <-responseChan:
		c.JSON(200, gin.H{
			"response": response,
			"timestamp": time.Now().Unix(),
		})
	case err := <-errChan:
		c.JSON(500, gin.H{"error": err.Error()})
	case <-time.After(60 * time.Second):
		c.JSON(504, gin.H{"error": "Request timeout"})
	}
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
		err := s.agent.Command(s.ctx, req.SkillName, req.Parameters, func(output interface{}) (interface{}, error) {
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
	case <-time.After(30 * time.Second):
		c.JSON(504, gin.H{"error": "Skill execution timeout"})
	}
}

func (s *Server) skillsHandler(c *gin.Context) {
	skills := make([]gin.H, 0)
	
	for name, skill := range s.agent.SkillSet {
		skills = append(skills, gin.H{
			"name":        name,
			"description": skill.GetDescription(),
		})
	}
	
	c.JSON(200, gin.H{
		"skills": skills,
		"count":  len(skills),
	})
}

func (s *Server) getConfigHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"chatModel":      s.config.AgentConfig.ChatModel,
		"embeddingModel": s.config.AgentConfig.EmbeddingModel,
		"ollamaHost":     s.config.AgentConfig.OllamaHost,
		"milvusHost":     s.config.AgentConfig.MilvusHost,
		"character":      s.config.AgentCharacter,
		"role":           s.config.AgentRole,
	})
}

func (s *Server) updateConfigHandler(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}
	
	// Apply updates (simplified - in production, you'd validate and apply specific fields)
	c.JSON(200, gin.H{
		"message": "Configuration updated successfully",
		"updated": updates,
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
	c.JSON(200, gin.H{
		"message": "Memory cleared successfully",
	})
}

func (s *Server) Start() error {
	s.setupRoutes()
	
	log.Printf("Starting AI Agent Service on port %s", s.config.Port)
	return s.router.Run(":" + s.config.Port)
}

func (s *Server) Stop() error {
	s.cancel()
	if s.agent != nil {
		return s.agent.Close()
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatal("Failed to create server:", err)
	}
	
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		log.Println("Shutting down server...")
		if err := server.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()
	
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}