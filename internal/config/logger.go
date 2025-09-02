package config

import (
	"fmt"
	"log"
	"os"
)

type Logger struct {
	level LogLevel
}

var defaultLogger *Logger

// InitLogger initializes the default logger with the specified log level
func InitLogger(level LogLevel) {
	defaultLogger = &Logger{level: level}
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	if defaultLogger == nil {
		defaultLogger = &Logger{level: LogLevelInfo}
	}
	return defaultLogger
}

// SetLevel updates the log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// Error logs error level messages
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level >= LogLevelError {
		log.Printf("[ERROR] "+format, args...)
	}
}

// Info logs info level messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level >= LogLevelInfo {
		log.Printf("[INFO] "+format, args...)
	}
}

// Verbose logs verbose level messages
func (l *Logger) Verbose(format string, args ...interface{}) {
	if l.level >= LogLevelVerbose {
		log.Printf("[VERBOSE] "+format, args...)
	}
}

// Fatal logs error and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	log.Printf("[FATAL] "+format, args...)
	os.Exit(1)
}

// Convenience functions for global logger
func LogError(format string, args ...interface{}) {
	GetLogger().Error(format, args...)
}

func LogInfo(format string, args ...interface{}) {
	GetLogger().Info(format, args...)
}

func LogVerbose(format string, args ...interface{}) {
	GetLogger().Verbose(format, args...)
}

func LogFatal(format string, args ...interface{}) {
	GetLogger().Fatal(format, args...)
}

func SetLogLevel(level LogLevel) {
	GetLogger().SetLevel(level)
}