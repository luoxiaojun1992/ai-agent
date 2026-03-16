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

func TestCompressor_FoldNearDuplicateMixedLanguage(t *testing.T) {
	compressor := NewCompressor(Config{
		BudgetTokens:           35,
		ReserveTokens:          0,
		Model:                  "qwen3:4b",
		NearDuplicateThreshold: 0.75,
	})

	input := []Message{
		{Role: "system", Content: "保持关键约束", Protected: true},
		{Role: "assistant", Content: "请帮我总结 this API design and keep all constraints"},
		{Role: "assistant", Content: "请帮我总结 this api design and keep all constraints."},
		{Role: "user", Content: "继续", Protected: true},
	}

	output := compressor.Compress(input)
	if len(output) >= len(input) {
		t.Fatalf("expected mixed-language near-duplicates to be folded, input=%d output=%d", len(input), len(output))
	}
}

func TestCompressor_FoldNearDuplicateCodeSnippets(t *testing.T) {
	compressor := NewCompressor(Config{
		BudgetTokens:           22,
		ReserveTokens:          0,
		Model:                  "",
		NearDuplicateThreshold: 0.70,
	})

	input := []Message{
		{Role: "assistant", Content: "func add(a int, b int) int { return a + b }"},
		{Role: "assistant", Content: "func add(a int,b int) int { return a+b }"},
		{Role: "assistant", Content: "func sub(a int, b int) int { return a - b }"},
	}

	output := compressor.Compress(input)
	if len(output) >= len(input) {
		t.Fatalf("expected similar code snippets to be folded, input=%d output=%d", len(input), len(output))
	}

	foundSub := false
	for _, msg := range output {
		if msg.Content == "func sub(a int, b int) int { return a - b }" {
			foundSub = true
			break
		}
	}
	if !foundSub {
		t.Fatalf("expected non-duplicate code snippet to be preserved")
	}
}

func TestCompressor_DefaultsAndZeroBudget(t *testing.T) {
	compressor := NewCompressor(Config{BudgetTokens: 0, ReserveTokens: -1})
	input := []Message{{Role: "assistant", Content: "a"}}
	output := compressor.Compress(input)
	if len(output) != 1 {
		t.Fatalf("expected single message unchanged")
	}
}

func TestTokenSetJaccard_EmptyAndEqual(t *testing.T) {
	if v := tokenSetJaccard("", ""); v != 1 {
		t.Fatalf("expected 1 for equal empty, got %v", v)
	}
	if v := tokenSetJaccard("", "a"); v != 0 {
		t.Fatalf("expected 0 for empty vs non-empty, got %v", v)
	}
}

func TestNormalizeAndCountBranches(t *testing.T) {
	if normalizeContent("   ") != "" {
		t.Fatalf("expected whitespace-only input to normalize to empty")
	}
	estimator := newTokenEstimator("qwen3:4b")
	if n := estimator.count("verylongtoken123456 another"); n <= 0 {
		t.Fatalf("expected positive token count")
	}
}

func TestTokenEstimator_Count_EmptyAndNoMatch(t *testing.T) {
	e := newTokenEstimator("")
	// empty text → return 0
	if n := e.count(""); n != 0 {
		t.Fatalf("expected 0 for empty text, got %d", n)
	}
	// whitespace-only: regex finds no tokens → len([]rune)/4+1
	if n := e.count("   "); n <= 0 {
		t.Fatalf("expected positive count for whitespace-only text (no token matches), got %d", n)
	}
}

func TestTokenSetJaccard_WhitespaceTokenSets(t *testing.T) {
	// whitespace-only strings produce empty token sets
	if v := tokenSetJaccard("   ", "hello"); v != 0 {
		t.Fatalf("expected 0 for whitespace vs text token sets, got %v", v)
	}
	if v := tokenSetJaccard("   ", "   "); v != 1 {
		t.Fatalf("expected 1 for equal whitespace-only strings, got %v", v)
	}
}

func TestDropOldestRemovable_ZeroBudget(t *testing.T) {
	c := NewCompressor(Config{BudgetTokens: 10})
	msgs := []Message{{Role: "user", Content: "hello"}}
	result := c.dropOldestRemovable(msgs, 0)
	if len(result) != len(msgs) {
		t.Fatalf("expected messages unchanged for zero budget, got len=%d", len(result))
	}
}

func TestDropOldestRemovable_AllProtected(t *testing.T) {
	c := NewCompressor(Config{BudgetTokens: 2})
	msgs := []Message{
		{Role: "system", Content: "important system instruction", Protected: true},
		{Role: "user", Content: "important user message", Protected: true},
	}
	// All messages are protected; cannot remove any despite being over budget
	result := c.dropOldestRemovable(msgs, 1)
	if len(result) != 2 {
		t.Fatalf("expected all protected messages kept, got len=%d", len(result))
	}
}
