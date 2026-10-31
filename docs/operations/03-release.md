# 03. リリース手順

`develop` の内容を prod に出し、バージョンタグとリリースノートを残す手順書。ブランチの役割、マージ方法、バージョニングの考え方は [01-operations.md](01-operations.md) が正本で、ここには、**誰が、何を、どの順で、どのコマンドで行い、何を確認するか**だけを書く。

リリースは、`develop` → `main` の PR のマージで始まる。`main` への push で `deploy-to-prod` が起動し、QA → インフラ(Terraform)→ アプリのデプロイの順に、prod へ反映する([../engineering/02-infra.md](../engineering/02-infra.md) の「CD」)。

## 前提

| 役割 | すること | 必要なもの |
| --- | --- | --- |
| 実施者 | リリース PR の作成とマージ、タグとリリースの作成、確認 | リポジトリの書き込み権限。`gh`(ログイン済み)と `git`。確認には、prod の GCP プロジェクトを読み取れる `gcloud` のアカウント |
| 承認者 | `deploy-to-prod` の承認待ちのジョブを承認する | Environment `prod` の Required reviewers に設定されていること |
| オーナー | 失敗したあとの復旧([02-recovery.md](02-recovery.md)) | 手元のオーナー権限 |

- コマンドは、このリポジトリのクローンの中で実行する(`gh api` の `{owner}/{repo}` は、`gh` が補う)
- `<PROJECT_ID>`・`<REGION>`・`<JOB_NAME>` は、Environment `prod` の変数 `GCP_PROJECT_ID`・`REGION`・`CLOUD_RUN_JOB_NAME` の値に置き換える(`gh variable list --env prod` で読める)。`<ID>` は、`gh run list` に出る実行の ID
- `main` は Ruleset で保護され、直 push はできない。バイパスもない
- この手順は、フェーズ 2 の運用を前提にする([01-operations.md](01-operations.md) の「運用フェーズ」)
- 秘匿情報(Secret の値、トークン)は、この手順では使わない。読まない

## 事前確認

リリース PR を作る前に、次を確かめる。

1. dev で動作を確認済みで、`deploy-to-dev` が成功していること

   ```
   gh run list --workflow deploy-to-dev.yml --branch develop --limit 3
   ```

2. リリースに含まれる変更が、意図したものと合っていること。`develop` のコミットは、スカッシュした PR 1 件ずつ

   ```
   git fetch origin
   git log --oneline origin/main..origin/develop
   git diff --stat origin/main origin/develop
   ```

3. 次の変更が含まれるときは、重点的にレビューする

   ```
   git diff --stat origin/main origin/develop -- infra/ .github/
   ```

   | 差分 | 見ること |
   | --- | --- |
   | `infra/` | prod の `terraform plan` は、PR 上では実行できない(prod は `main` からのみ)。代わりに、`deploy-to-dev` の `infra` ジョブのステップサマリ(`terraform plan`)で、削除(`delete`)や作り直し(`replace`)がないかを読む。dev と prod は state が違うので、同じ計画になるとは限らない。削除を含む計画は、prod では `infra` ジョブで止まる(「失敗したとき」) |
   | `.github/` | マージした `main` の workflow が、そのまま prod のデプロイに使われる。内容と同じ重さで読む |

4. prod の変数と、承認の設定を読む

   ```
   gh variable list --env prod
   gh api repos/{owner}/{repo}/environments/prod --jq '.protection_rules'
   ```

   - 変数の値の前後に、空白がないこと(v1.0.0 では `GCP_PROJECT_ID` の先頭の空白で、prod のリソースが作り直された)。デプロイの最初に `.github/actions/check-vars` も検査して失敗させるが、手元で先に見る
   - `SCHEDULER_PAUSED` が、意図した値であること。`terraform apply` が、この値(未設定は `true`)を Scheduler の一時停止の状態に設定する。リリースの手順に、Scheduler の再開は含まれない([../engineering/02-infra.md](../engineering/02-infra.md) の「Cloud Scheduler」)
   - `protection_rules` に、承認者(`required_reviewers`)が入っていること。空のときは、承認を挟まずに、マージの直後に prod へ反映される。01 と `02-infra.md` は承認ありを前提にしているので、空ならリリースの前に設定を直す

## 手順(リリース)

1. リリース PR を作る。タイトルと本文は、[01-operations.md](01-operations.md) の「Issue・PR・コミットの書き方」に従う(リリース PR 専用の書き方の規定はない)

   ```
   gh pr create --base main --head develop
   ```

2. `ci` の全ジョブが成功することを確かめる。必須ステータスチェックは未設定なので、失敗していてもマージできてしまう。**目で確かめる**

   ```
   gh pr checks <PR 番号> --watch
   ```

3. レビューを受ける。事前確認の 2・3 で見た差分を伝える。承認数は 0 の設定なので、Approve がなくてもマージはできるが、01 のフローはレビューを受けることにしている
4. **マージコミット**でマージする。`--delete-branch` は付けない(`develop` を消さない)

   ```
   gh pr merge <PR 番号> --merge
   ```

5. `main` への push で起動した `deploy-to-prod` を見る

   ```
   gh run list --workflow deploy-to-prod.yml --branch main --limit 1
   gh run watch <ID>
   ```

   ジョブは、`qa`(3 つ)→ `infra`(`terraform (plan / apply)`)→ `deploy`(`deploy`)の順に動く。前のジョブが失敗すると、後ろは動かない
6. Required reviewers が設定されていれば、`infra` と `deploy` は、それぞれ承認待ちになる([../engineering/02-infra.md](../engineering/02-infra.md) の「CD」)。承認者が、Actions の実行の画面から承認する。承認の画面では `plan` の内容が見えない。承認の前に、`qa` が成功していることと、事前確認の 3 を済ませたことを確かめる。迷ったときは、承認せずキャンセルして、[02-recovery.md](02-recovery.md) の「防ぐ仕組み」に従う
7. 「確認」を行う
8. 「タグとリリースノート」を行う

## 確認

`deploy-to-prod` の完了後に、次を確かめる。

1. すべてのジョブが成功していること

   ```
   gh run view <ID>
   ```

   - `infra` ジョブのステップサマリが「削除を含む変更: なし」で、「変更 N 件」か「plan に変更がないため、apply を飛ばす」のどちらかであること
   - `deploy` ジョブのステップサマリに、デプロイしたか、飛ばしたかが書かれていること。`backend/` に差分がないリリースでは、飛ばされるのが正常([01-operations.md](01-operations.md) の「差分がないときの動き」)

2. Cloud Run Job のイメージのタグ(`:` の後ろ)を読む

   ```
   gcloud run jobs describe <JOB_NAME> --project=<PROJECT_ID> --region=<REGION> --format='value(spec.template.spec.template.spec.containers[0].image)'
   ```

   - `backend/` に差分があったリリースでは、マージコミットの SHA(`gh pr view <PR 番号> --json mergeCommit --jq .mergeCommit.oid`)と一致する
   - 差分がなかったリリースでは、前回デプロイした SHA のまま

3. Scheduler の状態と、Job の実行を読む

   ```
   gcloud scheduler jobs list --project=<PROJECT_ID> --location=<REGION>
   gcloud run jobs executions list --job=<JOB_NAME> --project=<PROJECT_ID> --region=<REGION> --limit=5
   ```

   - Scheduler の状態が、意図どおり(`ENABLED` か `PAUSED`)であること
   - `ENABLED` のときは、実行が毎分入り、直近の実行が失敗していないこと。`PAUSED` のときは、実行は増えない

## タグとリリースノート

prod へのデプロイが成功したあとに行う。タグの形式(`vMAJOR.MINOR.PATCH`)と付ける対象(`main` のマージコミット)は、[01-operations.md](01-operations.md) の「リリースとバージョニング」に従う。

```
gh release create vX.Y.Z --target <マージコミットの SHA> --title vX.Y.Z --generate-notes
git fetch --tags origin
```

- マージコミットの SHA は、「確認」の 2 の `gh pr view` で読む
- タグが存在しなければ、リリースの作成と同時に、`--target` のコミットに作られる
- `--generate-notes` が、PR の一覧を生成する。変更点の補足は、`--notes` で、生成された内容の前に足せる。公開したあとの修正は `gh release edit vX.Y.Z --notes-file <ファイル>`
- 確認: `gh release view vX.Y.Z`

## 失敗したとき

| 症状 | 状態 | 対処 |
| --- | --- | --- |
| リリース PR の `ci` が失敗する | `main` は変わらない | `develop` を直す(作業ブランチ → `develop` の PR)。リリース PR の head は `develop` なので、PR に反映され、`ci` が再実行される |
| `deploy-to-prod` の `qa` が失敗する | `infra` と `deploy` は動かず、prod は変わらない。ただし `main` は更新済み | 一時的な失敗なら `gh run rerun <ID> --failed`。コードの問題なら、直して再度リリースするか、急ぐなら hotfix で出す |
| `infra` が「削除を含む変更」で止まる | 削除のガードが、`apply` の前に止めた。prod は変わらない | ステップサマリの一覧を読む。意図したものと確かめられるまで、`allow_destroy` を `true` にしない。意図した削除は、[01-operations.md](01-operations.md) の「インフラ(Terraform)の変更」の手順で適用する。意図しないなら、変数の空白などの原因を調べる([02-recovery.md](02-recovery.md)) |
| 承認のあとに、想定外の削除や作り直しが起きた | `apply` 済み | [02-recovery.md](02-recovery.md) の手順に従う。復旧するのはオーナー |
| `deploy` が失敗する | `infra` は成功済み | `gh run view <ID> --log-failed` で原因を読む。直ったら `gh run rerun <ID> --failed`。再実行は、元の実行と同じコミットを使うので、`main` がその後に進んでいるときは、`gh workflow run deploy-to-prod.yml --ref main` で手動実行する |

- prod を前の状態に戻す専用の手順は、決まっていない。問題の変更を取り消す修正を、通常のリリースか、hotfix で出す
- 復旧のあとの再実行は、[02-recovery.md](02-recovery.md) の「デプロイを再実行する」に従う

## hotfix

prod の緊急修正の手順。フローの定義は [01-operations.md](01-operations.md) の「hotfix フロー(フェーズ 2)」。

1. `main` から `hotfix/*` を切る。`--no-track` で、`main` を push 先にしない

   ```
   git fetch origin
   git switch -c hotfix/<Issue 番号>-<短い説明> --no-track origin/main
   ```

2. 修正し、`make verify` を通して、コミットする
3. 作業ブランチを push し、`main` 宛ての PR を作る。`ci` を確かめて、マージコミットでマージする

   ```
   git push -u origin HEAD
   gh pr create --base main
   gh pr checks <PR 番号> --watch
   gh pr merge <PR 番号> --merge
   ```

4. 「手順(リリース)」の 5〜7(`deploy-to-prod` を見る、承認、確認)を行う
5. 「タグとリリースノート」を行う。PATCH を上げる
6. 同じ修正を `develop` へバックマージする(`main` → `develop` の PR。スカッシュ)

   ```
   gh pr create --base develop --head main
   gh pr merge <PR 番号> --squash
   ```

   `develop` への push で `deploy-to-dev` が起動する。内容が変わらないときは、デプロイが飛ばされる([01-operations.md](01-operations.md) の「差分がないときの動き」)
