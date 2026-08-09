package shell

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/0funct0ry/maelsink/internal/config"
	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

func TestExpandAliases(t *testing.T) {
	tests := []struct {
		name    string
		aliases map[string]string
		line    string
		want    string
	}{
		{
			name:    "no aliases",
			aliases: nil,
			line:    "list foo",
			want:    "list foo",
		},
		{
			name:    "simple first-word substitution",
			aliases: map[string]string{"ls": "list"},
			line:    "ls foo",
			want:    "list foo",
		},
		{
			name:    "no match leaves line unchanged",
			aliases: map[string]string{"ls": "list"},
			line:    "show foo",
			want:    "show foo",
		},
		{
			name:    "chained expansion",
			aliases: map[string]string{"ll": "ls -a", "ls": "list"},
			line:    "ll",
			want:    "list -a",
		},
		{
			name:    "self-referential stops after one substitution",
			aliases: map[string]string{"ls": "ls -a"},
			line:    "ls foo",
			want:    "ls -a foo",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandAliases(tc.aliases, tc.line)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExpandAliasesRecursionCap(t *testing.T) {
	// a -> b, b -> a: without the self-ref short circuit this would need
	// the cap; verify it terminates and returns within maxAliasExpansions.
	aliases := map[string]string{"a": "b", "b": "a"}
	got, err := ExpandAliases(aliases, "a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "a" && got != "b" {
		t.Errorf("got %q, want a or b", got)
	}
}

func TestExpandTemplate(t *testing.T) {
	engine, err := tmpl.New(42, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	t.Run("disabled passthrough", func(t *testing.T) {
		got, err := ExpandTemplate(engine, nil, "hello {{.name}}", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello {{.name}}" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("enabled render", func(t *testing.T) {
		got, err := ExpandTemplate(engine, map[string]string{"name": "world"}, "hello {{.name}}", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("escape sequence preserved literally", func(t *testing.T) {
		got, err := ExpandTemplate(engine, nil, `literal \{{ here`, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "literal {{ here" {
			t.Errorf("got %q", got)
		}
	})
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    []string
		wantErr bool
	}{
		{name: "simple", line: "list foo bar", want: []string{"list", "foo", "bar"}},
		{name: "single quotes literal", line: `echo 'a b \n'`, want: []string{"echo", `a b \n`}},
		{name: "double quotes preserve spaces", line: `echo "a b"`, want: []string{"echo", "a b"}},
		{name: "double quote backslash escape", line: `echo "a \" b"`, want: []string{"echo", `a " b`}},
		{name: "outside quote backslash escapes space", line: `echo a\ b`, want: []string{"echo", "a b"}},
		{name: "unterminated single quote", line: `echo 'a`, wantErr: true},
		{name: "unterminated double quote", line: `echo "a`, wantErr: true},
		{name: "empty line", line: "", want: nil},
		{name: "redirection tokens", line: "list > out.txt", want: []string{"list", ">", "out.txt"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Tokenize(tc.line)
			if tc.wantErr {
				if !errors.Is(err, ErrUnterminatedQuote) {
					t.Fatalf("want ErrUnterminatedQuote, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !equalSlices(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSplitRedirection(t *testing.T) {
	tests := []struct {
		name    string
		tokens  []string
		want    []string
		redir   *Redirection
		wantErr bool
	}{
		{
			name:   "no redirection",
			tokens: []string{"list", "foo"},
			want:   []string{"list", "foo"},
		},
		{
			name:   "truncate redirection",
			tokens: []string{"list", ">", "out.txt"},
			want:   []string{"list"},
			redir:  &Redirection{Path: "out.txt", Append: false},
		},
		{
			name:   "append redirection",
			tokens: []string{"list", ">>", "out.txt"},
			want:   []string{"list"},
			redir:  &Redirection{Path: "out.txt", Append: true},
		},
		{
			name:    "pipeline token is an error",
			tokens:  []string{"list", "|", "grep", "foo"},
			wantErr: true,
		},
		{
			name:    "bare redirection with no path",
			tokens:  []string{"list", ">"},
			wantErr: true,
		},
		{
			name:    "redirection not at the end",
			tokens:  []string{"list", ">", "out.txt", "extra"},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotTokens, gotRedir, err := SplitRedirection(tc.tokens)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !equalSlices(gotTokens, tc.want) {
				t.Errorf("tokens: got %v, want %v", gotTokens, tc.want)
			}
			if (gotRedir == nil) != (tc.redir == nil) {
				t.Fatalf("redir presence mismatch: got %v, want %v", gotRedir, tc.redir)
			}
			if gotRedir != nil && *gotRedir != *tc.redir {
				t.Errorf("redir: got %+v, want %+v", *gotRedir, *tc.redir)
			}
		})
	}
}

// stubBuiltin is a minimal Builtin for Dispatch tests.
type stubBuiltin struct {
	name    string
	aliases []string
	run     func(ctx context.Context, s *Session, args []string) error
}

func (b *stubBuiltin) Name() string      { return b.name }
func (b *stubBuiltin) Aliases() []string { return b.aliases }
func (b *stubBuiltin) Flags() *pflag.FlagSet {
	return pflag.NewFlagSet(b.name, pflag.ContinueOnError)
}
func (b *stubBuiltin) Run(ctx context.Context, s *Session, args []string) error {
	return b.run(ctx, s, args)
}

func newTestSession() *Session {
	return &Session{
		Vars: map[string]string{},
		Out:  new(bytes.Buffer),
		Err:  new(bytes.Buffer),
	}
}

func TestDispatch(t *testing.T) {
	ok := &stubBuiltin{name: "ok", run: func(ctx context.Context, s *Session, args []string) error {
		return nil
	}}
	fail := &stubBuiltin{name: "fail", aliases: []string{"f"}, run: func(ctx context.Context, s *Session, args []string) error {
		return errors.New("boom")
	}}
	echo := &stubBuiltin{name: "echo", run: func(ctx context.Context, s *Session, args []string) error {
		s.Out.Write([]byte(strings.Join(args, " ")))
		return nil
	}}
	reg := NewRegistry(ok, fail, echo)

	t.Run("success", func(t *testing.T) {
		s := newTestSession()
		if err := Dispatch(context.Background(), s, reg, []string{"ok"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		s := newTestSession()
		if err := Dispatch(context.Background(), s, reg, []string{"fail"}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("alias resolution", func(t *testing.T) {
		s := newTestSession()
		if err := Dispatch(context.Background(), s, reg, []string{"f"}); err == nil {
			t.Fatal("expected error via alias")
		}
	})

	t.Run("echoes args", func(t *testing.T) {
		s := newTestSession()
		if err := Dispatch(context.Background(), s, reg, []string{"echo", "hi", "there"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := s.Out.(*bytes.Buffer).String(); got != "hi there" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("unknown command", func(t *testing.T) {
		s := newTestSession()
		err := Dispatch(context.Background(), s, reg, []string{"nope"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("help flag is swallowed, not treated as failure", func(t *testing.T) {
		helpErr := &stubBuiltin{name: "helpme", run: func(ctx context.Context, s *Session, args []string) error {
			fs := pflag.NewFlagSet("helpme", pflag.ContinueOnError)
			fs.SetOutput(s.Out)
			return fs.Parse(args) // args contains "-h", which pflag.Parse itself
			// prints usage for and returns pflag.ErrHelp for, exactly like
			// every real builtin's own flag parsing does.
		}}
		reg := NewRegistry(helpErr)
		s := newTestSession()
		if err := Dispatch(context.Background(), s, reg, []string{"helpme", "-h"}); err != nil {
			t.Fatalf("expected -h to be swallowed as success, got: %v", err)
		}
	})

	t.Run("command prefix honored", func(t *testing.T) {
		s := newTestSession()
		s.CommandPrefix = ":"
		// unprefixed no longer resolves
		if err := Dispatch(context.Background(), s, reg, []string{"ok"}); err == nil {
			t.Fatal("expected unknown command error for unprefixed token with prefix set")
		}
		if err := Dispatch(context.Background(), s, reg, []string{":ok"}); err != nil {
			t.Fatalf("unexpected error for prefixed token: %v", err)
		}
	})
}

func TestEvalIntegration(t *testing.T) {
	echo := &stubBuiltin{name: "echo", run: func(ctx context.Context, s *Session, args []string) error {
		s.Out.Write([]byte(strings.Join(args, ",")))
		return nil
	}}
	reg := NewRegistry(echo)

	engine, err := tmpl.New(1, false)
	if err != nil {
		t.Fatalf("tmpl.New: %v", err)
	}
	defer engine.Close()

	s := NewSession(config.Shell{TemplateEnabled: false}, nil, "", nil, engine, new(bytes.Buffer), new(bytes.Buffer), nil)

	if err := Eval(context.Background(), s, reg, "echo a b"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := s.Out.(*bytes.Buffer).String(); got != "a,b" {
		t.Errorf("got %q", got)
	}
	if s.LastStatus != 0 {
		t.Errorf("LastStatus = %d, want 0", s.LastStatus)
	}
	if v, _ := s.GetVar("status"); v != "ok" {
		t.Errorf("status var = %q, want ok", v)
	}

	// comment-only line: no-op, must not error and must not touch previous status incorrectly
	if err := Eval(context.Background(), s, reg, "  # a comment"); err != nil {
		t.Fatalf("comment line should not error: %v", err)
	}
}
