package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthAndStatus(t *testing.T) {
	s := newState()
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	healthResp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected health status: %d", healthResp.StatusCode)
	}

	statusResp, err := http.Get(ts.URL + "/status")
	if err != nil {
		t.Fatalf("status request failed: %v", err)
	}
	defer statusResp.Body.Close()
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", statusResp.StatusCode)
	}
}

func TestChatNonStreamAndMemory(t *testing.T) {
	s := newState()
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	payload := map[string]interface{}{"message": "hello mock"}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(ts.URL+"/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("chat request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected chat status: %d", resp.StatusCode)
	}

	memResp, err := http.Get(ts.URL + "/memory")
	if err != nil {
		t.Fatalf("memory request failed: %v", err)
	}
	defer memResp.Body.Close()

	var mem map[string]interface{}
	if err := json.NewDecoder(memResp.Body).Decode(&mem); err != nil {
		t.Fatalf("decode memory failed: %v", err)
	}
	length, ok := mem["length"].(float64)
	if !ok || length < 1 {
		t.Fatalf("unexpected memory length: %#v", mem["length"])
	}
}

func TestChatStream(t *testing.T) {
	s := newState()
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	reqBody, _ := json.Marshal(map[string]interface{}{
		"message": "stream message",
		"stream":  true,
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/chat", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream chat request failed: %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("unexpected content-type: %s", resp.Header.Get("Content-Type"))
	}

	scanner := bufio.NewScanner(resp.Body)
	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	got := content.String()
	if !strings.Contains(got, "event: message") || !strings.Contains(got, "event: complete") {
		t.Fatalf("unexpected stream body: %s", got)
	}
}

func TestConfigAndSkillAndClearMemory(t *testing.T) {
	s := newState()
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	cfgBody, _ := json.Marshal(map[string]interface{}{"chatModel": "new-mock-model"})
	putReq, _ := http.NewRequest(http.MethodPut, ts.URL+"/config", bytes.NewReader(cfgBody))
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		t.Fatalf("put config failed: %v", err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected put config status: %d", putResp.StatusCode)
	}

	getCfgResp, err := http.Get(ts.URL + "/config")
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	defer getCfgResp.Body.Close()
	var cfg map[string]interface{}
	_ = json.NewDecoder(getCfgResp.Body).Decode(&cfg)
	if cfg["chatModel"] != "new-mock-model" {
		t.Fatalf("chatModel not updated: %#v", cfg["chatModel"])
	}

	skillBody, _ := json.Marshal(map[string]interface{}{
		"skillName":  "sleep",
		"parameters": map[string]interface{}{"duration": "1s"},
	})
	skillResp, err := http.Post(ts.URL+"/skill", "application/json", bytes.NewReader(skillBody))
	if err != nil {
		t.Fatalf("skill request failed: %v", err)
	}
	defer skillResp.Body.Close()
	if skillResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected skill status: %d", skillResp.StatusCode)
	}

	delReq, _ := http.NewRequest(http.MethodDelete, ts.URL+"/memory", nil)
	delResp, err := http.DefaultClient.Do(delReq)
	if err != nil {
		t.Fatalf("delete memory failed: %v", err)
	}
	defer delResp.Body.Close()
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected delete memory status: %d", delResp.StatusCode)
	}
}
