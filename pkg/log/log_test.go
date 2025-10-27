package log

import (
	"bytes"
	"os"
	"testing"
)

func TestErrorMsg(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ErrorMsg("test error: %s", "something")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("ErrorMsg() produced no output")
	}
	if !bytes.Contains([]byte(output), []byte("test error")) {
		t.Errorf("ErrorMsg() output does not contain expected text: %q", output)
	}
}

func TestInfoMsg(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	InfoMsg("test info: %s", "something")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("InfoMsg() produced no output")
	}
	if !bytes.Contains([]byte(output), []byte("test info")) {
		t.Errorf("InfoMsg() output does not contain expected text: %q", output)
	}
}

func TestLogger_VerboseMsg_Enabled(t *testing.T) {
	logger := NewLogger(true)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger.VerboseMsg("verbose test: %s", "enabled")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("VerboseMsg() with verbose=true produced no output")
	}
	if !bytes.Contains([]byte(output), []byte("verbose test")) {
		t.Errorf("VerboseMsg() output does not contain expected text: %q", output)
	}
	if !bytes.Contains([]byte(output), []byte("[v]")) {
		t.Errorf("VerboseMsg() output does not contain [v] prefix: %q", output)
	}
}

func TestLogger_VerboseMsg_Disabled(t *testing.T) {
	logger := NewLogger(false)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger.VerboseMsg("verbose test: %s", "disabled")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("VerboseMsg() with verbose=false produced output: %q", output)
	}
}

func TestLogger_VerboseMsg_Nil(t *testing.T) {
	var logger *Logger

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger.VerboseMsg("verbose test: %s", "nil")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output != "" {
		t.Errorf("VerboseMsg() on nil Logger produced output: %q", output)
	}
}

func TestLogger_ErrorMsg(t *testing.T) {
	logger := NewLogger(false)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger.ErrorMsg("logger error: %s", "test")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Logger.ErrorMsg() produced no output")
	}
	if !bytes.Contains([]byte(output), []byte("logger error")) {
		t.Errorf("Logger.ErrorMsg() output does not contain expected text: %q", output)
	}
}

func TestLogger_InfoMsg(t *testing.T) {
	logger := NewLogger(false)

	// Capture stderr
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger.InfoMsg("logger info: %s", "test")

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if output == "" {
		t.Error("Logger.InfoMsg() produced no output")
	}
	if !bytes.Contains([]byte(output), []byte("logger info")) {
		t.Errorf("Logger.InfoMsg() output does not contain expected text: %q", output)
	}
}
