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
