// Package job は、Cloud Run Job として 1 回実行して終わる処理の入口。
package job

import "context"

// Executor はユースケースの実行。
type Executor interface {
	Execute(ctx context.Context) error
}

// Scan は ROLE=scan の入口。ビジネスロジックは持たず、ユースケースを 1 回呼ぶだけにする。
type Scan struct {
	UseCase Executor
}

// Run はスキャンを 1 回実行する。
func (s Scan) Run(ctx context.Context) error {
	return s.UseCase.Execute(ctx)
}
