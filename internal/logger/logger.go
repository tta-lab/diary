package logger

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

func Setup() (func() error, error) {
	// Create logger
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
	})

	// Get log file path
	logDir := filepath.Join(os.Getenv("HOME"), ".diary", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Log disabled, use stderr only
		Logger.SetLevel(log.WarnLevel)
		return func() error { return nil }, nil
	}

	logFile := filepath.Join(logDir, "diary.log")
	f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		// Log disabled, use stderr only
		Logger.SetLevel(log.WarnLevel)
		return func() error { return nil }, nil
	}

	// Log to file
	Logger.SetOutput(f)
	Logger.SetLevel(log.DebugLevel)
	Logger.Info("diary logger initialized", "logfile", logFile)

	return f.Close, nil
}

// Debug logs when logger is nil
func Debug(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Debug(msg, keyvals...)
	}
}

func Info(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Info(msg, keyvals...)
	}
}

func Error(msg string, keyvals ...interface{}) {
	if Logger != nil {
		Logger.Error(msg, keyvals...)
	}
}
