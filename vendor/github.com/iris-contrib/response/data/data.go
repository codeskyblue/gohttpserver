package data

const (
	// ContentType the key for the engine, the user/dev can still use its own
	ContentType = "application/octet-stream"
)

// Engine the response engine which renders a binary, data
type Engine struct {
	config Config
}

// New returns a new data response engine
func New(cfg ...Config) *Engine {
	c := Config{} // I know it's just empty for now
	if len(cfg) > 0 {
		c = cfg[0]
	}
	return &Engine{config: c}
}

// Response accepts the 'object' value and converts it to bytes in order to be 'renderable'
// implements the iris.ResponseEngine
func (e *Engine) Response(val interface{}, options ...map[string]interface{}) ([]byte, error) {
	return val.([]byte), nil
}
