package mcp

import (
	"context"
	"testing"
)

func TestNewMcpClient_InvalidType(t *testing.T) {
	_, err := newMcpClient(&Config{Host: "http://localhost:1234", ClientType: ClientType("invalid")})
	if err == nil {
		t.Fatalf("expected invalid client type error")
	}
}

func TestNewMcpClient_ValidTypes(t *testing.T) {
	if c, err := newMcpClient(&Config{Host: "http://localhost:1234", ClientType: ClientTypeSSE}); err != nil || c == nil {
		t.Fatalf("expected sse client, err=%v", err)
	}
	if c, err := newMcpClient(&Config{Host: "http://localhost:1234", ClientType: ClientTypeStream}); err != nil || c == nil {
		t.Fatalf("expected stream client, err=%v", err)
	}
}

func TestNewClient_InvalidType(t *testing.T) {
	_, err := NewClient(&Config{Host: "http://localhost:1234", ClientType: ClientType("bad")})
	if err == nil {
		t.Fatalf("expected constructor error")
	}
}

func TestClient_Methods_ErrorPaths(t *testing.T) {
	client, err := NewClient(&Config{Host: "http://127.0.0.1:1", ClientType: ClientTypeSSE})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if err := client.Initialize(context.Background()); err == nil {
		t.Fatalf("expected initialize error on unavailable endpoint")
	}

	if _, err := client.ListTools(context.Background()); err == nil {
		t.Fatalf("expected list tools error before init")
	}

	if _, err := client.CallTool(context.Background(), "x", map[string]interface{}{}); err == nil {
		t.Fatalf("expected call tool error before init")
	}

	if err := client.Close(); err == nil {
		// Close may return error depending on internal state; both non-nil and nil are acceptable.
	}
}
