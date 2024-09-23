package logrus

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

type LogLevel string

const (
	LevelError   LogLevel = "error"
	LevelWarn    LogLevel = "warn"
	LevelDebug   LogLevel = "debug"
	LevelSuccess LogLevel = "success"
	LevelInfo    LogLevel = "info"
)

var loggerInit *logrus.Logger

// ANSI color codes.
const (
	AnsiReset   = 0
	AnsiRed     = 31
	AnsiGreen   = 32
	AnsiYellow  = 33
	AnsiMagenta = 35
	AnsiCyan    = 36
	White       = 90
)

func (f *CustomTextFormatter) Color(level logrus.Level, s string) string {
	if f.DisableColor == true {
		return s
	}
	var levelColor int
	switch level {
	case logrus.DebugLevel:
		levelColor = AnsiCyan
	case logrus.WarnLevel:
		levelColor = AnsiYellow
	case logrus.ErrorLevel:
		levelColor = AnsiRed
	case logrus.PanicLevel:
		levelColor = AnsiMagenta
	case logrus.FatalLevel:
		levelColor = AnsiMagenta
	default:
		levelColor = AnsiGreen
	}
	if levelColor == AnsiReset {
		return s
	}
	// Colorize.
	return "\033[" + strconv.Itoa(levelColor) + "m" + s + "\033[0m"
}

func (f *CustomTextFormatter) ColorWith(levelColor int, s string) string {
	// Colorize.
	if f.DisableColor == true {
		return s
	}
	return "\033[" + strconv.Itoa(levelColor) + "m" + s + "\033[0m"
}

type Logger interface {
	Debug(message any)
	Info(message any)
	Warn(message any)
	Error(message any)
	Fatal(message any)

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

type CustomTextFormatter struct {
	DisableColor bool
}

func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	// Get the file and line number where the log was called
	_, filename, line, _ := runtime.Caller(7)

	// Get the script name from the full file path
	scriptName := filepath.Base(filename)

	var message strings.Builder
	// Format the log message
	message.WriteString(fmt.Sprintf("[%s] [%s] [%s:%d] %s",
		f.ColorWith(White, entry.Time.Format("2006-01-02 15:04:05")), // Date-time
		f.Color(entry.Level, entry.Level.String()),                   // Log level
		scriptName,    // Script name
		line,          // Line number
		entry.Message, // Log message
	))

	for key, val := range entry.Data {
		message.WriteString(f.ColorWith(White, fmt.Sprintf(" [%s=%v]", key, val)))
	}
	message.WriteString("\n")

	return []byte(message.String()), nil
}

type LoggerBuildProcess interface {
	SetContext(*context.Context) LoggerBuildProcess
	AddField(map[string]interface{}) LoggerBuildProcess
	Builder() Logger
}

type LogrusLogger struct {
	logger *logrus.Logger
	ctx    *context.Context
	fields map[string]interface{}
}

func (l *LogrusLogger) SetContext(ctx *context.Context) LoggerBuildProcess {

	if ctx != nil {
		if commonFields, ok := (*ctx).Value("commonFields").(map[string]interface{}); ok == true {
			for k, v := range commonFields {
				if _, ok := l.fields[k]; !ok {
					l.fields[k] = v
				}
			}
		}
	}
	l.ctx = ctx
	return l
}

func (l *LogrusLogger) AddField(fieldNew map[string]interface{}) LoggerBuildProcess {
	for k, v := range fieldNew {
		if _, ok := l.fields[k]; !ok {
			l.fields[k] = v
		}
	}
	return l
}

func (l *LogrusLogger) GetField(fieldKey string) interface{} {
	return l.fields[fieldKey]
}

func (l *LogrusLogger) Builder() Logger {
	return l
}

func LogrusLoggerWithContext(ctx *context.Context) *LogrusLogger {
	var logger *LogrusLogger
	if ctx != nil {
		if loggerCtx, ok := (*ctx).Value("logger").(*LogrusLogger); ok {
			logger = loggerCtx
		} else {
			logger = NewLogrusLogger()
		}

		if commonFields, ok := (*ctx).Value("commonFields").(map[string]interface{}); ok {
			logger.fields = commonFields
		} else {
			logger.fields = map[string]interface{}{}
		}
		logger.ctx = ctx
	}

	return logger

}

func InitLogrusLogger() {
	loggerInit = logrus.New()

	customFormatter := &CustomTextFormatter{}
	// Set the custom formatter as the formatter for the logger
	loggerInit.SetFormatter(customFormatter)
	loggerInit.SetLevel(logrus.TraceLevel)
}

func NewLogrusLogger() *LogrusLogger {
	loggerNew := loggerInit
	return &LogrusLogger{logger: loggerNew}
}

func (l *LogrusLogger) Debug(msg any) {
	l.logger.WithFields(l.fields).Debug(msg)
}

func (l *LogrusLogger) Info(msg any) {
	l.logger.WithFields(l.fields).Info(msg)
}

func (l *LogrusLogger) Warn(msg any) {
	l.logger.WithFields(l.fields).Warn(msg)
}

func (l *LogrusLogger) Error(msg any) {
	l.logger.WithFields(l.fields).Error(msg)
}

func (l *LogrusLogger) Fatal(msg any) {
	l.logger.WithFields(l.fields).Fatal(msg)
}

func (l *LogrusLogger) Debugf(format string, args ...any) {
	l.logger.WithFields(l.fields).Debugf(format, args...)
}

func (l *LogrusLogger) Infof(format string, args ...any) {
	l.logger.WithFields(l.fields).Infof(format, args...)
}

func (l *LogrusLogger) Warnf(format string, args ...any) {
	l.logger.WithFields(l.fields).Warnf(format, args...)
}

func (l *LogrusLogger) Errorf(format string, args ...any) {
	l.logger.WithFields(l.fields).Errorf(format, args...)
}

func (l *LogrusLogger) Fatalf(format string, args ...any) {
	l.logger.WithFields(l.fields).Fatalf(format, args...)
}
