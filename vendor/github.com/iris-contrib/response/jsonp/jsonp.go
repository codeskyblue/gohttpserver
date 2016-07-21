package jsonp

import (
	"encoding/json"

	"github.com/kataras/iris/utils"
)

const (
	// ContentType the key for the engine, the user/dev can still use its own
	ContentType = "application/javascript"
)

// Engine the response engine which renders a JSONP 'object' with its callback
type Engine struct {
	config Config
	buffer *utils.BufferPool
}

// New returns a new jsonp response engine
func New(cfg ...Config) *Engine {
	c := DefaultConfig().Merge(cfg)
	return &Engine{config: c, buffer: utils.NewBufferPool(8)}
}

func (e *Engine) getCallbackOption(options map[string]interface{}) string {
	callbackOpt := options["callback"]
	if s, isString := callbackOpt.(string); isString {
		return s
	}
	return e.config.Callback
}

// Response accepts the 'object' value and converts it to bytes in order to be 'renderable'
// implements the iris.ResponseEngine
func (e *Engine) Response(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	var result []byte
	var err error
	w := e.buffer.Get()
	defer e.buffer.Put(w)
	if e.config.Indent {
		result, err = json.MarshalIndent(val, "", "  ")
	} else {
		result, err = json.Marshal(val)
	}

	if err != nil {
		return nil, err
	}

	// the config's callback can be overriden with the options
	callback := e.config.Callback
	if len(options) > 0 {
		callback = e.getCallbackOption(options[0])
	}

	if callback != "" {
		w.Write([]byte(callback + "("))
		w.Write(result)
		w.Write([]byte(");"))
	} else {
		w.Write(result)
	}

	if e.config.Indent {
		w.Write([]byte("\n"))
	}
	return w.Bytes(), nil
}
