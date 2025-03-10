package prompt

import (
	"encoding/json"
	"regexp"
	"strings"
)

type FunctionCall struct {
	Function   string                 `json:"function"`
	Parameters map[string]interface{} `json:"parameters"`
}

func ParseFunctionCalling(prompt string) ([]*FunctionCall, error) {
	regExp := regexp.MustCompile(`\<tool>(.+)\</tool>`)
	matches := regExp.FindAllString(prompt, -1)
	functionCallList := make([]*FunctionCall, 0, len(matches))
	for _, match := range matches {
		functionCallJson := strings.ReplaceAll(match, "<tool>", "")
		functionCallJson = strings.ReplaceAll(functionCallJson, "</tool>", "")
		functionCall := &FunctionCall{}
		if err := json.Unmarshal([]byte(functionCallJson), functionCall); err != nil {
			return nil, err
		}
		functionCallList = append(functionCallList, functionCall)
	}
	return functionCallList, nil
}
