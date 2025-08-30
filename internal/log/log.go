package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARNING
	ERROR
)

type LogMessage struct {
	Timestamp time.Time
	Module    string
	Level     LogLevel
	Message   string
}

type Logger struct {
	MinLevel LogLevel
	Name     string
}

var logFile *os.File = nil
var AllLoggers []*Logger = make([]*Logger, 0)
var flushStopCh chan struct{}

func SetupLogging(filename string) {
	// Ensure the directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create log directory: %v", err))
	}

	// Note: We remove O_TRUNC to append to the file instead of truncating
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	logFile = f

	// Start background flusher goroutine
	flushStopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if logFile != nil {
					logFile.Sync()
				}
			case <-flushStopCh:
				return
			}
		}
	}()
}

func CleanupLogging() {
	// Stop the flusher goroutine
	if flushStopCh != nil {
		close(flushStopCh)
		flushStopCh = nil
	}

	if logFile != nil {
		logFile.Sync() // Final flush before closing
		logFile.Close()
		logFile = nil
	}
}

func NewLogger(name string) *Logger {
	logger := &Logger{DEBUG, name}
	AllLoggers = append(AllLoggers, logger)
	return logger
}

func levelToString(level LogLevel) string {
	switch level {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARNING:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		panic("Unsupported log level")
	}
}

func (logger *Logger) log(level LogLevel, format string, args ...interface{}) {
	if logFile != nil && logger != nil && level >= logger.MinLevel {
		t := fmt.Sprintf(format, args...)
		_, file, line, _ := runtime.Caller(2)
		logLine := fmt.Sprintf("%s %6s %5s %.10s:%4d: %s\n",
			time.Now().Format("2006-01-02T15:04:05.000000"), logger.Name, levelToString(level),
			filepath.Base(file), line, t)
		logFile.Write([]byte(logLine))
	}
}

func (logger *Logger) Trace(format string, args ...interface{}) {
	logger.log(TRACE, format, args...)
}

func (logger *Logger) Debug(format string, args ...interface{}) {
	logger.log(DEBUG, format, args...)
}

func (logger *Logger) Info(format string, args ...interface{}) {
	logger.log(INFO, format, args...)
}

func (logger *Logger) Warning(format string, args ...interface{}) {
	logger.log(WARNING, format, args...)
}

func (logger *Logger) Error(format string, args ...interface{}) {
	logger.log(ERROR, format, args...)
}

// SetLevel sets the minimum log level for this logger
func (logger *Logger) SetLevel(level LogLevel) {
	logger.MinLevel = level
}

// SetAllLogLevel sets the log level for all loggers
func SetAllLogLevel(level LogLevel) {
	for _, logger := range AllLoggers {
		logger.SetLevel(level)
	}
}

// ParseLogLevel parses a string log level into a LogLevel
func ParseLogLevel(levelStr string) (LogLevel, error) {
	switch levelStr {
	case "trace":
		return TRACE, nil
	case "debug":
		return DEBUG, nil
	case "info":
		return INFO, nil
	default:
		return INFO, fmt.Errorf("invalid log level: %s (supported: info, debug, trace)", levelStr)
	}
}
