package contextcompress

import "testing"

func TestCompressor_FoldExactAndNearDuplicates(t *testing.T) {
	compressor := NewCompressor(Config{
		BudgetTokens:           24,
		ReserveTokens:          0,
		Model:                  "",
		NearDuplicateThreshold: 0.80,
	})

	input := []Message{
		{Role: "system", Content: "fixed instruction", Protected: true},
		{Role: "assistant", Content: "the weather in beijing is sunny today"},
		{Role: "assistant", Content: "the weather in beijing is sunny today"},
		{Role: "assistant", Content: "today beijing weather is sunny"},
		{Role: "user", Content: "thanks", Protected: true},
	}

	output := compressor.Compress(input)

	if len(output) >= len(input) {
		t.Fatalf("expected compressed output to be smaller than input, got input=%d output=%d", len(input), len(output))
	}
}

func TestCompressor_RespectProtectedMessagesUnderBudgetPressure(t *testing.T) {
	compressor := NewCompressor(Config{
		BudgetTokens:           25,
		ReserveTokens:          0,
		Model:                  "",
		NearDuplicateThreshold: 0.90,
	})

	input := []Message{
		{Role: "system", Content: "always keep this instruction", Protected: true},
		{Role: "assistant", Content: "this is an old and verbose memory chunk that should be removed first"},
		{Role: "assistant", Content: "another removable chunk with duplicated information"},
		{Role: "user", Content: "latest user request must be kept", Protected: true},
	}

	output := compressor.Compress(input)

	if len(output) < 2 {
		t.Fatalf("expected protected messages to remain, got len=%d", len(output))
	}

	foundSystem := false
	foundUser := false
	for _, msg := range output {
		if msg.Role == "system" && msg.Content == "always keep this instruction" {
			foundSystem = true
		}
		if msg.Role == "user" && msg.Content == "latest user request must be kept" {
			foundUser = true
		}
	}

	if !foundSystem || !foundUser {
		t.Fatalf("expected protected system/user messages to be preserved")
	}
}
