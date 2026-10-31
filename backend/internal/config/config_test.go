package config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/config"
)

func lookup(env map[string]string) config.Lookup {
	return func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
}

func validScan() map[string]string {
	return map[string]string{
		"ROLE":                "scan",
		"APP_ENV":             "dev",
		"SPREADSHEET_ID":      "sheet-id",
		"LINE_CHANNEL_ID":     "1234567890",
		"LINE_CHANNEL_SECRET": "super-secret",
	}
}

func TestLoad_ValidScan(t *testing.T) {
	cfg, err := config.Load(lookup(validScan()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Role != config.RoleScan || cfg.AppEnv != "dev" || cfg.SpreadsheetID != "sheet-id" || cfg.LineChannelID != "1234567890" {
		t.Errorf("cfg = %+v", cfg)
	}
	if cfg.SheetName != "工程表" || cfg.LeadMinutes != 10 {
		t.Errorf("既定値が違う: %q, %d", cfg.SheetName, cfg.LeadMinutes)
	}
	if cfg.LineChannelSecret.Reveal() != "super-secret" {
		t.Error("シークレットの値が取れない")
	}
}

func TestLoad_Overrides(t *testing.T) {
	env := validScan()
	env["SHEET_NAME"] = "本番の工程表"
	env["NOTIFY_LEAD_MINUTES"] = "15"
	cfg, err := config.Load(lookup(env))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SheetName != "本番の工程表" || cfg.LeadMinutes != 15 {
		t.Errorf("cfg = %+v", cfg)
	}
}

func TestLoad_TrimsWhitespace(t *testing.T) {
	// GitHub の変数に、値の先頭の空白が入っていた事例への対策
	env := validScan()
	env["SPREADSHEET_ID"] = "  sheet-id \n"
	env["APP_ENV"] = " dev"
	cfg, err := config.Load(lookup(env))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.SpreadsheetID != "sheet-id" || cfg.AppEnv != "dev" {
		t.Errorf("空白が残っている: %q %q", cfg.SpreadsheetID, cfg.AppEnv)
	}
}

func TestLoad_ReportsAllProblemsAtOnce(t *testing.T) {
	_, err := config.Load(lookup(map[string]string{"ROLE": "scan"}))
	var cfgErr *config.Error
	if !errors.As(err, &cfgErr) {
		t.Fatalf("*config.Error のはず: %v", err)
	}
	for _, want := range []string{"APP_ENV", "SPREADSHEET_ID", "LINE_CHANNEL_ID", "LINE_CHANNEL_SECRET"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s の不備が報告されていない: %v", want, err)
		}
	}
}

func TestLoad_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(map[string]string)
		want   string
	}{
		{"ROLE が未設定", func(e map[string]string) { delete(e, "ROLE") }, "ROLE が未設定"},
		{"ROLE が未知", func(e map[string]string) { e["ROLE"] = "worker" }, "ROLE が不正"},
		{"APP_ENV が未知", func(e map[string]string) { e["APP_ENV"] = "staging" }, "APP_ENV が不正"},
		{"リードタイムが数値でない", func(e map[string]string) { e["NOTIFY_LEAD_MINUTES"] = "abc" }, "NOTIFY_LEAD_MINUTES"},
		{"リードタイムが 0", func(e map[string]string) { e["NOTIFY_LEAD_MINUTES"] = "0" }, "NOTIFY_LEAD_MINUTES"},
		{"リードタイムが大きすぎる", func(e map[string]string) { e["NOTIFY_LEAD_MINUTES"] = "100000" }, "NOTIFY_LEAD_MINUTES"},
		{"チャネル ID が空白のみ", func(e map[string]string) { e["LINE_CHANNEL_ID"] = "   " }, "LINE_CHANNEL_ID が未設定"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := validScan()
			tt.mutate(env)
			_, err := config.Load(lookup(env))
			var cfgErr *config.Error
			if !errors.As(err, &cfgErr) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q を含む *config.Error", err, tt.want)
			}
		})
	}
}

func TestLoad_APIRoleDoesNotRequireScanSettings(t *testing.T) {
	cfg, err := config.Load(lookup(map[string]string{"ROLE": "api", "APP_ENV": "prod"}))
	if err != nil || cfg.Role != config.RoleAPI {
		t.Errorf("%+v, %v", cfg, err)
	}
}

func TestSecret_NeverLeaks(t *testing.T) {
	s := config.Secret("super-secret")

	j, err := json.Marshal(struct{ S config.Secret }{s})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("確認", "secret", s, "cfg", config.Config{LineChannelSecret: s})

	outputs := map[string]string{
		"Sprint": fmt.Sprint(s),
		"書式化":    fmt.Sprintf("%v %+v %#v %s", s, s, s, s),
		"Config": fmt.Sprintf("%+v", config.Config{LineChannelSecret: s}),
		"JSON":   string(j),
		"slog":   logs.String(),
	}
	for name, out := range outputs {
		if strings.Contains(out, "super-secret") {
			t.Errorf("%s にシークレットが含まれている: %s", name, out)
		}
	}
}
