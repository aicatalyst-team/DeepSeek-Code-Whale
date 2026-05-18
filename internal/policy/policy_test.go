package policy

import (
	"testing"

	"github.com/usewhale/whale/internal/core"
)

func TestDefaultToolPolicyPrefixRulesApplyToShellRunCommand(t *testing.T) {
	p := DefaultToolPolicy{
		Mode:          ApprovalModeOnRequest,
		AllowPrefixes: []string{"git status"},
		DenyPrefixes:  []string{"rm -rf"},
	}
	spec := core.ToolSpec{Name: "shell_run"}

	allow := p.Decide(spec, core.ToolCall{Name: "shell_run", Input: `{"command":"git status --short"}`})
	if !allow.Allow || allow.RequiresApproval || allow.Code != "allow_prefix" || allow.MatchedRule != "git status" {
		t.Fatalf("expected allow-prefix decision for shell_run.command: %+v", allow)
	}

	deny := p.Decide(spec, core.ToolCall{Name: "shell_run", Input: `{"command":"rm -rf /tmp/x"}`})
	if deny.Allow || deny.Code != "policy_denied" || deny.MatchedRule != "rm -rf" {
		t.Fatalf("expected deny-prefix decision for shell_run.command: %+v", deny)
	}
}

func TestDefaultToolPolicyNeverSkipsApprovalForMutatingTools(t *testing.T) {
	p := DefaultToolPolicy{Mode: ApprovalModeNever}
	tests := []struct {
		name string
		spec core.ToolSpec
		call core.ToolCall
	}{
		{
			name: "write",
			spec: core.ToolSpec{Name: "write"},
			call: core.ToolCall{Name: "write", Input: `{"file_path":"a.txt","content":"x"}`},
		},
		{
			name: "apply_patch",
			spec: core.ToolSpec{Name: "apply_patch"},
			call: core.ToolCall{Name: "apply_patch", Input: `{"patch":"*** Begin Patch\n*** End Patch\n"}`},
		},
		{
			name: "shell_run",
			spec: core.ToolSpec{Name: "shell_run"},
			call: core.ToolCall{Name: "shell_run", Input: `{"command":"go test ./..."}`},
		},
		{
			name: "mcp",
			spec: core.ToolSpec{Name: "mcp__github__create_issue"},
			call: core.ToolCall{Name: "mcp__github__create_issue", Input: `{}`},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := p.Decide(tc.spec, tc.call)
			if !decision.Allow || decision.RequiresApproval || decision.Code != "auto_allow" {
				t.Fatalf("decision: %+v", decision)
			}
		})
	}
}

func TestDefaultToolPolicyNeverStillHonorsDenyPrefixes(t *testing.T) {
	p := DefaultToolPolicy{
		Mode:         ApprovalModeNever,
		DenyPrefixes: []string{"rm -rf"},
	}
	decision := p.Decide(
		core.ToolSpec{Name: "shell_run"},
		core.ToolCall{Name: "shell_run", Input: `{"command":"rm -rf /tmp/x"}`},
	)
	if decision.Allow || decision.Code != "policy_denied" || decision.MatchedRule != "rm -rf" {
		t.Fatalf("decision: %+v", decision)
	}
}

func TestShellCommandFromInput(t *testing.T) {
	if got := shellCommandFromInput(`{"command":" echo hi "}`); got != "echo hi" {
		t.Fatalf("shellCommandFromInput = %q, want %q", got, "echo hi")
	}
	if got := shellCommandFromInput(`{`); got != "" {
		t.Fatalf("shellCommandFromInput malformed = %q, want empty", got)
	}
}
