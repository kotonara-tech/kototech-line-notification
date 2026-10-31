# CLAUDE.md

## プロジェクト概要

ことならテックプロジェクトのイベント情報を発信する LINE 公式 Bot のアプリケーション。

## ディレクトリ

- `docs/engineering/` 設計ドキュメント
- `docs/mocks/` Flex Message の HTML モック(全シナリオ)
- `backend/` バックエンド (Go, Cloud Run Job)。`make test` / `lint` / `integration-test` / `build`
- `infra/terraform/` インフラ (`bootstrap` / `env/{dev,prod}` / `modules`)
- `docs/infra/` インフラ構成図
- `scripts/verify.mjs` ドキュメントと GitHub Actions(yml)の簡単な検査(`make verify` から呼ばれる)
- `.claude/skills/` 開発用スキル: `go-professional`(backend の Go)、`infra-professional`(Terraform / GCP)、`docs-professional`(ドキュメント)、`verify-professional`(完了前の検証)。該当する作業の前に参照する

## 方針

- 実装前に `docs/engineering/` の設計を確認・更新する
- Flex Message を変更したら `docs/mocks/` のモックも更新する
- backend は DDD の構成。依存は `domain` ← `application` ← `infrastructure` / `interfaces` の一方向で、`domain` は標準ライブラリのみに依存する。エントリポイントは環境変数 `ROLE`(scan / 将来 api)で切り替える([docs/engineering/03-backend.md](docs/engineering/03-backend.md))
- **作業が終わったら、報告・コミット・push の前に、必ず `make verify` を実行して、失敗を 0 にする**。GitHub Actions はローカルで動かせないため、`verify-professional` の論理検証のチェックリストで確認し、push 後は `gh run` で結果を確かめる
- インフラは Terraform で管理し、GitHub Actions + OIDC でデプロイする。GAS は使わない
- Terraform の `apply` は明示的に指示されるまで実行しない
- 作業は Issue から始める。ブランチは `<種類>/<Issue番号>-<短い説明>`(例: `chore/2-title-prefix`)。Issue・PR のタイトルは `【INFRA】【BE】【DOCS】【CORE】` で始める。PR の先頭に `Closes #番号` を書いて、必ず Issue と関連付ける。**1 PR で閉じるのは 1 Issue まで**([docs/operations/01-operations.md](docs/operations/01-operations.md))
- ブランチ運用: `develop` を起点に作業し、`main` は prod の写し。`develop` と `main` は Ruleset で保護済みで、**直 push はできない**(作業ブランチを切って PR を作る。`develop` 宛てはスカッシュ、`main` 宛て(リリース)はマージコミット)。他の人の Approve の必須化は、メンバーが揃ってから([docs/operations/01-operations.md](docs/operations/01-operations.md))
- 返答・ドキュメントは日本語で書く
- PR を作る・出すとき(タイトル・本文・宛先・Milestone・担当者・push・レビュワーに触れるとき)は、`pr-professional` の順番に従う。ルールの本体は [docs/operations/01-operations.md](docs/operations/01-operations.md) の「PR」
