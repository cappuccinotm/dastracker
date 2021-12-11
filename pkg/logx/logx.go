package logx

// Logger defines an interface for a single logger method.
type Logger interface {
	Printf(s string, args ...interface{})
}
