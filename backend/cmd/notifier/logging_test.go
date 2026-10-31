package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewLogger_UsesCloudLoggingKeys(t *testing.T) {
	var buf bytes.Buffer
	logger := newLogger(&buf)
	logger.Debug("d")
	logger.Info("i")
	logger.Warn("w", "row", 5)
	logger.Error("e")

	want := []struct{ message, severity string }{{"i", "INFO"}, {"w", "WARNING"}, {"e", "ERROR"}}
	dec := json.NewDecoder(&buf)
	for _, w := range want {
		var entry map[string]any
		if err := dec.Decode(&entry); err != nil {
			t.Fatal(err)
		}
		if entry["message"] != w.message || entry["severity"] != w.severity {
			t.Errorf("entry = %v, want message=%s severity=%s", entry, w.message, w.severity)
		}
		if _, ok := entry["msg"]; ok {
			t.Error("msg ではなく message を使うはず")
		}
		if _, ok := entry["level"]; ok {
			t.Error("level ではなく severity を使うはず")
		}
	}
}
