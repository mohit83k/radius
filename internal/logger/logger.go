package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Logger is our abstract logging interface.
type Logger interface {
	Info(msg string)
	Error(err error)
	WithFields(fields map[string]any) Logger
}

// LogrusLogger implements Logger using logrus.
type LogrusLogger struct {
	entry *logrus.Entry
}

// NewLogrusLogger creates a new file-based logrus logger.
func NewLogrusLogger(filepath string) (Logger, error) {
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	mw := io.MultiWriter(os.Stdout, file)

	baseLogger := logrus.New()
	baseLogger.SetOutput(mw)
	baseLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000000",
	})
	baseLogger.SetLevel(logrus.InfoLevel)

	return &LogrusLogger{
		entry: logrus.NewEntry(baseLogger),
	}, nil
}

func (l *LogrusLogger) Info(msg string) {
	l.entry.Info(msg)
}

func (l *LogrusLogger) Error(err error) {
	l.entry.Error(err)
}

func (l *LogrusLogger) WithFields(fields map[string]any) Logger {
	return &LogrusLogger{
		entry: l.entry.WithFields(logrus.Fields(fields)),
	}
}
