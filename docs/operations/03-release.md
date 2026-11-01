# 03. リリース手順

`develop` の内容を prod に出し、バージョンタグとリリースノートを残す手順書。ブランチの役割、マージ方法、バージョニングの考え方は [01-operations.md](01-operations.md) が正本で、ここには、**誰が、何を、どの順で、どのコマンドで行い、何を確認するか**だけを書く。

リリースは、`develop` → `main` の PR のマージと、そのあとの `deploy-to-prod` の手動起動で行う。`main` への push では、`deploy-to-prod` は起動しない。起動では、最新の `main` の先頭の SHA を入力する。最初のジョブ `guard` が入力を検査し、通ると、QA → インフラ(Terraform)→ アプリのデプロイの順に、prod へ反映する([../engineering/02-infra.md](../engineering/02-infra.md) の「CD」)。

**Environment `prod` の承認は待たない**(D36)。起動した実行は、止めない限り、そのまま prod へ反映される。そのため、計画の確認は、起動の**前**に済ませる(`apply-to-prod` を plan のみで先に実行する)。起動したあとに止められるのは、`apply` が始まる前まで(「起動した実行を止める」)。

**検証の状態**: この手順書は、workflow・設定・`gh` の読み取りで確かめて書いた。リリースを通して実行しての確認は、未実施。手動起動と SHA の入力(`guard`)も、実際の実行では未確認(判定のシェルは、ローカルで確かめた)。plan のみの実行(`apply-to-prod` の `-f apply=false`)と、実行のキャンセルで止める手順も、実際の実行では未確認(特に、`plan` の途中でキャンセルしたときに、state のロックが残らないか)。確かめる項目は「最初の起動で確かめること」にある。最初のリリースで確かめ、食い違いを直す。

## 前提

| 役割 | すること | 必要なもの |
| --- | --- | --- |
| 実施者 | リリース PR の作成とマージ、`apply-to-prod`(plan のみ)と `deploy-to-prod` の手動起動、起動した実行の確認と、必要なときのキャンセル、タグとリリースの作成、確認 | リポジトリの書き込み権限。`gh`(ログイン済み)と `git`。確認には、prod の GCP プロジェクトを読み取れる `gcloud` のアカウント |
| オーナー | 失敗したあとの復旧([02-recovery.md](02-recovery.md)) | 手元のオーナー権限 |

- コマンドは、このリポジトリのクローンの中で実行する(`gh api` の `{owner}/{repo}` は、`gh` が補う)
- `<PROJECT_ID>`・`<REGION>`・`<JOB_NAME>` は、Environment `prod` の変数 `GCP_PROJECT_ID`・`REGION`・`CLOUD_RUN_JOB_NAME` の値に置き換える(`gh variable list --env prod` で読める)。`<ID>` は、`gh run list` に出る実行の ID
- `main` は Ruleset で保護され、直 push はできない([01-operations.md](01-operations.md) の「ブランチ保護(Ruleset)」)
- この手順は、フェーズ 2 の運用を前提にする([01-operations.md](01-operations.md) の「運用フェーズ」)
- 秘匿情報(Secret の値、トークン)は、この手順では使わない。読まない
- **Environment `prod` は、承認者を設定せず、承認を待たない**(D36。2026-11-01 に `gh api` で確かめた: `protection_rules` が空)。`deploy-to-prod` を起動すると、`guard` と `qa` が通れば、承認なしで `infra` と `deploy` が進む。そのため、事前確認と計画の確認は、**起動の前**に済ませる。起動のあとに止められるのは、`apply` が始まる前まで(「起動した実行を止める」)。承認を再開するときは、[../engineering/02-infra.md](../engineering/02-infra.md) の「承認を再開するとき」

## 事前確認

リリース PR を作る前に、次を確かめる。

1. dev で動作を確認済みで、`deploy-to-dev` が成功していること。失敗しているときは、リリースしない

   ```
   gh run list --workflow deploy-to-dev.yml --branch develop --limit 3
   ```

2. リリースに含まれる変更が、意図したものと合っていること。`develop` のコミットは、スカッシュした PR 1 件ずつ

   ```
   git fetch origin
   git log --oneline origin/main..origin/develop
   git diff --stat origin/main origin/develop
   ```

3. `infra/` と `.github/` の差分を、一覧ではなく、中身まで読む(リリース PR を作ったあとは、Files changed で開いてもよい)

   ```
   git diff origin/main origin/develop -- infra/ .github/
   ```

   | 差分 | 見ること |
   | --- | --- |
   | `infra/` | 削除(`delete`)や作り直し(`replace`)につながる変更がないか。prod の `terraform plan` は PR 上では実行できないので、次の 4 で dev の plan を読む |
   | `.github/` | マージした `main` の workflow が、そのまま prod のデプロイに使われる。`infra/` と同じ重さで読む |

4. `infra/` に差分があるときは、その変更をマージしたときの `deploy-to-dev` の実行で、`terraform plan` を読む。**最新の実行は読まない**(適用済みなので、plan は「変更なし」になる)

   ```
   git log --oneline origin/main..origin/develop -- infra/
   gh run list --workflow deploy-to-dev.yml --commit <SHA> --json databaseId,conclusion,url
   gh run view <ID> --web
   ```

   - `<SHA>` は、1 行目に出たコミットの SHA。出たコミットごとに繰り返す
   - 開いた実行の画面の Summary で、`infra` ジョブの `terraform plan` を読む。削除(`delete`)や作り直し(`replace`)がないことを確かめる
   - 実行が見つからない、または失敗しているときは、リリースしない。dev に適用されていない変更を、prod へ出さない
   - dev と prod は state が違うので、同じ計画になるとは限らない。削除を含む計画は、prod では `infra` ジョブで止まる(「失敗したとき」)

5. prod の変数と、Environment の設定を読む

   ```
   gh variable list --env prod
   gh api repos/{owner}/{repo}/environments/prod --jq '.protection_rules'
   ```

   - 変数の値の前後に、空白がないこと(v1.0.0 では `GCP_PROJECT_ID` の先頭の空白で、prod のリソースが作り直された)。デプロイの最初に `.github/actions/check-vars` も検査して失敗させるが、手元で先に見る
   - `SCHEDULER_PAUSED` が、意図した値であること。`terraform apply` が、この値(未設定は `true`)を Scheduler の一時停止の状態に設定する。リリースの手順に、Scheduler の再開は含まれない([../engineering/02-infra.md](../engineering/02-infra.md) の「Cloud Scheduler」)
   - `protection_rules` が空であること(承認を待たない設定。D36)。承認者(`required_reviewers`)が入っていると、`infra` と `deploy` が承認待ちで止まる。承認を再開したのでなければ、設定が変わっているので、オーナーに確かめる

## 手順(リリース)

1. リリース PR を作る。タイトルと本文の書き方は未決(Q21)。[01-operations.md](01-operations.md) の「Issue・PR・コミットの書き方」は、1 つの Issue と関連付ける PR を前提にしていて、複数の変更を含むリリース PR には、そのまま当てはまらない

   ```
   gh pr create --base main --head develop --title "<タイトル>" --body "<本文>"
   ```

2. `ci` の全ジョブが成功することを確かめる。必須ステータスチェックが設定されるまでは、失敗していてもマージできてしまうので、**目で確かめる**([01-operations.md](01-operations.md) の「運用フェーズ」の「現在の状態」)

   ```
   gh pr checks <PR 番号> --watch
   ```

3. レビューを受ける。事前確認の 3・4 で見た差分を伝える。他の人の Approve の必須化が設定されるまでは、Approve がなくてもマージはできる。ただし、[01-operations.md](01-operations.md) のリリースフローは、レビューを受けることにしている
4. **マージコミット**でマージする。`--delete-branch` は付けない(`develop` を消さない)

   ```
   gh pr merge <PR 番号> --merge
   ```

5. 起動で入力する SHA を取る(マージのあとに行う)

   ```
   git fetch origin
   git rev-parse origin/main
   gh pr view <PR 番号> --json mergeCommit --jq .mergeCommit.oid
   ```

   - 1 つ目が、いまの `main` の先頭の SHA(40 桁の 16 進)で、入力するのはこれ。2 つ目は、リリース PR のマージコミットの SHA
   - 2 つが一致することを、目で確かめる。一致しないときは、マージのあとに別の変更が `main` に入っている。`git log --oneline <マージコミットの SHA>..origin/main` で何が入ったかを読んでから、進む
6. `apply-to-prod` を、`apply` を false(plan のみ)にして起動し、計画を読む。`deploy-to-prod` を起動する**前**に行う

   ```
   gh workflow run apply-to-prod.yml --ref main -f sha=<5 で取った SHA> -f apply=false
   gh run list --workflow apply-to-prod.yml --event workflow_dispatch --branch main --limit 5 --json databaseId,headSha,createdAt,status
   gh run watch <ID>
   gh run view <ID> --web
   ```

   - `<ID>` の選び方は、手順 8 と同じ(`headSha` が手順 5 の SHA と一致する実行のうち、`createdAt` が最も新しいもの)
   - この実行は prod を変えない(`apply: false` のため、`apply` のステップは動かない)。`allow_destroy` は渡さない
   - 開いた実行の画面の Summary で、`infra` ジョブ(`terraform (plan)`)のステップサマリを読む
     - 「削除を含む変更」が「なし」であること。**plan のみの実行では、削除があっても失敗にならず、警告とサマリの一覧になる**(`apply` しないため)。必ず読む
     - 「terraform plan (prod)」の中身に、意図しない作成・更新がないこと。「plan に変更がないため、apply を飛ばす」が出るときは、インフラの変更がない
   - 削除を含む変更が出たとき、または意図しない変更があるときは、手順 7 に進まない。原因を調べる(「失敗したとき」)。変数の空白などの原因は、[02-recovery.md](02-recovery.md) の「防ぐ仕組み」
   - 本番の実行(手順 7)は `plan` をやり直す。手順 6 から 7 までのあいだに、GitHub の変数を変えない。別の変更が `main` に入ると、取った SHA が古くなり、手順 7 の `guard` が失敗する。そのときは、5 から取り直して、6 からやり直す
7. `deploy-to-prod` を手動で起動する。**承認は待たない**(D36)ので、`guard` と `qa` が通ると、`infra` が `plan` のあと、そのまま `apply` に進む(削除を含む計画だけは、削除のガードが止める)

   ```
   gh workflow run deploy-to-prod.yml --ref main -f sha=<5 で取った SHA>
   ```

   - `--ref main` を必ず付ける。省略すると、既定ブランチ(`develop`)で動こうとして、`guard` が失敗する
   - 最初のジョブ `guard` が、起動した ref・SHA の形式・入力が最新の `main` の先頭と一致するか・起動したコミットが同じ先頭と一致するかを確かめる。1 つでも満たさないと失敗して、以降は動かない(「失敗したとき」)
   - 手順 5 から起動までのあいだに別の変更が `main` に入ると、取った SHA が古くなり、`guard` が失敗する。そのときは、5 から取り直して、起動し直す
   - 起動したあとに止めたくなったら、すぐに「起動した実行を止める」に従う
8. 起動した `deploy-to-prod` を見る

   ```
   gh run list --workflow deploy-to-prod.yml --event workflow_dispatch --branch main --limit 5 --json databaseId,headSha,createdAt,status
   gh run watch <ID>
   ```

   - `headSha` が手順 5 の SHA と一致する実行のうち、`createdAt` が最も新しいものを選び、その `databaseId` を `<ID>` にする。起動の直後は、一覧に出ないことがある。出るまで数秒おいて、実行し直す。1 つ前の実行を選ぶと、古い結果を見て成功と思い込む
   - ジョブは、`guard` → `qa`(3 つ)→ `infra`(`terraform (plan / apply)`)→ `deploy`(`deploy`)の順に動く。承認待ちはない。前のジョブが失敗すると、後ろは動かない
   - 実行を見ているあいだに `main` が進んだときは、この実行は、入力した古い SHA のコミットのまま進む(`guard` は検査し直さない)。`apply` が始まる前なら、キャンセルし、手順 5 から取り直して、起動し直す(「起動した実行を止める」)
9. 「確認」を行う
10. 「タグとリリースノート」を行う

## 起動した実行を止める

Environment `prod` は承認を待たない(D36)ので、実行を止める手段は、起動の前の確認(手順 6)と、起動したあとのキャンセルだけ。`infra` の `apply` が**始まる前**なら、キャンセルしても prod は変わらない。始まったあとは、止めない。

```
gh run cancel <ID>
```

| 実行の状態 | 止めるか | 理由 |
| --- | --- | --- |
| `guard`・`qa` | キャンセルしてよい | prod のリソースを変えるステップが、まだ動いていない |
| `infra` の、`apply` より前のステップ(`check-vars`・リリース元の検査・認証・`init`・`validate`・`plan`・変更の有無の判定・サマリ・削除のガード) | キャンセルしてよい | `apply` が始まっておらず、prod は変わらない。`plan` の途中でキャンセルしたとき、state のロックが残らないかは未確認 |
| `infra` の `apply` が実行中 | **キャンセルしない**。終わるのを待つ | `terraform apply` を途中で止めると、一部のリソースだけが変わった状態や、state のロックが残った状態になり、state が壊れるおそれがある。ワークフローも、`apply` を途中で中断させない作りにしている(`reusable-infra-deploy.yml` の `concurrency`、`cancel-in-progress: false`) |
| `infra` が成功し、`deploy` が実行中 | キャンセルできるが、インフラだけが新しい状態になる | `deploy` は Terraform を動かさないので、state は壊れない。`infra` で適用された変更は戻らず、Job のイメージは古いまま(更新の途中でないかは、「確認」の 2 で読む)。直すときは、SHA を取り直して(手順 5)、`deploy-to-prod` を起動し直す(`infra` は、変更がなければ `apply` を飛ばす) |

`apply` が始まったかは、キャンセルの**直前**に読む。

```
gh run view <ID> --json jobs --jq '.jobs[] | select(.name | startswith("infra")) | {name, status, apply: [.steps[] | select(.name == "apply") | {status, conclusion}]}'
```

- `apply` の `status` が `in_progress` のときは、キャンセルしない。`completed` のときは、`conclusion` が `success` なら適用済み、`skipped` なら適用していない(変更がない、または plan のみ)。それ以外で、まだ始まっていないときは、キャンセルしてよい(始まる前の `status` の値は、実際の実行では未確認)
- ジョブ名が `terraform (plan)` の実行は plan のみで、`apply` のステップは飛ばされる。prod は変わらない
- 読んでからキャンセルするまでのあいだに、`apply` が始まることがある。`apply` の実行中にキャンセルが届いたときは、「失敗したとき」の対処に従う
- キャンセルしたあとは、原因を直し、SHA を取り直して(手順 5)、手順 6 からやり直す

## 確認

`deploy-to-prod` の完了後に、次を確かめる。

1. すべてのジョブが成功していること

   ```
   gh run view <ID>
   ```

   `gh run view <ID>` で読めるのは、ジョブの成否まで。ステップサマリは出ない。サマリは、Actions の実行の画面(Summary)で読む(`gh run view <ID> --web` で開く)

   - `infra` ジョブのサマリが「削除を含む変更: なし」で、「変更 N 件」か「plan に変更がないため、apply を飛ばす」のどちらかであること
   - `deploy` ジョブのサマリに、デプロイしたか、飛ばしたかが書かれていること。`backend/` に差分がないリリースでは、飛ばされるのが正常([01-operations.md](01-operations.md) の「差分がないときの動き」)
   - ログ(`gh run view <ID> --log`)でも、「削除を含む変更: なし」と「plan に変更がないため、apply を飛ばす」は読める(スクリプトの行も一緒に出る)。「変更 N 件」は、サマリでしか読めない

2. Cloud Run Job のイメージのタグ(`:` の後ろ)を読む

   ```
   gcloud run jobs describe <JOB_NAME> --project=<PROJECT_ID> --region=<REGION> --format='value(spec.template.spec.template.spec.containers[0].image)'
   ```

   - `backend/` に差分があったリリースでは、起動で入力した SHA と一致する(手順 5 で 2 つの SHA が一致していれば、マージコミットの SHA とも同じ)
   - 差分がなかったリリースでは、前回デプロイした SHA のまま

3. Scheduler の状態と、Job の実行を読む

   ```
   gcloud scheduler jobs list --project=<PROJECT_ID> --location=<REGION>
   gcloud run jobs executions list --job=<JOB_NAME> --project=<PROJECT_ID> --region=<REGION> --limit=5
   ```

   - Scheduler の状態が、意図どおり(`ENABLED` か `PAUSED`)であること
   - `ENABLED` のときは、実行が 5 分ごとに入り、直近の実行が失敗していないこと。`PAUSED` のときは、実行は増えない

この確認は、GCP 側の状態までで、prod の LINE に通知が届くことの確認は含まない。その手順は未整備(Q24)。

## タグとリリースノート

prod へのデプロイが成功したあとに行う。タグの形式(`vMAJOR.MINOR.PATCH`)と付ける対象(`main` のマージコミット)は、[01-operations.md](01-operations.md) の「リリースとバージョニング」に従う。`X.Y.Z` を決める基準(何の変更で、どれを上げるか)は未決(Q22)。

```
gh release create vX.Y.Z --target <マージコミットの SHA> --title vX.Y.Z --generate-notes
git fetch --tags origin
```

- マージコミットの SHA は、「手順(リリース)」の 5 の `gh pr view` で読む
- タグが存在しなければ、リリースの作成と同時に、`--target` のコミットに作られる
- `--generate-notes` が、PR の一覧を生成する。変更点の補足は、`--notes` で、生成された内容の前に足せる。公開したあとの修正は `gh release edit vX.Y.Z --notes-file <ファイル>`
- 確認: `gh release view vX.Y.Z`

## 失敗したとき

| 症状 | 状態 | 対処 |
| --- | --- | --- |
| リリース PR の `ci` が失敗する | `main` は変わらない | `develop` を直す(作業ブランチ → `develop` の PR)。リリース PR の head は `develop` なので、PR に反映され、`ci` が再実行される |
| `deploy-to-prod` の `guard` が失敗する | `qa` 以降は動かず、prod は変わらない | 失敗のメッセージ(Actions の実行の画面のアノテーション。`gh run view <ID> --log-failed`)を読む。起動した ref が `main` でない、SHA が 40 桁の 16 進でない、SHA が最新の `main` の先頭と違う、起動したコミットが最新の `main` の先頭と違う、先頭を取得できない、のいずれか。手順 5 から SHA を取り直して、起動し直す。取得できないときは、時間をおいて起動し直す |
| `deploy-to-prod` の `qa` が失敗する | `infra` と `deploy` は動かず、prod は変わらない。ただし `main` は更新済み | 一時的な失敗で、`main` が進んでいないなら、`gh run rerun <ID> --failed`(下の説明)。コードの問題なら、直して再度リリースするか、急ぐなら hotfix で出す |
| `infra` が「削除を含む変更」で止まる(意図しない削除) | 削除のガードが、`apply` の前に止めた。prod は変わらない。`deploy` は動いていない | ステップサマリの一覧を読む。意図したものと確かめられるまで、`allow_destroy` を `true` にしない。変数の空白などの原因を調べる([02-recovery.md](02-recovery.md)) |
| `infra` が「削除を含む変更」で止まる(意図した削除) | 同上 | [01-operations.md](01-operations.md) の「インフラ(Terraform)の変更」の手順で適用する。適用したら、SHA を取り直して(手順 5)、`gh workflow run deploy-to-prod.yml --ref main -f sha=<最新の main の SHA>` で再度起動する(`infra` は変更なしで飛ばされ、`deploy` が動く)。そのあと、「確認」と「タグとリリースノート」を行う |
| `apply` のあとに、想定外の削除や作り直しが起きた | `apply` 済み | [02-recovery.md](02-recovery.md) の手順に従う。復旧するのはオーナー |
| キャンセルが、`infra` の `apply` の実行中に届いた | `terraform apply` が途中で止まり、一部のリソースだけが変わった、または state のロックが残った可能性がある | オーナーが、手元で `terraform plan` を読み、state と実体の差を確かめる([02-recovery.md](02-recovery.md) の「前提」)。ロックが残っていれば、他の `apply` が動いていないことを確かめてから解除する(`terraform force-unlock`。実行での確認は未実施)。確かめるまで、`deploy-to-prod` も `apply-to-prod` も起動しない |
| `deploy` が失敗する | `infra` は成功済み | `gh run view <ID> --log-failed` で原因を読む。直ったら、`main` が進んでいないことを確かめて、`gh run rerun <ID> --failed`(下の説明)。進んでいるときは、新しい SHA で起動し直す |

- 失敗したジョブだけの再実行(`gh run rerun <ID> --failed`)は、元の実行と同じコミット(`github.sha`)で動き、成功済みの `guard` は再実行されない。そのため、`main` が進んでいても検査されず、古い SHA のコミットが prod に出る。再実行の前に、`git fetch origin` のあと `git rev-parse origin/main` が、元の実行で入力した SHA と同じであることを確かめる。違うときは、再実行せず、手順 5 から取り直して起動し直す(新しいコミットを出した実行のあとに、古いコミットの内容を適用して、巻き戻すのを避けるため)
- 全ジョブの再実行(`gh run rerun <ID>`)は、`guard` も再実行される。入力 `sha` の引き継ぎは未確認([../engineering/02-infra.md](../engineering/02-infra.md) の「CD」)。`guard` が失敗したら、新しい SHA で起動し直す
- prod を前の状態に戻す専用の手順は、決まっていない(Q23)。問題の変更を取り消す修正を、通常のリリースか、hotfix で出す
- 復旧のあとの再実行は、[02-recovery.md](02-recovery.md) の「デプロイを再実行する」に従う

## 最初の起動で確かめること

`guard` は、GitHub Actions 上の実行では確かめていない(判定のシェルだけ、ローカルで確かめた)。最初のリリースで、次を確かめる。食い違いがあれば、この手順書と [../engineering/02-infra.md](../engineering/02-infra.md) の「CD」を直す。

1. 古い SHA で、`guard` が失敗すること(prod は変わらない)。リリース PR のマージのあと、手順 5 の前に、マージ前の `main` の先頭を入れて起動する

   ```
   git fetch origin
   gh workflow run deploy-to-prod.yml --ref main -f sha=$(git rev-parse origin/main^1)
   ```

   `origin/main^1` は、マージコミットの第 1 親(マージ前の `main` の先頭)。

   - `guard` が「入力 sha ... 一致しない」で失敗し、`qa` 以降が「skipped」になること
   - 万一 `guard` が通ってしまったら、すぐに実行をキャンセルする(承認を待たないので、`qa` の成功のあとに `infra` が進む。「起動した実行を止める」。`apply` が始まる前に止める)
   - この失敗した実行を、全ジョブで再実行(`gh run rerun <ID>`)すると、入力 `sha` が引き継がれるかが分かる。引き継がれれば、同じ「一致しない」で失敗する。引き継がれなければ、「40 桁の 16 進ではない」で失敗する(どちらも prod は変わらない)
2. `main` 以外の ref で、`guard` が失敗すること。この変更が `develop` に入ったあとなら、リリースの前に試せる

   ```
   git fetch origin
   gh workflow run deploy-to-prod.yml --ref develop -f sha=$(git rev-parse origin/develop)
   ```

   - `guard` が「起動した ref が refs/heads/main ではない」で失敗し、`qa` 以降が「skipped」になること
3. 正しい SHA の実際の起動(手順 6 の plan のみの実行と、手順 7)で、`guard` が成功し、`qa` 以降が動くこと
   - 手順 6 の実行で、`-f apply=false` が効いて、`infra` ジョブが `terraform (plan)` の名前で動き、`apply` のステップが飛ばされること。サマリに「削除を含む変更」と「terraform plan (prod)」が出ること
   - `guard` のステップサマリに、「prod の起動の検査: <SHA> は、いまの main の先頭と一致した」と出ること(`github.token` で `main` の先頭を取得できている)
   - `infra` と `deploy` の `actions/checkout` が取得したコミットが、入力した SHA と同じであること(実行のログで読む)。同じでなければ、`guard` が確かめた SHA と、デプロイされるコミットが違う

## hotfix

prod の緊急修正の手順。フローの定義は [01-operations.md](01-operations.md) の「hotfix フロー(フェーズ 2)」。

1. `main` から `hotfix/*` を切る。`--no-track` で、`main` を push 先にしない

   ```
   git fetch origin
   git switch -c hotfix/<Issue 番号>-<短い説明> --no-track origin/main
   ```

2. 修正し、`make verify` を通して、コミットする
3. 作業ブランチを push し、`main` 宛ての PR を作る

   ```
   git push -u origin HEAD
   gh pr create --base main
   ```

4. `ci` を確かめ、レビューを受ける。Approve の必須化が設定されるまでは、Approve がなくてもマージはできる。ただし、[01-operations.md](01-operations.md) の hotfix フローは、`ci` と承認を経てマージすることにしている

   ```
   gh pr checks <PR 番号> --watch
   ```

5. マージコミットでマージする

   ```
   gh pr merge <PR 番号> --merge
   ```

6. 「手順(リリース)」の 5〜9(SHA を取る、`apply-to-prod` の plan のみの実行で計画を読む、`deploy-to-prod` の起動、実行を見る、確認)を行う。止めたいときは「起動した実行を止める」
7. 「タグとリリースノート」を行う。PATCH を上げる
8. 同じ修正を `develop` へバックマージする(`main` → `develop` の PR。スカッシュ)

   ```
   gh pr create --base develop --head main
   gh pr merge <PR 番号> --squash
   ```

   `develop` への push で `deploy-to-dev` が起動する。内容が変わらないときは、デプロイが飛ばされる([01-operations.md](01-operations.md) の「差分がないときの動き」)
