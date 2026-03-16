package prompt

import "testing"

func TestParseFunctionCalling_Single(t *testing.T) {
	input := `<tool>{"function":"search","context":{"query":"weather"},"abort_on_error":true}</tool>`
	list, err := ParseFunctionCalling(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 function call, got %d", len(list))
	}
	if list[0].Function != "search" {
		t.Fatalf("unexpected function: %s", list[0].Function)
	}
	if !list[0].AbortOnError {
		t.Fatalf("expected abort_on_error=true")
	}
}

func TestParseFunctionCalling_Multiple(t *testing.T) {
	input := `before<tool>{"function":"a","context":{},"abort_on_error":false}</tool>mid<tool>{"function":"b","context":{"x":1},"abort_on_error":true}</tool>after`
	list, err := ParseFunctionCalling(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 function calls, got %d", len(list))
	}
	if list[0].Function != "a" || list[1].Function != "b" {
		t.Fatalf("unexpected call order: %s, %s", list[0].Function, list[1].Function)
	}
}

func TestParseFunctionCalling_InvalidJSON(t *testing.T) {
	input := `<tool>{"function":"a",bad_json}</tool>`
	if _, err := ParseFunctionCalling(input); err == nil {
		t.Fatalf("expected json parse error")
	}
}

func TestParseLoopEnd(t *testing.T) {
	if !ParseLoopEnd("hello <loop_end/> world") {
		t.Fatalf("expected loop end marker to be detected")
	}
	if ParseLoopEnd("hello world") {
		t.Fatalf("did not expect loop end marker")
	}
}
