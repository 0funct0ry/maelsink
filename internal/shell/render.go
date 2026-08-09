package shell

import "os"

// openRedirection opens the file described by redir for writing, truncating
// unless Append is set (in which case it appends, creating if necessary).
func openRedirection(redir *Redirection) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if redir.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	return os.OpenFile(redir.Path, flags, 0o644)
}

// UpdateStatus records the outcome of a dispatched command on the session:
// LastStatus (0 success, 1 failure) and the reserved Vars["status"] /
// Vars["last_error"]. Builtins are responsible for writing their own
// output (via cliclient's RenderTable/RenderTemplate/RenderDetail etc.) to
// Session.Out during Run — this function only tracks status.
func UpdateStatus(s *Session, err error) {
	if err != nil {
		s.LastStatus = 1
		s.SetVar("status", "error")
		s.SetVar("last_error", err.Error())
		return
	}
	s.LastStatus = 0
	s.SetVar("status", "ok")
	s.SetVar("last_error", "")
}
