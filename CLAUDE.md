# CLAUDE.md

## プロジェクト概要

ことならテックプロジェクトのイベント情報を発信する LINE 公式 Bot のアプリケーション。

## ディレクトリ

- `docs/engineering/` 設計ドキュメント
- `docs/mocks/` Flex Message の HTML モック(全シナリオ)
- `backend/` バックエンド (Go, Cloud Run Job)。`make test` / `lint` / `integration-test` / `build`
- `infra/terraform/` インフラ (`bootstrap` / `env/{dev,prod}` / `modules`)
- `docs/infra/` インフラ構成図
- `.claude/skills/` 開発用スキル: `go-professional`(backend の Go)、`infra-professional`(Terraform / GCP)、`docs-professional`(ドキュメント)。該当する作業の前に参照する

## 方針

- 実装前に `docs/engineering/` の設計を確認・更新する
- Flex Message を変更したら `docs/mocks/` のモックも更新する
- backend は DDD の構成。依存は `domain` ← `application` ← `infrastructure` / `interfaces` の一方向で、`domain` は標準ライブラリのみに依存する。エントリポイントは環境変数 `ROLE`(scan / 将来 api)で切り替える([docs/engineering/03-backend.md](docs/engineering/03-backend.md))
- インフラは Terraform で管理し、GitHub Actions + OIDC でデプロイする。GAS は使わない
- Terraform の `apply` は明示的に指示されるまで実行しない
- ブランチ運用: `develop` を起点に作業し、`main` は prod の写し。開発が安定するまでは `develop` への直 push 可で、安定後に `main` を保護して PR + 他者の Approve を必須にする([docs/operations/01-operations.md](docs/operations/01-operations.md))
- 返答・ドキュメントは日本語で書く
