package logx

// Logger defines an interface for a single logger method.
type Logger interface {
	Printf(s string, args ...interface{})
}

// LoggerFunc is an adapter to use ordinary functions as Logger.
type LoggerFunc func(string, ...interface{})

// Printf calls the wrapped func.
func (f LoggerFunc) Printf(s string, args ...interface{}) { f(s, args...) }

// NopLogger logs literally nothing.
func NopLogger() Logger {
	return LoggerFunc(func(string, ...interface{}) {})
}
