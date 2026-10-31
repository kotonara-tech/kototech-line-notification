// Package config は、環境変数の読み込みと検証を行う。
// 他のパッケージは os.Getenv を呼ばず、ここで作った Config を受け取る。
package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

// Role はこのプロセスの役割。
type Role string

// 役割の一覧。
const (
	RoleScan Role = "scan"
	RoleAPI  Role = "api"
)

const (
	defaultSheetName   = "工程表"
	defaultLeadMinutes = 10
	maxLeadMinutes     = 24 * 60
)

// Secret は秘匿情報。ログや書式化の出力には出ない。値が必要なときだけ Reveal を呼ぶ。
type Secret string

const masked = "****"

// String は値を隠す。
func (Secret) String() string { return masked }

// GoString は %#v でも値を隠す。
func (Secret) GoString() string { return masked }

// MarshalJSON は JSON にも値を出さない。
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + masked + `"`), nil }

// LogValue は slog にも値を出さない。
func (Secret) LogValue() slog.Value { return slog.StringValue(masked) }

// Reveal は実際の値を返す。
func (s Secret) Reveal() string { return string(s) }

// Config はアプリの設定。
type Config struct {
	Role   Role
	AppEnv string

	// ROLE=scan で使う設定
	SpreadsheetID     string
	SheetName         string
	LeadMinutes       int
	LineChannelID     string
	LineChannelSecret Secret
}

// Error は設定の不備。足りない項目などを、まとめて報告する。
type Error struct {
	Problems []string
}

func (e *Error) Error() string {
	return "設定が不正: " + strings.Join(e.Problems, "; ")
}

// Lookup は環境変数を引く関数(os.LookupEnv と同じ形)。
type Lookup func(key string) (string, bool)

// Load は環境変数から設定を作る。ROLE ごとに必須の項目を検証し、不備はまとめて返す。
// 値の前後の空白は取り除く(設定の入力ミスで、値の先頭に空白が入ることがあるため)。
func Load(lookup Lookup) (Config, error) {
	get := func(key string) string {
		v, _ := lookup(key)
		return strings.TrimSpace(v)
	}

	var problems []string
	require := func(key string) string {
		v := get(key)
		if v == "" {
			problems = append(problems, key+" が未設定")
		}
		return v
	}

	cfg := Config{AppEnv: get("APP_ENV")}

	switch role := Role(get("ROLE")); role {
	case RoleScan, RoleAPI:
		cfg.Role = role
	case "":
		problems = append(problems, "ROLE が未設定(scan / api のいずれか)")
	default:
		problems = append(problems, fmt.Sprintf("ROLE が不正: %q(scan / api のいずれか)", role))
	}

	switch cfg.AppEnv {
	case "dev", "prod", "local":
	case "":
		problems = append(problems, "APP_ENV が未設定(dev / prod / local のいずれか)")
	default:
		problems = append(problems, fmt.Sprintf("APP_ENV が不正: %q(dev / prod / local のいずれか)", cfg.AppEnv))
	}

	if cfg.Role == RoleScan {
		cfg.SpreadsheetID = require("SPREADSHEET_ID")
		cfg.LineChannelID = require("LINE_CHANNEL_ID")
		cfg.LineChannelSecret = Secret(require("LINE_CHANNEL_SECRET"))

		cfg.SheetName = get("SHEET_NAME")
		if cfg.SheetName == "" {
			cfg.SheetName = defaultSheetName
		}

		cfg.LeadMinutes = defaultLeadMinutes
		if raw := get("NOTIFY_LEAD_MINUTES"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > maxLeadMinutes {
				problems = append(problems, fmt.Sprintf("NOTIFY_LEAD_MINUTES が不正: %q(1〜%d の整数)", raw, maxLeadMinutes))
			} else {
				cfg.LeadMinutes = n
			}
		}
	}

	if len(problems) > 0 {
		return Config{}, &Error{Problems: problems}
	}
	return cfg, nil
}
