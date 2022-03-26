package logx

import (
	"log"
	"strings"
)

//go:generate rm -f logx_mock.go
//go:generate moq -out logx_mock.go -fmt goimports . Logger

// Logger defines an interface for a single logger method.
type Logger interface {
	Printf(s string, args ...interface{})
	Sub(p string) Logger
}

type nop struct{}

// Printf is a no-op.
func (nop) Printf(s string, args ...interface{}) {}

// Sub is a no-op.
func (nop) Sub(p string) Logger { return nop{} }

// Nop is a no-op logger.
func Nop() Logger { return nop{} }

type std struct {
	prefix string
	l      *log.Logger
	levels []string
}

// Printf logs a message with the given format and args.
func (l *std) Printf(s string, args ...interface{}) {
	lvl, line := l.extractLevel(s)
	l.l.Printf("["+lvl+"] "+l.prefix+line, args...)
}

// Sub returns a new logger with the given prefix.
func (l *std) Sub(p string) Logger { return &std{prefix: l.prefix + p, l: l.l, levels: l.levels} }

// Std is a logger that writes to the standard log package.
func Std(log *log.Logger, levels []string) Logger { return &std{l: log, levels: levels} }

func (l *std) extractLevel(line string) (string, string) {
	for _, lv := range l.levels {
		if strings.HasPrefix(line, lv) {
			return lv, strings.TrimSpace(line[len(lv):])
		}
		if strings.HasPrefix(line, "["+lv+"]") {
			return lv, strings.TrimSpace(line[len("["+lv+"]"):])
		}
	}
	return "INFO", line
}
