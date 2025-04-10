package impl

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ai_agent "github.com/luoxiaojun1992/ai-agent"
)

type Team struct {
	Members map[string]*ai_agent.AgentDouble
}

func (t *Team) GetDescription() string {
	description := `
The "team" skill represents a collaborative group of AI agents, each with specialized roles and capabilities. This enables complex problem-solving through coordinated interactions among team members.
1. Team Members
%s
Each member is designed to contribute unique expertise to the team's collective intelligence.
2. Invocation Context
To activate the "team" skill, use the following JSON format:
{
  "member": "[specific member name]",
  "message": "[message or task description for the member]"
}
member: Specifies which team member should handle the task.
message: Contains the detailed request, question, or instruction for the designated member.
This skill allows precise coordination among team members for efficient task execution.
This description provides clear guidance on the team's composition and how to interact with its members through the specified JSON format.
`
	memberDescriptionList := make([]string, 0, len(t.Members))
	for _, member := range t.Members {
		memberDescriptionList = append(memberDescriptionList, member.Agent.GetDescription())
	}
	allMemberDescription := strings.Join(memberDescriptionList, "\n\n")
	return fmt.Sprintf(description, allMemberDescription)
}

func (t *Team) Do(ctx context.Context, cmdCtx any, callback func(output any) (any, error)) error {
	params, isValidParams := cmdCtx.(map[string]any)
	if !isValidParams {
		return errors.New("error converting params for team skill")
	}

	memberName, hasMemberName := params["member"]
	if !hasMemberName {
		return errors.New("not found member from params")
	}
	memberNameStr, isValidMemberName := memberName.(string)
	if !isValidMemberName {
		return errors.New("error converting member from params")
	}
	member, hasMember := t.Members[memberNameStr]
	if !hasMember {
		return fmt.Errorf("not found member [%s]", memberNameStr)
	}

	message, hasMessage := params["message"]
	if !hasMessage {
		return errors.New("not found message from params")
	}
	messageStr, isValidMessage := message.(string)
	if !isValidMessage {
		return errors.New("error converting message from params")
	}

	return member.ListenAndWatch(ctx, messageStr, nil, func(response string) error {
		_, err := callback(response)
		return err
	})
}
