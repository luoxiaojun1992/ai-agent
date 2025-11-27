package prompt

import (
	"encoding/json"
	"regexp"
	"strings"
)

type FunctionCall struct {
	Function     string `json:"function"`
	Context      any    `json:"context"`
	AbortOnError bool   `json:"abort_on_error"`
}

func ParseFunctionCalling(prompt string) ([]*FunctionCall, error) {
	regExp := regexp.MustCompile(`(?s)<tool>(.+?)</tool>`)
	matches := regExp.FindAllString(prompt, -1)
	funcCallList := make([]*FunctionCall, 0, len(matches))
	for _, match := range matches {
		funcCallJson := strings.ReplaceAll(match, "<tool>", "")
		funcCallJson = strings.ReplaceAll(funcCallJson, "</tool>", "")
		functionCall := &FunctionCall{}
		if err := json.Unmarshal([]byte(funcCallJson), functionCall); err != nil {
			return nil, err
		}
		funcCallList = append(funcCallList, functionCall)
	}
	return funcCallList, nil
}

func ParseLoopEnd(prompt string) bool {
	regExp := regexp.MustCompile(`(?s)<loop_end/>`)
	return regExp.MatchString(prompt)
}
