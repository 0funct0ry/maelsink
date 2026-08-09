package lineedit

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

type fakeSource struct {
	builtins  []string
	flags     map[string][]string
	vars      []string
	msgIDs    []string
	attIDs    map[string][]string
	funcNames []string
}

func (f *fakeSource) BuiltinNames() []string { return f.builtins }
func (f *fakeSource) FlagsFor(name string) []string {
	return f.flags[name]
}
func (f *fakeSource) VarNames() []string { return f.vars }
func (f *fakeSource) RecentMessageIDs(ctx context.Context) []string {
	return f.msgIDs
}
func (f *fakeSource) AttachmentIDs(ctx context.Context, msgID string) []string {
	return f.attIDs[msgID]
}
func (f *fakeSource) TemplateFuncNames() []string { return f.funcNames }

func newFakeSource() *fakeSource {
	return &fakeSource{
		builtins: []string{"list", "show", "delete", "send"},
		flags: map[string][]string{
			"list": {"--limit", "-n", "--subject"},
			"send": {"--to", "--dir", "--template"},
		},
		vars:      []string{"connected", "lastID", "count"},
		msgIDs:    []string{"01ABC", "01DEF"},
		attIDs:    map[string][]string{"01ABC": {"0", "1"}},
		funcNames: []string{"upper", "lower", "uuid"},
	}
}

func assertCandidates(t *testing.T, got []string, wantWordStart, gotWordStart int, want []string) {
	t.Helper()
	sort.Strings(got)
	sort.Strings(want)
	if len(got) == 0 && len(want) == 0 {
		// nil vs []string{} both mean "no candidates".
	} else if !reflect.DeepEqual(got, want) {
		t.Errorf("candidates = %v, want %v", got, want)
	}
	if gotWordStart != wantWordStart {
		t.Errorf("wordStart = %d, want %d", gotWordStart, wantWordStart)
	}
}

func TestCompleteFirstWord(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("li")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 0, start, []string{"list"})
}

func TestCompleteAfterBuiltinFlags(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("list --l")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 5, start, []string{"--limit"})
}

func TestCompleteVar(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("set foo $conn")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 8, start, []string{"$connected"})
}

func TestCompleteInsideTemplateFunc(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("send --body {{ up")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 15, start, []string{"upper"})
}

func TestCompleteInsideTemplateVarAfterDot(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("send --body {{ .co")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 15, start, []string{".connected", ".count"})
}

func TestCompleteNotInsideTemplateAfterClosed(t *testing.T) {
	c := NewCompleter(newFakeSource())
	// {{ }} already closed, so this is no longer template-function
	// completion. Per the plan's dispatch order, flags are only offered
	// when the token immediately preceding the current word is a
	// recognized builtin name — here it's "}}", so nothing is offered.
	line := []rune("send {{ .x }} --t")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 14, start, []string{})
}

func TestCompletePathFlag(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("send --dir ")
	cands, _ := c.Complete(line, len(line))
	// We can't assert exact filesystem contents portably, but it must not
	// fall through to builtin-name completion.
	for _, cand := range cands {
		if cand == "list" || cand == "show" {
			t.Errorf("expected filesystem candidates, got builtin name %q", cand)
		}
	}
}

func TestCompleteMessageIDPosition(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("show 01")
	cands, start := c.Complete(line, len(line))
	assertCandidates(t, cands, 5, start, []string{"01ABC", "01DEF"})
}

func TestCompleteAttachmentIDPosition(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("attachment 01ABC ")
	cands, _ := c.Complete(line, len(line))
	assertCandidates(t, cands, 18, 18, []string{"0", "1"})
}

func TestCompleteUnknownContextReturnsNothing(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("list --subject foo ba")
	cands, _ := c.Complete(line, len(line))
	if len(cands) != 0 {
		t.Errorf("expected no candidates in an unrecognized positional context, got %v", cands)
	}
}

func TestDoConvertsToSuffixes(t *testing.T) {
	c := NewCompleter(newFakeSource())
	line := []rune("li")
	newLines, offset := c.Do(line, len(line))
	if offset != 2 {
		t.Fatalf("offset = %d, want 2", offset)
	}
	if len(newLines) != 1 || string(newLines[0]) != "st" {
		t.Fatalf("newLines = %v, want [st]", newLines)
	}
}
