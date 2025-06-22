package logger

import (
	"os"
	"testing"
)

func TestNewLogrusLogger_Success(t *testing.T) {
	// Use a temp file
	tmpfile, err := os.CreateTemp("", "testlog-*.log")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	log, err := NewLogrusLogger(tmpfile.Name())
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if log == nil {
		t.Fatal("expected logger, got nil")
	}
}

func TestNewLogrusLogger_Failure(t *testing.T) {
	// Intentionally invalid file path
	_, err := NewLogrusLogger("/invalid-path/does-not-exist.log")
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestWithFields(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	log, _ := NewLogrusLogger(tmpfile.Name())

	l := log.WithFields(map[string]any{"foo": "bar"})
	if l == nil {
		t.Fatal("expected logger with fields, got nil")
	}
}
