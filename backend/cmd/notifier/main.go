// notifier は、工程表を読み、開始間近のプログラムを LINE へ broadcast する。
//
// 環境変数 ROLE で役割を切り替える(MVP は scan のみ。api は将来)。切り替えは次の 2 か所にある:
//   - internal/config:    ROLE を読み、scan / api 以外や未設定をエラー(終了コード 2)にする
//   - internal/bootstrap: NewRunner の switch が、ROLE に対応する Runner を選ぶ
//
// このファイルは、設定を読み、Runner を実行し、結果を終了コードにするだけで、役割ごとの処理は持たない。
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	// 最小のコンテナイメージにはタイムゾーンのデータがないため、埋め込む
	_ "time/tzdata"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/bootstrap"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/config"
)

// 終了コード。
const (
	exitOK       = 0
	exitFailure  = 1 // 処理の失敗
	exitBadSetup = 2 // 設定エラー
)

func main() {
	os.Exit(run())
}

// run は処理を行い、終了コードを返す。os.Exit は main の 1 か所だけで呼ぶ(defer を確実に実行するため)。
func run() int {
	logger := newLogger(os.Stdout)

	// Cloud Run Job は、タイムアウトやキャンセルで SIGTERM を受ける
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(os.LookupEnv)
	if err != nil {
		logger.Error("設定エラー", slog.String("error", err.Error()))
		return exitBadSetup
	}
	logger = logger.With(slog.String("role", string(cfg.Role)), slog.String("app_env", cfg.AppEnv))

	// ROLE に対応する Runner は、bootstrap.NewRunner が選ぶ(scan なら、工程表のスキャン)。
	// 役割を増やすときは、config.Role と bootstrap.NewRunner の switch に足す
	runner, err := bootstrap.NewRunner(ctx, cfg, logger)
	if err != nil {
		logger.Error("起動に失敗", slog.String("error", err.Error()))
		if errors.Is(err, bootstrap.ErrNotImplemented) {
			return exitBadSetup
		}
		return exitFailure
	}

	if err := runner.Run(ctx); err != nil {
		logger.Error("処理に失敗", slog.String("error", err.Error()))
		return exitFailure
	}
	return exitOK
}
