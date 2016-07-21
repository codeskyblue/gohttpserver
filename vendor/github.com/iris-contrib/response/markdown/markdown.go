package markdown

import (
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

const (
	// ContentType the key for the engine, the user/dev can still use its own
	ContentType = "text/markdown"
)

// Engine the response engine which renders a markdown contents as html
type Engine struct {
	config Config
}

// New returns a new markdown response engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig().Merge(cfg)
	return &Engine{config: c}
}

// Response accepts the 'object' value and converts it to bytes in order to be 'renderable'
// implements the iris.ResponseEngine
func (e *Engine) Response(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	var b []byte
	if s, isString := val.(string); isString {
		b = []byte(s)
	} else {
		b = val.([]byte)
	}
	buf := blackfriday.MarkdownCommon(b)
	if e.config.MarkdownSanitize {
		buf = bluemonday.UGCPolicy().SanitizeBytes(buf)
	}

	return buf, nil
}
