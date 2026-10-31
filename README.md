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

## ステータス

初期セットアップ中。
