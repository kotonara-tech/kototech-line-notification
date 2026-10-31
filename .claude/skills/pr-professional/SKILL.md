---
name: pr-professional
description: "PR(プルリクエスト)を作る・出す・直すときの、作る前から出すまでの順番と、このプロジェクトのルールへの参照先(タイトルのプレフィックス・本文の `Closes #`・宛先とマージ方法・Milestone・担当者・push・レビュアー)。`gh pr create` で PR を作る、PR のタイトル・本文・Milestone・担当者・レビュアーに触れる、PR を出すために push する、といった作業では、ユーザーが「PR」と明示しなくても必ず使うこと。対象は作業ブランチ → `develop` の PR。リリース PR(`develop` → `main`)や hotfix の PR は、手順 4 の宛先とマージ方法を使う。"
---

# pr-professional

PR を出すときに、**作る前から出すまでに確認する順番**をまとめる。PR のルールは複数の場所に分かれているので、ここから一度に辿れるようにする。

**ルールの本体は `docs/operations/01-operations.md`**。ここには、順番と参照先だけを書く(同じ内容を二重に書かない)。ルールを変えるときは、本体を直す。

## 手順

上から順に確認して、PR を出す。

| 順 | 確認すること | ルールの本体(`docs/operations/01-operations.md`) |
| --- | --- | --- |
| 1 | **作る前**: Issue があるか(なければ先に立てる)。ブランチ名が `<種類>/<Issue番号>-<短い説明>` か。`make verify` を実行して、失敗を 0 にしたか(`verify-professional`) | 「作業ブランチ」 |
| 2 | **タイトル**: 領域のプレフィックス(`【INFRA】` / `【BE】` / `【DOCS】` / `【CORE】`)で始めたか | 「Issue・PR・コミットの書き方」の「タイトルのプレフィックス」 |
| 3 | **本文**: `.github/pull_request_template.md` の項目を埋めたか。**先頭**に `Closes #番号`(必須)。**1 PR で閉じるのは 1 Issue まで**。ほかの Issue に関わる(閉じない)ときは `Ref #番号` | 「Issue・PR・コミットの書き方」の「PR」 |
| 4 | **宛先とマージ方法**: 作業ブランチ → `develop` は**スカッシュ**。`main` 宛ては**マージコミット**(リリース PR は `develop` → `main`、hotfix の PR は `hotfix/*` → `main`)。hotfix のバックマージ(`main` → `develop`)は、`develop` 宛てなのでスカッシュ | 「ブランチ保護(Ruleset)」の「マージ方法(設定済み)」、「リリースフロー(prod)」、「hotfix フロー(フェーズ 2)」 |
| 5 | **Milestone と担当者**: Issue と同じ Milestone を付けたか(Issue に無ければ、先に Issue に付ける)。担当者は、作成者自身にしたか | 「Issue・PR・コミットの書き方」の「PR」 |
| 6 | **push**: force push が要るときは、`--force` ではなく `--force-with-lease`。必須の `git config` を設定済みか | 本体は、別の Issue(#19)で `01-operations.md` に書く。書かれたら、ここに節名を足す |
| 7 | **レビュアー**: **自動ではリクエストしない**(通知が飛ぶため)。人が必要なときに指定する | 「Issue・PR・コミットの書き方」の「PR」 |

- 手順 3・5 は、作業ブランチ → `develop` の PR の規則。**リリース PR は、閉じる Issue を持たないことがあるため、手順 3・5 の対象外**(`Closes` と Milestone の付け方は、本体に定めがない)
- hotfix の PR も、Issue から始める(手順 1)ので、手順 3・5 に従う

## 出し方の例

```
gh pr create \
  --base develop \
  --title "【DOCS】…" \
  --body-file <テンプレートを埋めた本文のファイル> \
  --milestone "<Issue と同じ Milestone 名>" \
  --assignee "@me"
```

- `--base` で宛先を明示する(手順 4)
- `--body-file` に渡す本文は、`Closes #番号` から始める
- Milestone 名は、閉じる Issue から読める: `gh issue view <Issue番号> --json milestone -q .milestone.title`。出力が空なら、Issue に Milestone が無い。先に Issue に付けてから、`--milestone` に渡す
- `--assignee "@me"` が、作成者自身になる
- **`--reviewer` は付けない**(手順 7)

## 出したあと

`ci` の結果を確かめる。見かたは `verify-professional` の「push したあとに、実環境で確かめる」を参照。
