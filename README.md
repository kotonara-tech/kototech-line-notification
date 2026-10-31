# kototech-line-notification

ことならテックプロジェクトのイベント情報を LINE 公式アカウントの Bot から発信するためのアプリケーション。

## 構成

| ディレクトリ | 内容 |
| --- | --- |
| `docs/` | 設計ドキュメントと Flex Message のモック |
| `functions/` | Google Apps Script (GAS) で利用するコード |
| `backend/` | バックエンドのコード (Go) |
| `.claude/` | 開発用のスキル等 AI 向け設定 |
| `CLAUDE.md` | AI (Claude Code) 向けのプロジェクト設定 |

## 概要

- イベント情報を LINE Messaging API 経由で友だちに配信する
- 配信メッセージは Flex Message を利用する
- 詳細な設計は [docs/](docs/README.md) を参照

## ステータス

初期セットアップ中。
