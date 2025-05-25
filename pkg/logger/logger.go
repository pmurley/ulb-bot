package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

type Logger struct {
	level  Level
	logger *log.Logger
}

func New(levelStr string) *Logger {
	level := parseLevel(levelStr)
	return &Logger{
		level:  level,
		logger: log.New(os.Stdout, "", 0),
	}
}

func parseLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DebugLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

func (l *Logger) log(level Level, prefix string, v ...interface{}) {
	if level >= l.level {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		msg := fmt.Sprint(v...)
		l.logger.Printf("[%s] %s %s", timestamp, prefix, msg)
	}
}

func (l *Logger) Debug(v ...interface{}) {
	l.log(DebugLevel, "[DEBUG]", v...)
}

func (l *Logger) Info(v ...interface{}) {
	l.log(InfoLevel, "[INFO]", v...)
}

func (l *Logger) Warn(v ...interface{}) {
	l.log(WarnLevel, "[WARN]", v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.log(ErrorLevel, "[ERROR]", v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.log(ErrorLevel, "[FATAL]", v...)
	os.Exit(1)
}