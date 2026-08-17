package compose

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/0funct0ry/maelsink/internal/compose/job"
)

// jobTickInterval is how often jobStreamHandler polls the job's snapshot
// and pushes a progress tick — compose's own job table is in-process, so a
// short poll is simpler than wiring a per-job pub/sub and still feels live.
const jobTickInterval = 250 * time.Millisecond

var jobStreamUpgrader = websocket.Upgrader{
	// Compose's SPA and its WS endpoints are always same-origin (no
	// cross-origin WebSocket clients are expected), mirroring internal/ws's
	// upgrader.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// jobStreamHandler implements GET /compose-api/v1/jobs/:jobId/stream
// (SPEC.md §7.7.4.3): upgrades to a WebSocket and pushes a JSON progress
// tick (job.Snapshot) every jobTickInterval until the job reaches a
// terminal state, sends one final tick, then closes. Unlike
// internal/ws.Hub, this is a single-connection push loop, not a
// multi-client fan-out hub — compose's job progress has exactly one
// natural subscriber per job stream. Registered as ":kind" in compose.go
// (see cancelJobHandler's comment) — gin requires every route sharing the
// "/jobs/:x" prefix to use the same wildcard name.
func jobStreamHandler(mgr *job.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		j, ok := mgr.Get(c.Param("kind"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "no job with that id"}})
			return
		}

		conn, err := jobStreamUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		ticker := time.NewTicker(jobTickInterval)
		defer ticker.Stop()

		writeTick := func() bool {
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			return conn.WriteJSON(j.Snapshot()) == nil
		}

		if !writeTick() {
			return
		}
		for !j.Done() {
			select {
			case <-ticker.C:
				if !writeTick() {
					return
				}
			case <-c.Request.Context().Done():
				return
			}
		}
		// Final tick, guaranteed to reflect the terminal state even if the
		// job finished between two poll ticks.
		writeTick()
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}
}
