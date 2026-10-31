// Package port は、ユースケースが外部サービスを使うためのインターフェースを定義する。
package port

import (
	"context"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
)

// Notifier は通知を送る。
type Notifier interface {
	Notify(ctx context.Context, n notice.Notice) error
}

// Clock は現在時刻を返す。
type Clock interface {
	Now() time.Time
}
