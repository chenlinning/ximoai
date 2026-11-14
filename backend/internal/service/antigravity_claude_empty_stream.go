package service

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const antigravityEmptyResponseCode = "ANTIGRAVITY_EMPTY_RESPONSE"
const antigravityEmptyResponseMessage = "Antigravity upstream completed without content"

// antigravityClaudeStreamGuard observes the converted SSE blocks, rather than
// token counts or individual Gemini chunks. A final usage-only chunk is valid
// when an earlier chunk has already supplied content.
type antigravityClaudeStreamGuard struct {
	sawBlock     bool
	sawSignal    bool
	finishReason string
	pending      []byte
}

func newAntigravityClaudeStreamGuard(model string) *antigravityClaudeStreamGuard {
	model = strings.ToLower(strings.TrimSpace(model))
	if model != "gemini-4-hydrogen" && !strings.HasPrefix(model, "gemini-4-hydrogen-") {
		return nil
	}
	return &antigravityClaudeStreamGuard{}
}

func (g *antigravityClaudeStreamGuard) observe(line string, events []byte) []byte {
	line = strings.TrimSpace(line)
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if strings.HasPrefix(line, "data:") && gjson.Valid(payload) {
		root := gjson.Parse(payload)
		if inner := root.Get("response"); inner.IsObject() {
			root = inner
		}
		if root.Get("error").Exists() || root.Get("promptFeedback.blockReason").String() != "" {
			g.sawSignal = true
		}
		if reason := strings.TrimSpace(root.Get("candidates.0.finishReason").String()); reason != "" {
			g.finishReason = reason
		}
	}
	g.sawBlock = g.sawBlock || bytes.Contains(events, []byte("event: content_block_start\n"))
	// The converter normally emits message_stop in the same chunk as STOP.
	// Hold an otherwise empty terminal chunk until EOF, so late usage is kept
	// and an error can replace the successful terminator.
	if !g.sawBlock && !g.sawSignal && g.finishReason == "STOP" && bytes.Contains(events, []byte("event: message_stop\n")) {
		g.pending = append(g.pending, events...)
		return nil
	}
	if len(g.pending) != 0 && (g.sawBlock || g.sawSignal || (g.finishReason != "" && g.finishReason != "STOP")) {
		events = append(g.pending, events...)
		g.pending = nil
	}
	return events
}

func (g *antigravityClaudeStreamGuard) finish(events []byte, started bool) ([]byte, bool) {
	g.sawBlock = g.sawBlock || bytes.Contains(events, []byte("event: content_block_start\n"))
	empty := started && !g.sawBlock && !g.sawSignal && (g.finishReason == "" || g.finishReason == "STOP")
	if empty {
		return nil, true
	}
	return append(g.pending, events...), false
}

// Report the semantic failure without dropping the collected upstream usage.
// Once any SSE frame has been flushed, the wire status cannot be changed.
func writeAntigravityEmptyClaudeResponse(c *gin.Context, cw *antigravityClientWriter, finishReason string) {
	if finishReason == "" {
		finishReason = "missing"
	}
	message := antigravityEmptyResponseMessage + " (finishReason=" + finishReason + ")"
	MarkOpsStreamFailure(c, "upstream_error", antigravityEmptyResponseCode, message, http.StatusBadGateway)
	if !c.Writer.Written() {
		c.Writer.Header().Del("Content-Type")
		c.JSON(http.StatusBadGateway, gin.H{"type": "error", "error": gin.H{"type": "api_error", "code": antigravityEmptyResponseCode, "message": message}})
		return
	}
	cw.Write([]byte("event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"api_error\",\"code\":\"" + antigravityEmptyResponseCode + "\",\"message\":\"" + message + "\"}}\n\n"))
}
