package main

import "testing"

func TestRun_ExitCodes(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want int
	}{
		{"ROLE が未設定なら設定エラー", map[string]string{"APP_ENV": "dev"}, exitBadSetup},
		{"ROLE が未知なら設定エラー", map[string]string{"ROLE": "worker", "APP_ENV": "dev"}, exitBadSetup},
		{"scan で必須の設定が足りなければ設定エラー", map[string]string{"ROLE": "scan", "APP_ENV": "dev"}, exitBadSetup},
		{"api は未実装なので設定エラー", map[string]string{"ROLE": "api", "APP_ENV": "dev"}, exitBadSetup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 他の環境変数の影響を受けないよう、関係する変数をすべて空にしてから設定する
			for _, k := range []string{"ROLE", "APP_ENV", "SPREADSHEET_ID", "LINE_CHANNEL_ID", "LINE_CHANNEL_SECRET", "SHEET_NAME", "NOTIFY_LEAD_MINUTES"} {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := run(); got != tt.want {
				t.Errorf("run() = %d, want %d", got, tt.want)
			}
		})
	}
}
