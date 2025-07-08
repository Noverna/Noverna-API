package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

type LogLevel int


const (
	DEBUG LogLevel = iota 
	INFO
	WARN
	ERROR
	FATAL
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorGreen  = "\033[32m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Function  string                 `json:"function,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

type Logger struct {
	level      LogLevel
	output     io.Writer
	mu         sync.RWMutex
	jsonFormat bool
	colorize   bool
	showCaller bool
	fields     map[string]any
}

func NewLogger() *Logger {
	return &Logger{
		level:      INFO,
		output:     os.Stdout,
		jsonFormat: false,
		colorize:   true,
		showCaller: true,
		fields:     make(map[string]any),
	}
}

func (l *Logger) SetLevel(level LogLevel) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	return l
}

func (l *Logger) SetOutput(w io.Writer) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
	return l
}

func (l *Logger) SetJSONFormat(enabled bool) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.jsonFormat = enabled
	return l
}

func (l *Logger) SetColorize(enabled bool) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.colorize = enabled
	return l
}

func (l *Logger) SetShowCaller(enabled bool) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.showCaller = enabled
	return l
}

func (l *Logger) WithField(key string, value any) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fields[key] = value
	return l
}

func (l *Logger) WithFields(fields map[string]any) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, v := range fields {
		l.fields[k] = v
	}
	return l
}

func (l *Logger) getCaller() (string, int, string) {
	pc, file, line, ok := runtime.Caller(3)
	if !ok {
		return "", 0, ""
	}
	
	parts := strings.Split(file, "/")
	filename := parts[len(parts)-1]
	
	funcName := runtime.FuncForPC(pc).Name()
	parts = strings.Split(funcName, ".")
	funcName = parts[len(parts)-1]
	
	return filename, line, funcName
}

func (l *Logger) getColor(level LogLevel) string {
	if !l.colorize {
		return ""
	}
	
	switch level {
	case DEBUG:
		return ColorCyan
	case INFO:
		return ColorGreen
	case WARN:
		return ColorYellow
	case ERROR:
		return ColorRed
	case FATAL:
		return ColorPurple
	default:
		return ColorWhite
	}
}

func (l *Logger) log(level LogLevel, message string, fields map[string]any) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	if level < l.level {
		return
	}
	
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Fields:    make(map[string]any),
	}
	
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	
	for k, v := range fields {
		entry.Fields[k] = v
	}
	
	if l.showCaller {
		file, line, function := l.getCaller()
		entry.File = file
		entry.Line = line
		entry.Function = function
	}
	
	if l.jsonFormat {
		l.writeJSON(entry)
	} else {
		l.writeText(entry, level)
	}
}

func (l *Logger) writeJSON(entry LogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger error: %v\n", err)
		return
	}
	
	fmt.Fprintln(l.output, string(data))
}

func (l *Logger) writeText(entry LogEntry, level LogLevel) {
	color := l.getColor(level)
	reset := ""
	if l.colorize {
		reset = ColorReset
	}
	
	var output strings.Builder
	
	output.WriteString(fmt.Sprintf("%s[%s]%s %s%s%s",
		color, entry.Timestamp, reset,
		color, entry.Level, reset))
	
	if l.showCaller && entry.File != "" {
		output.WriteString(fmt.Sprintf(" %s(%s:%d %s)%s",
			ColorWhite, entry.File, entry.Line, entry.Function, reset))
	}
	
	output.WriteString(fmt.Sprintf(" %s", entry.Message))
	
	if len(entry.Fields) > 0 {
		output.WriteString(" ")
		for k, v := range entry.Fields {
			output.WriteString(fmt.Sprintf("%s=%v ", k, v))
		}
	}
	
	fmt.Fprintln(l.output, output.String())
}

func (l *Logger) Debug(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, message, f)
}

func (l *Logger) Info(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, message, f)
}

func (l *Logger) Warn(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, message, f)
}

func (l *Logger) Error(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ERROR, message, f)
}

func (l *Logger) Fatal(message string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(FATAL, message, f)
	os.Exit(1)
}

var defaultLogger = NewLogger()

func SetLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

func SetOutput(w io.Writer) {
	defaultLogger.SetOutput(w)
}

func SetJSONFormat(enabled bool) {
	defaultLogger.SetJSONFormat(enabled)
}

func Debug(message string, fields ...map[string]any) {
	defaultLogger.Debug(message, fields...)
}

func Info(message string, fields ...map[string]any) {
	defaultLogger.Info(message, fields...)
}

func Warn(message string, fields ...map[string]any) {
	defaultLogger.Warn(message, fields...)
}

func Error(message string, fields ...map[string]any) {
	defaultLogger.Error(message, fields...)
}

func Fatal(message string, fields ...map[string]any) {
	defaultLogger.Fatal(message, fields...)
}