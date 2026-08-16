package compose

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/shell/tmpl"
)

// funcDocResponse mirrors tmpl.FuncDoc without its Fn field, which is not
// JSON-serializable and not needed client-side (SPEC.md §7.7.4.1's
// function-picker only needs name/args/description to build snippets).
type funcDocResponse struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Args        string `json:"args"`
	Returns     string `json:"returns"`
	Description string `json:"description"`
}

// functionsHandler is a thin proxy over tmpl.Engine.Registry(), letting the
// Composer's function-picker search the same function list "template
// --funcs"/the shell's "functions" builtin document (SPEC.md §7.5.7.1).
func functionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		engine, err := tmpl.New(0, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "engine_init_failed", "message": err.Error()}})
			return
		}
		defer engine.Close()

		registry := engine.Registry()
		out := make([]funcDocResponse, 0, len(registry))
		for _, d := range registry {
			out = append(out, funcDocResponse{
				Name:        d.Name,
				Category:    string(d.Category),
				Args:        d.Args,
				Returns:     d.Returns,
				Description: d.Description,
			})
		}
		c.JSON(http.StatusOK, gin.H{"functions": out})
	}
}
