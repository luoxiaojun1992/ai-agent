package mcp

import "testing"

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
