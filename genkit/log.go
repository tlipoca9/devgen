// Package genkit provides code generation utilities.
package genkit

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorGray    = "\033[90m"
)

// Emoji for log levels
const (
	EmojiInfo  = "📦"
	EmojiWarn  = "⚠️"
	EmojiError = "❌"
	EmojiDone  = "✅"
	EmojiFind  = "🔍"
	EmojiWrite = "📝"
	EmojiLoad  = "📂"
)

// Logger provides styled logging for code generators.
type Logger struct {
	w       io.Writer
	noColor bool
}

// NewLogger creates a new Logger writing to stdout.
func NewLogger() *Logger {
	return &Logger{w: os.Stdout}
}

// NewLoggerWithWriter creates a new Logger with custom writer.
func NewLoggerWithWriter(w io.Writer) *Logger {
	return &Logger{w: w}
}

// SetNoColor disables color output.
func (l *Logger) SetNoColor(noColor bool) *Logger {
	l.noColor = noColor
	return l
}

// format applies automatic highlighting to args based on type.
func (l *Logger) format(format string, args ...any) string {
	highlighted := make([]any, len(args))
	for i, arg := range args {
		highlighted[i] = l.highlight(arg)
	}
	return fmt.Sprintf(format, highlighted...)
}

// highlight applies color based on argument type.
func (l *Logger) highlight(arg any) any {
	if l.noColor {
		switch v := arg.(type) {
		case GoImportPath:
			return fmt.Sprintf("'%s'", v)
		case string:
			if strings.Contains(v, "/") || (strings.Contains(v, ".") && !strings.Contains(v, " ")) {
				return fmt.Sprintf("'%s'", v)
			}
			return v
		default:
			return arg
		}
	}
	switch v := arg.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%s%v%s", colorYellow, v, colorReset)
	case GoImportPath:
		return fmt.Sprintf("%s'%s'%s", colorMagenta, v, colorReset)
	case string:
		// Highlight paths (containing / or .)
		if strings.Contains(v, "/") || (strings.Contains(v, ".") && !strings.Contains(v, " ")) {
			return fmt.Sprintf("%s'%s'%s", colorMagenta, v, colorReset)
		}
		// Highlight identifiers (PascalCase or camelCase, no spaces)
		if !strings.Contains(v, " ") && len(v) > 0 && v[0] >= 'A' && v[0] <= 'Z' {
			return fmt.Sprintf("%s%s%s", colorCyan, v, colorReset)
		}
		return v
	default:
		return arg
	}
}

// color returns the color code if color is enabled, empty string otherwise.
func (l *Logger) color(c string) string {
	if l.noColor {
		return ""
	}
	return c
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s  %s[INFO]%s %s\n", EmojiInfo, l.color(colorBlue), l.color(colorReset), l.format(format, args...))
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s  %s[WARN]%s %s\n", EmojiWarn, l.color(colorYellow), l.color(colorReset), l.format(format, args...))
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s %s[ERROR]%s %s\n", EmojiError, l.color(colorRed), l.color(colorReset), l.format(format, args...))
}

// Done logs a completion message.
func (l *Logger) Done(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s  %s[DONE]%s %s\n", EmojiDone, l.color(colorGreen), l.color(colorReset), l.format(format, args...))
}

// Find logs a discovery message.
func (l *Logger) Find(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s  %s[FIND]%s %s\n", EmojiFind, l.color(colorCyan), l.color(colorReset), l.format(format, args...))
}

// Write logs a file write message.
func (l *Logger) Write(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s %s[WRITE]%s %s\n", EmojiWrite, l.color(colorGreen), l.color(colorReset), l.format(format, args...))
}

// Load logs a loading message.
func (l *Logger) Load(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "%s  %s[LOAD]%s %s\n", EmojiLoad, l.color(colorBlue), l.color(colorReset), l.format(format, args...))
}

// Item logs an indented item under the previous log entry.
func (l *Logger) Item(format string, args ...any) {
	_, _ = fmt.Fprintf(l.w, "           %s•%s %s\n", l.color(colorGray), l.color(colorReset), l.format(format, args...))
}
