// Package bootstrap は、具体的な実装を選んで、ユースケースに注入する(Composition Root)。
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/application/scan"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/config"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/infrastructure/clock"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/infrastructure/line"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/infrastructure/sheets"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/interfaces/job"
)

// timezone は、工程表の時刻を解釈するタイムゾーン。
const timezone = "Asia/Tokyo"

// ErrNotImplemented は、まだ実装していない ROLE が指定されたときのエラー。
var ErrNotImplemented = errors.New("未実装の ROLE")

// Runner は、役割ごとの入口。
type Runner interface {
	Run(ctx context.Context) error
}

// NewRunner は、ROLE に対応する Runner を作る。環境変数 ROLE による役割の切り替えは、ここにある。
// 役割を増やすときは、config.Role に値を足し、ここに case を足す(ビルドする Go バイナリは 1 つのまま)。
func NewRunner(ctx context.Context, cfg config.Config, logger *slog.Logger) (Runner, error) {
	switch cfg.Role {
	case config.RoleScan:
		return newScan(ctx, cfg, logger)
	default:
		return nil, fmt.Errorf("%w: %s", ErrNotImplemented, cfg.Role)
	}
}

func newScan(ctx context.Context, cfg config.Config, logger *slog.Logger) (Runner, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("タイムゾーン %s の読み込み: %w", timezone, err)
	}
	lead, err := program.NewLeadTime(cfg.LeadMinutes)
	if err != nil {
		return nil, err
	}
	repo, err := sheets.New(ctx, sheets.Config{
		SpreadsheetID: cfg.SpreadsheetID,
		SheetName:     cfg.SheetName,
		Location:      loc,
		Logger:        logger,
	})
	if err != nil {
		return nil, err
	}
	notifier := line.NewNotifier(line.Config{
		Credentials: line.Credentials{
			ChannelID:     cfg.LineChannelID,
			ChannelSecret: cfg.LineChannelSecret.Reveal(),
		},
		Logger: logger,
	})

	return job.Scan{UseCase: &scan.UseCase{
		Programs: repo,
		Notifier: notifier,
		Clock:    clock.New(loc),
		Lead:     lead,
		Logger:   logger,
	}}, nil
}
