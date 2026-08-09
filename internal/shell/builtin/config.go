package builtin

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"

	"github.com/0funct0ry/maelsink/internal/shell"
)

// Config implements the "config list|get <key>|set <key> <value>|save
// [path]" builtin (SPEC.md §7.5.4): operates on the session's shell.*
// settings only (never server config, per §7.5.14's non-goal).
//
// "save [path]" semantics: writes the session's CURRENT shell.* settings
// into the target maelsink.yaml file's "shell:" key, preserving every other
// top-level section untouched — this is a targeted read-modify-write over a
// generic YAML map, not a full config round-trip (Session only holds
// config.Shell, not the full config.Config, so a full round-trip isn't
// available without widening Session's surface). --force additionally
// creates the file (with only a "shell:" section) if it does not exist yet;
// without --force, save refuses to create a brand-new file and only
// updates an existing one.
type Config struct{}

func (Config) Name() string      { return "config" }
func (Config) Aliases() []string { return nil }
func (Config) Short() string     { return "Read/modify the session's shell.* settings" }

// shellFields is the hand-written key table for get/set, mirroring
// config.Shell's fields (SPEC.md §3.1's shell.* keys). A hand-written
// switch is used instead of reflection for clarity over this small, fixed
// field set, per the plan's guidance.
var shellFields = []string{
	"command_prefix", "prompt", "history_file", "history_size", "color",
	"seed", "editor", "sh_enabled", "exit_on_error", "abbr_trigger_key",
	"template_enabled", "template_unsafe_funcs",
}

func (Config) Flags() *pflag.FlagSet {
	fs := pflag.NewFlagSet("config", pflag.ContinueOnError)
	fs.String("format", "table", "output format for `list`: table|json|yaml")
	fs.Bool("force", false, "on `save`, create the file if it does not already exist")
	return fs
}

func (b Config) Run(ctx context.Context, s *shell.Session, args []string) error {
	fs := b.Flags()
	if err := fs.Parse(args); err != nil {
		return err
	}
	pos := fs.Args()
	if len(pos) == 0 {
		return fmt.Errorf("config: requires a subcommand: list|get|set|save")
	}

	switch pos[0] {
	case "list":
		format, _ := fs.GetString("format")
		return writeFormatted(s.Out, format, s.Cfg, func(w io.Writer) error {
			for _, k := range shellFields {
				v, _ := getShellField(s, k)
				fmt.Fprintf(w, "%s\t%s\n", k, v)
			}
			return nil
		})
	case "get":
		if len(pos) < 2 {
			return fmt.Errorf("config get: requires <key>")
		}
		v, err := getShellField(s, pos[1])
		if err != nil {
			return err
		}
		fmt.Fprintln(s.Out, v)
		return nil
	case "set":
		if len(pos) < 3 {
			return fmt.Errorf("config set: requires <key> <value>")
		}
		return setShellField(s, pos[1], pos[2])
	case "save":
		path := "maelsink.yaml"
		if len(pos) >= 2 {
			path = pos[1]
		}
		force, _ := fs.GetBool("force")
		return saveShellConfig(s, path, force)
	default:
		return fmt.Errorf("config: unknown subcommand %q (want list|get|set|save)", pos[0])
	}
}

func getShellField(s *shell.Session, key string) (string, error) {
	c := s.Cfg
	switch key {
	case "command_prefix":
		return c.CommandPrefix, nil
	case "prompt":
		return c.Prompt, nil
	case "history_file":
		return c.HistoryFile, nil
	case "history_size":
		return strconv.Itoa(c.HistorySize), nil
	case "color":
		return c.Color, nil
	case "seed":
		return strconv.FormatInt(c.Seed, 10), nil
	case "editor":
		return c.Editor, nil
	case "sh_enabled":
		return strconv.FormatBool(c.ShEnabled), nil
	case "exit_on_error":
		return strconv.FormatBool(c.ExitOnError), nil
	case "abbr_trigger_key":
		return c.AbbrTriggerKey, nil
	case "template_enabled":
		return strconv.FormatBool(c.TemplateEnabled), nil
	case "template_unsafe_funcs":
		return strconv.FormatBool(c.TemplateUnsafeFuncs), nil
	default:
		return "", fmt.Errorf("config: unknown key %q", key)
	}
}

func setShellField(s *shell.Session, key, value string) error {
	c := &s.Cfg
	switch key {
	case "command_prefix":
		c.CommandPrefix = value
	case "prompt":
		c.Prompt = value
	case "history_file":
		c.HistoryFile = value
	case "history_size":
		n, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.HistorySize = n
	case "color":
		c.Color = value
	case "seed":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		c.Seed = n
	case "editor":
		c.Editor = value
	case "sh_enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		c.ShEnabled = v
	case "exit_on_error":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		c.ExitOnError = v
	case "abbr_trigger_key":
		c.AbbrTriggerKey = value
	case "template_enabled":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		c.TemplateEnabled = v
	case "template_unsafe_funcs":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		c.TemplateUnsafeFuncs = v
	default:
		return fmt.Errorf("config: unknown key %q", key)
	}
	return nil
}

// saveShellConfig writes s.Cfg into path's "shell:" top-level key, leaving
// every other section as-is. It reads the existing file into a generic
// map[string]any (if the file exists), overwrites only the "shell" entry,
// and writes the whole map back. If the file does not exist and force is
// false, it refuses; if force is true, it creates a new file containing
// only the shell section.
func saveShellConfig(s *shell.Session, path string, force bool) error {
	doc := map[string]any{}
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(existing, &doc); err != nil {
			return fmt.Errorf("config save: parsing existing %s: %w", path, err)
		}
	case os.IsNotExist(err):
		if !force {
			return fmt.Errorf("config save: %s does not exist (use --force to create it)", path)
		}
	default:
		return err
	}

	shellDoc := map[string]any{}
	for _, k := range shellFields {
		v, _ := getShellField(s, k)
		shellDoc[k] = v
	}
	doc["shell"] = shellDoc

	out, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(s.Out, "wrote %s\n", path)
	return nil
}
