// Package scan は、工程表を読み、開始間近のプログラムを通知するユースケース。
package scan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/application/port"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

// UseCase は scan のユースケース。
type UseCase struct {
	Programs program.Repository
	Notifier port.Notifier
	Clock    port.Clock
	Lead     program.LeadTime
	Logger   *slog.Logger
}

// Execute は工程表を 1 回スキャンして、通知すべきプログラムを通知する。
// 1 件の失敗で全体を止めず、他のプログラムの通知は続ける。失敗は最後にまとめて返す。
func (u *UseCase) Execute(ctx context.Context) error {
	logger := u.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	now := u.Clock.Now()
	programs, err := u.Programs.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("工程表の取得: %w", err)
	}
	// 開始が早い順に処理する(結果を決定的にするため)
	slices.SortStableFunc(programs, func(a, b *program.Program) int {
		return a.StartAt().Compare(b.StartAt())
	})

	var errs []error
	notified := 0
	for _, p := range programs {
		if err := ctx.Err(); err != nil {
			errs = append(errs, fmt.Errorf("中断: %w", err))
			break
		}
		if !p.IsDueForNotice(now, u.Lead) {
			continue
		}
		if err := u.notify(ctx, p, now); err != nil {
			errs = append(errs, fmt.Errorf("プログラム %q の通知: %w", p.Name(), err))
			continue
		}
		notified++
		logger.Info("通知した",
			slog.String("program_id", string(p.ID())),
			slog.String("program", p.Name()),
			slog.Time("start_at", p.StartAt()),
		)
	}

	logger.Info("スキャン完了",
		slog.Int("programs", len(programs)),
		slog.Int("notified", notified),
		slog.Int("failed", len(errs)),
	)
	return errors.Join(errs...)
}

// notify は送信し、通知済みを保存する。
// 送信後に保存へ失敗しても、次回の実行で同じキーの通知が重複として扱われ、二重送信にはならない。
func (u *UseCase) notify(ctx context.Context, p *program.Program, now time.Time) error {
	if err := u.Notifier.Notify(ctx, notice.NewStartingSoon(p)); err != nil {
		return fmt.Errorf("送信: %w", err)
	}
	if err := p.MarkNotified(now); err != nil {
		return fmt.Errorf("通知済みにする: %w", err)
	}
	if err := u.Programs.Save(ctx, p); err != nil {
		return fmt.Errorf("通知済みの保存: %w", err)
	}
	return nil
}
