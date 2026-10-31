# kototech-line-notification

ことならテックプロジェクトのイベント情報を LINE 公式アカウントの Bot から発信するためのアプリケーション。

## 構成

| ディレクトリ | 内容 |
| --- | --- |
| `docs/` | 設計ドキュメントと、Flex Message のモック(今後追加) |
| `backend/` | バックエンドのコード (Go, Cloud Run Job) |
| `infra/` | Terraform(`bootstrap` / `env/{dev,prod}` / `modules`) |
| `scripts/` | ドキュメントと yml の検査(`make verify` から呼ばれる) |
| `.claude/` | 開発用のスキル等 AI 向け設定 |
| `CLAUDE.md` | AI (Claude Code) 向けのプロジェクト設定 |

## 概要

- 工程表(スプレッドシート)に合わせて、LINE Messaging API 経由で通知する
- 構成: Cloud Scheduler(毎分) → Cloud Run Job(Go) → Sheets API / LINE。GAS は使わない
- 配信メッセージは、MVP ではテキストメッセージの broadcast(文面のテンプレートは固定)。Flex Message 化は今後の候補で、`docs/mocks/` で設計してから行う
- 詳細な設計は [docs/](docs/README.md) を参照

## 開発を始めるとき

運用ルールの詳細は [docs/operations/01-operations.md](docs/operations/01-operations.md)。

1. リポジトリを取得する: `git clone https://github.com/kotonara-tech/kototech-line-notification.git`
2. 必須の `git config`(`merge.ff` と `pull.ff`)を設定する。コマンドと目的、force push のルールは [Git の設定と force push](docs/operations/01-operations.md#git-の設定と-force-push)
3. 作業は Issue から始める。作業ブランチを切る(名前の付け方と切る元は [作業ブランチ](docs/operations/01-operations.md#作業ブランチ)。通常は `develop` から、hotfix は `main` から)
4. コミット・push の前に、`make verify` を実行して、失敗を 0 にする([CLAUDE.md](CLAUDE.md) の方針)。Go・golangci-lint・Terraform・Node.js が要る(バージョンの目安は `.github/workflows/reusable-qa.yml`)
5. `develop` へ PR を作る([通常の開発フロー](docs/operations/01-operations.md#通常の開発フローフェーズ-2))

## ステータス

初期セットアップ中。
