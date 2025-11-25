package prompt

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"
)

type FunctionCall struct {
	Function     string `json:"function"`
	Context      any    `json:"context"`
	AbortOnError bool   `json:"abort_on_error"`
}

func ParseFunctionCalling(prompt string) ([]*FunctionCall, error) {
	//todo test
	//todo remove debug log
	log.Println(prompt)
	regExp := regexp.MustCompile(`\<tool>(.+)\</tool>`)
	matches := regExp.FindAllString(prompt, -1)
	//todo remove debug log
	log.Println(matches)
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
	//todo remove debug log
	log.Println(funcCallList)
	return funcCallList, nil
}

func ParseLoopEnd(prompt string) bool {
	regExp := regexp.MustCompile(`\<loop_end/>`)
	return regExp.MatchString(prompt)
}
