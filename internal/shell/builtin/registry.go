package builtin

import "github.com/0funct0ry/maelsink/internal/shell"

// All returns every builtin in this package, in the order cmd/shell.go
// should register them into a shell.Registry via shell.NewRegistry(All()...).
func All() []shell.Builtin {
	return []shell.Builtin{
		List{},
		Show{},
		Delete{},
		Clear{},
		Export{},
		Attachment{},
		Send{},
		Echo{},
		Example{},
		Prompt{},
		Functions{},
		Stats{},
		Health{},
		Version{},
		Config{},
		Set{},
		Unset{},
		Vars{},
		Alias{},
		Unalias{},
		Abbr{},
		Unabbr{},
		Template{},
		History{},
		Edit{},
		Sh{},
		Source{},
		Help{},
		Exit{},
	}
}
