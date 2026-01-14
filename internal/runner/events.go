package runner

import (
	"openllmjudge/config"
	"openllmjudge/judge"
	"openllmjudge/testcase"
)

// Define message types for the run TUI
// These are defined here to avoid cyclic imports between runner and tui/run

// JudgeStartMsg indicates a judge has started execution
type JudgeStartMsg struct {
	Name   string
	Total  int
	Config config.JudgeConfig
}

// TestStartMsg indicates a test case has started
type TestStartMsg struct {
	JudgeName string
	TestID    string
	TestData  testcase.TestCase
}

// TokenStreamMsg carries a streamed token from the LLM
type TokenStreamMsg struct {
	JudgeName string
	TestID    string
	Token     string
	TokenType judge.TokenType
	Usage     judge.TokenUsage
}

// StreamCompleteMsg indicates streaming has finished for a test
type StreamCompleteMsg struct {
	JudgeName string
	TestID    string
}

// HoldOnMsg indicates a temporary pause
type HoldOnMsg struct {
	JudgeName string
	TestID    string
	Message   string
}

// RoastingStartMsg indicates the start of the roasting phase
type RoastingStartMsg struct {
	JudgeName string
	TestID    string
}

// TestCompleteMsg indicates a test case has finished evaluation
type TestCompleteMsg struct {
	JudgeName string
	TestID    string
	Result    testcase.JudgeResult
	Tokens    int
}

// JudgeCompleteMsg indicates a judge has finished all tests
type JudgeCompleteMsg struct {
	JudgeName string
	// Summary Stats could go here
}

// ErrorMsg carries an error that occurred during execution
type ErrorMsg struct {
	JudgeName string
	TestID    string
	Error     error
}

// Implement Event interface
func (JudgeStartMsg) isEvent()     {}
func (TestStartMsg) isEvent()      {}
func (TokenStreamMsg) isEvent()    {}
func (StreamCompleteMsg) isEvent() {}
func (HoldOnMsg) isEvent()         {}
func (RoastingStartMsg) isEvent()  {}
func (TestCompleteMsg) isEvent()   {}
func (JudgeCompleteMsg) isEvent()  {}
func (ErrorMsg) isEvent()          {}
