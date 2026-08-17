package builtin

import "github.com/0funct0ry/maelsink/internal/msgspec"

// exampleTemplate/exampleTemplates/findScenario are thin aliases onto
// internal/msgspec's canned-scenario table (shared with compose's job
// kinds, M13.3) — kept as package-local names so example.go/randmsg.go
// don't need touching beyond this file.
type exampleTemplate = msgspec.ExampleTemplate

var exampleTemplates = msgspec.ExampleTemplates

func findScenario(name string) (exampleTemplate, bool) {
	return msgspec.FindScenario(name)
}
