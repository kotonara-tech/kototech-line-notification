package main

import (
	"io"
	"log/slog"
)

// newLogger は、Cloud Logging が構造化ログとして解釈できる JSON のロガーを作る。
// slog の既定のキー(level / msg)のままだと、Cloud Logging が重大度とメッセージを認識せず、
// すべて INFO 扱いになって、エラーだけを絞り込めないため、severity / message に直す。
func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{ReplaceAttr: cloudLoggingAttr}))
}

func cloudLoggingAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.MessageKey:
		a.Key = "message"
	case slog.LevelKey:
		a.Key = "severity"
		if l, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(severity(l))
		}
	}
	return a
}

// severity は、slog のレベルを Cloud Logging の重大度の名前にする。
func severity(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARNING"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}
