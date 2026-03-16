package contextcompress

import (
	"regexp"
	"strings"
)

// Message represents a generic context item for compression.
type Message struct {
	Role      string
	Content   string
	Images    []string
	Protected bool
}

type Config struct {
	BudgetTokens           int
	ReserveTokens          int
	Model                  string
	NearDuplicateThreshold float64
}

type Compressor struct {
	cfg       Config
	estimator *tokenEstimator
}

func NewCompressor(cfg Config) *Compressor {
	if cfg.NearDuplicateThreshold <= 0 {
		cfg.NearDuplicateThreshold = 0.92
	}
	if cfg.ReserveTokens < 0 {
		cfg.ReserveTokens = 0
	}
	return &Compressor{
		cfg:       cfg,
		estimator: newTokenEstimator(cfg.Model),
	}
}

func (c *Compressor) Compress(input []Message) []Message {
	if len(input) <= 1 {
		return cloneMessages(input)
	}

	output := cloneMessages(input)
	targetBudget := c.cfg.BudgetTokens - c.cfg.ReserveTokens
	if targetBudget <= 0 {
		targetBudget = c.cfg.BudgetTokens
	}

	if targetBudget <= 0 || c.tokenCount(output) <= targetBudget {
		return output
	}

	output = c.foldExactDuplicates(output)
	if c.tokenCount(output) <= targetBudget {
		return output
	}

	output = c.foldNearDuplicates(output)
	if c.tokenCount(output) <= targetBudget {
		return output
	}

	output = c.dropOldestRemovable(output, targetBudget)
	return output
}

func (c *Compressor) foldExactDuplicates(messages []Message) []Message {
	seen := make(map[string]struct{}, len(messages))
	kept := make([]Message, 0, len(messages))

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		key := msg.Role + "|" + normalizeContent(msg.Content)
		if msg.Protected {
			kept = append(kept, msg)
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		kept = append(kept, msg)
	}

	reverseMessages(kept)
	return kept
}

func (c *Compressor) foldNearDuplicates(messages []Message) []Message {
	kept := make([]Message, 0, len(messages))
	keptNorm := make([]string, 0, len(messages))

	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		norm := normalizeContent(msg.Content)
		if msg.Protected {
			kept = append(kept, msg)
			keptNorm = append(keptNorm, norm)
			continue
		}

		duplicated := false
		for j := range kept {
			if kept[j].Role != msg.Role {
				continue
			}
			sim := tokenSetJaccard(norm, keptNorm[j])
			if sim >= c.cfg.NearDuplicateThreshold {
				duplicated = true
				break
			}
		}
		if duplicated {
			continue
		}

		kept = append(kept, msg)
		keptNorm = append(keptNorm, norm)
	}

	reverseMessages(kept)
	return kept
}

func (c *Compressor) dropOldestRemovable(messages []Message, targetBudget int) []Message {
	if targetBudget <= 0 {
		return messages
	}

	output := cloneMessages(messages)
	for len(output) > 1 && c.tokenCount(output) > targetBudget {
		removed := false
		for i := 0; i < len(output); i++ {
			if output[i].Protected {
				continue
			}
			output = append(output[:i], output[i+1:]...)
			removed = true
			break
		}
		if !removed {
			break
		}
	}
	return output
}

func (c *Compressor) tokenCount(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += c.estimator.count(msg.Role + "\n" + msg.Content)
	}
	return total
}

func cloneMessages(input []Message) []Message {
	output := make([]Message, 0, len(input))
	for _, msg := range input {
		output = append(output, Message{
			Role:      msg.Role,
			Content:   msg.Content,
			Images:    append(make([]string, 0, len(msg.Images)), msg.Images...),
			Protected: msg.Protected,
		})
	}
	return output
}

func reverseMessages(messages []Message) {
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
}

func normalizeContent(content string) string {
	content = strings.ToLower(content)
	content = strings.TrimSpace(content)
	if content == "" {
		return content
	}
	return strings.Join(strings.Fields(content), " ")
}

func tokenSetJaccard(a, b string) float64 {
	if a == "" || b == "" {
		if a == b {
			return 1
		}
		return 0
	}
	setA := splitToTokenSet(a)
	setB := splitToTokenSet(b)

	if len(setA) == 0 || len(setB) == 0 {
		if a == b {
			return 1
		}
		return 0
	}

	intersection := 0
	for token := range setA {
		if _, ok := setB[token]; ok {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func splitToTokenSet(s string) map[string]struct{} {
	tokens := strings.Fields(s)
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

type tokenEstimator struct {
	model      string
	wordRegexp *regexp.Regexp
}

func newTokenEstimator(model string) *tokenEstimator {
	return &tokenEstimator{
		model:      model,
		wordRegexp: regexp.MustCompile(`[\p{L}\p{N}_]+|[^\s]`),
	}
}

func (e *tokenEstimator) count(text string) int {
	if text == "" {
		return 0
	}
	// Pure-Go approximation without external tokenizer dependency.
	// For most mixed natural language/code payloads, this gives a stable budget signal.
	matches := e.wordRegexp.FindAllString(text, -1)
	if len(matches) == 0 {
		return len([]rune(text))/4 + 1
	}
	tokens := len(matches)
	// Longer terms usually split into multiple sub-tokens in BPE-like tokenizers.
	for _, m := range matches {
		if len(m) >= 8 {
			tokens++
		}
		if len(m) >= 16 {
			tokens++
		}
	}
	if strings.Contains(e.model, "qwen") || strings.Contains(e.model, "llama") {
		// Slightly conservative for common local models.
		tokens = int(float64(tokens)*1.1) + 1
	}
	return tokens
}
