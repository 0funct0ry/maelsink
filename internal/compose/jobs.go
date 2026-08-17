package compose

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0funct0ry/maelsink/internal/compose/job"
)

// targetOf projects a compose TargetConfig down to the fields job.Target
// needs — job must not import internal/compose (which imports job).
func targetOf(cfg TargetConfig) job.Target {
	return job.Target{SMTPAddr: cfg.SMTPAddr, SMTPUser: cfg.SMTPUser, SMTPPass: cfg.SMTPPass}
}

// bindJobParams decodes c's JSON body into dst. Every job kind's params
// struct is entirely optional (every field has a CLI-flag-equivalent
// default), so a missing/empty body is not an error — only malformed JSON
// is.
func bindJobParams(c *gin.Context, dst any) error {
	if c.Request.ContentLength == 0 {
		return nil
	}
	dec := json.NewDecoder(c.Request.Body)
	if err := dec.Decode(dst); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func bindError(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "invalid_request", "message": err.Error()}})
}

// startJobHandler implements POST /compose-api/v1/jobs/:kind (SPEC.md
// §7.7.4.3): binds the JSON body to :kind's params struct and starts the
// job via mgr, returning its id immediately — intmsg/blast/deluge continue
// running after the response; randmsg/weirdmsg complete synchronously or
// near-instantly, but the handler still returns as soon as the job is
// registered, not once it finishes, so every kind is started the same way.
func startJobHandler(mgr *job.Manager, target TargetConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		kind := c.Param("kind")
		tgt := targetOf(target)

		var run job.RunFunc
		switch kind {
		case "randmsg":
			var p job.RandMsgParams
			if err := bindJobParams(c, &p); err != nil {
				bindError(c, err)
				return
			}
			run = job.RunRandMsg(tgt, p)
		case "intmsg":
			var p job.IntMsgParams
			if err := bindJobParams(c, &p); err != nil {
				bindError(c, err)
				return
			}
			run = job.RunIntMsg(tgt, p)
		case "weirdmsg":
			var p job.WeirdMsgParams
			if err := bindJobParams(c, &p); err != nil {
				bindError(c, err)
				return
			}
			run = job.RunWeirdMsg(tgt, p)
		case "blast":
			var p job.BlastParams
			if err := bindJobParams(c, &p); err != nil {
				bindError(c, err)
				return
			}
			run = job.RunBlast(tgt, p)
		case "deluge":
			var p job.DelugeParams
			if err := bindJobParams(c, &p); err != nil {
				bindError(c, err)
				return
			}
			run = job.RunDeluge(tgt, p)
		default:
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "unknown_kind", "message": "kind must be one of randmsg, intmsg, weirdmsg, blast, deluge"}})
			return
		}

		j := mgr.Start(kind, run)
		c.JSON(http.StatusOK, gin.H{"jobId": j.ID})
	}
}

// cancelJobHandler implements POST /compose-api/v1/jobs/:jobId/cancel.
// Registered as ":kind" in compose.go, not ":jobId" — gin's router requires
// every route sharing the "/jobs/:x" prefix to use the same wildcard name,
// and startJobHandler's "/jobs/:kind" already claims that slot.
func cancelJobHandler(mgr *job.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("kind")
		j, ok := mgr.Get(id)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "not_found", "message": "no job with that id"}})
			return
		}
		mgr.Cancel(id)
		c.JSON(http.StatusOK, j.Snapshot())
	}
}

// listJobsHandler implements GET /compose-api/v1/jobs — every registered
// job (running or finished), for the Jobs Panel's recent-jobs list to
// refresh from on load/reconnect (SPEC.md §7.7.4.3: no persistence, so this
// is empty immediately after a compose restart).
func listJobsHandler(mgr *job.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		jobs := mgr.List()
		snapshots := make([]job.Snapshot, len(jobs))
		for i, j := range jobs {
			snapshots[i] = j.Snapshot()
		}
		c.JSON(http.StatusOK, gin.H{"jobs": snapshots})
	}
}
