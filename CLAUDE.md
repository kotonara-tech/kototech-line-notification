# CLAUDE.md

## プロジェクト概要

ことならテックプロジェクトのイベント情報を発信する LINE 公式 Bot のアプリケーション。

## ディレクトリ

- `docs/engineering/` 設計ドキュメント
- `docs/mocks/` Flex Message の HTML モック(全シナリオ)
- `functions/` GAS 用コード
- `backend/` バックエンド (Go)
- `.claude/` 開発用スキル

## 方針

- 実装前に `docs/engineering/` の設計を確認・更新する
- Flex Message を変更したら `docs/mocks/` のモックも更新する
- 返答・ドキュメントは日本語で書く
