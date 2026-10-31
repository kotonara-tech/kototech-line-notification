---
name: issue-professional
description: Issue の建て方(タイトル・本文の構成・ラベル・マイルストーン・担当者・ボードの項目・親 Issue と子 Issue・起票の分け方)と、`gh` コマンドの注意点。Issue を作る・直す・ボード(GitHub Projects)に載せるとき、マイルストーンや Priority・Estimate・Target date を設定するとき、サブ Issue を付けるときは、ユーザーが「Issue」と明示しなくても必ず使うこと。`gh issue` / `gh project` / GraphQL の `setIssueFieldValue` に触れる作業も対象。
---

# issue-professional

Issue は「あとで読む人(と Claude)が、何を・なぜ・どこまでやるかを、迷わず判断できる」ために建てる。これまで会話の中で決めていた建て方を、ここにまとめる。

ルールの本体は `docs/operations/01-operations.md` の「Issue・PR・コミットの書き方」(タイトルのプレフィックスの表、1 Issue 1 目的、PR との関連付け)。このスキルは重複して書かず、**起票の順番、本文の構成、マイルストーン、ボードの項目、コマンドの注意点**を持つ。PR の作り方は対象外。

## 起票の流れ

1. 本文を決める(下の「本文の構成」)。1 つの Issue は 1 つの目的に絞る
2. `gh issue create` で作る。タイトル・ラベル・マイルストーン・担当者は、引数で一度に付ける
3. ボードに載るのを待つ(Auto-add で載る)。Estimate を入れる
4. Priority と Target date を入れる(GraphQL。「コマンドの注意点」)
5. Issue 側から読み直して、揃っているか確認する

```
gh issue create \
  --title "【DOCS】…" \
  --body-file <本文のファイル> \
  --label DOCS \
  --milestone v1.0.1 \
  --assignee ShuMasui
```

## タイトルとラベル

- タイトルは、領域のプレフィックス(`【INFRA】` `【BE】` `【DOCS】` `【CORE】`)で始める。対象の表は `docs/operations/01-operations.md`。**人の名前は入れない**
- ラベルは、プレフィックスと同じ 4 種類(`INFRA` / `BE` / `DOCS` / `CORE`)から **1 つ**付ける

## 本文の構成

テンプレート(`.github/ISSUE_TEMPLATE/task.md`)は「背景」と「具体的にやること」の 2 項目。次の 4 項目で書く。

| 項目 | 書くこと |
| --- | --- |
| 背景 | なぜやるのか。困っていること、きっかけ、現状 |
| 具体的にやること | チェックボックスで、1 項目ごとに、**終わったかを判断できる粒度**で書く |
| 今回は対象外 | 今回やらないこと。別の Issue にすることは、番号を添える |
| 完了条件 | 満たしたら閉じてよい条件 |

```
## 背景

…

## 具体的にやること

- [ ] 終わったかを判断できる 1 項目

### 今回は対象外

- 別の Issue にすること(Issue の番号)

## 完了条件

- …
```

## マイルストーン

- 名前は `v1.<半年ごと>.<2週ごと>`。2026-11-01 に開始し、最初は `v1.0.1`
- **期限は、マイルストーン自体**(`due_on`)に設定する。**開始日は、説明欄**に「開始 / 期限」で書く(例: `開始 2026-11-01 / 期限 2026-11-15`)
- クローズ済みの過去の Issue は、紐づけない
- 一覧(番号・期限・説明): `gh api 'repos/{owner}/{repo}/milestones?state=all' --jq '.[] | {number, title, due_on, description}'`
- 改名で番号をずらすときは、**一番後ろのマイルストーンから順に**行う(同じタイトルが並ぶ途中の状態を作らない)。改名は `gh api repos/{owner}/{repo}/milestones/<number> -X PATCH -f title=<新しい名前>`

## 担当者(Assignee)

当面は `ShuMasui`(`--assignee ShuMasui`)。

## ボードの項目

| 項目 | 種類 | 入れる値 |
| --- | --- | --- |
| Priority | Issue フィールド(単一選択) | `Urgent` / `High` / `Medium` / `Low` |
| Target date | Issue フィールド(日付) | マイルストーンの期限 |
| Estimate | ボードの項目(数値) | 人日相当のポイント |
| Start date | Issue フィールド(日付) | **個別には入れない**(マイルストーンの説明欄に書く) |
| Size | — | **使わない** |
| Status | ボードの項目(単一選択) | 選択肢は `gh project field-list` の `options` で見る |

## 親 Issue と子 Issue

- GitHub の**サブ Issue**を使う。**親 Issue は必須ではない**(親も子も持たない単独の Issue でよい)
- 子 Issue を持つ親 Issue は、リリースゲート(Issue #20)の対象外にする。親 Issue の完了は、子 Issue の完了で決まるため
- 子として作る: `gh issue create --parent <親の番号>`。あとから付け外し: `gh issue edit <番号> --parent <親の番号>` / `--remove-parent`(親側からは `--add-sub-issue` / `--remove-sub-issue`)

## 起票の分け方

- **workflow(`.github/workflows/`)の変更を含むものは、実装の前に、まず Issue に起票する**
- 1 つの Issue に 2 つ以上の目的が入ったら、分ける。今回やらないことは「今回は対象外」に書き、別の Issue にする

## コマンドの注意点

これまでに詰まった点。ID(`PVT_…` `IFSS_…` など)は、コマンドで得る。値を、スキルや文書に書き写さない(変わりうる)。

### 値の ID を得る

```
# リポジトリのオーナー
gh repo view --json owner -q .owner.login

# プロジェクトの番号と ID
gh project list --owner <owner>
gh project view <番号> --owner <owner> --format json -q .id

# ボードの項目(Estimate、Status)の ID と、Status の選択肢
gh project field-list <番号> --owner <owner> --format json

# Issue フィールド(Priority、Target date)の ID と、Priority の選択肢の ID
gh api graphql -F owner='{owner}' -f query='
query($owner: String!) {
  organization(login: $owner) {
    issueFields(first: 20) {
      nodes {
        __typename
        ... on IssueFieldSingleSelect { id name options { id name } }
        ... on IssueFieldDate { id name }
      }
    }
  }
}'

# Issue の ID と、ボードの項目 ID と、設定済みの Issue フィールド
gh api graphql -F owner='{owner}' -F name='{repo}' -F number=<Issue の番号> -f query='
query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    issue(number: $number) {
      id
      projectItems(first: 5) { nodes { id project { number } } }
      issueFieldValues(first: 10) {
        nodes {
          __typename
          ... on IssueFieldSingleSelectValue { name field { ... on IssueFieldSingleSelect { name } } }
          ... on IssueFieldDateValue { value field { ... on IssueFieldDate { name } } }
        }
      }
    }
  }
}'
```

### Priority・Target date(Issue フィールド)

- Priority・Start date・Target date は **Issue フィールド**で、ボードの項目(Estimate、Status)とは別。`gh project item-edit` ではなく、GraphQL の `setIssueFieldValue` で設定する
- `gh project field-list` は、Priority を単一選択として出すが、選択肢(`options`)を出さない。`item-edit` に渡す選択肢の ID を、ここからは得られない
- `issueFields` は配列。Priority と Target date を、1 回の呼び出しでまとめて渡せる

```
gh api graphql -f query='
mutation($issue: ID!, $priorityField: ID!, $option: ID!, $targetField: ID!, $date: String!) {
  setIssueFieldValue(input: {
    issueId: $issue
    issueFields: [
      { fieldId: $priorityField, singleSelectOptionId: $option }
      { fieldId: $targetField, dateValue: $date }
    ]
  }) { issue { number } }
}' -f issue=<Issue の ID> -f priorityField=<Priority の ID> -f option=<選択肢の ID> \
  -f targetField=<Target date の ID> -f date=<YYYY-MM-DD>
```

### Estimate・Status(ボードの項目)

- `gh project item-edit` で更新する。**1 回の呼び出しで更新できる値は 1 つ**
- Issue を作った直後は、ボードに載るまで数秒かかる。ボードの項目 ID(`projectItems` の `id`)が出るまで待ってから、更新する

```
gh project item-edit --id <ボードの項目 ID> --project-id <プロジェクトの ID> \
  --field-id <Estimate の ID> --number <ポイント>
gh project item-edit --id <ボードの項目 ID> --project-id <プロジェクトの ID> \
  --field-id <Status の ID> --single-select-option-id <選択肢の ID>
```

### 確認

- **`gh project item-list` は、Issue フィールド(Priority、日付)を表示しない**。確認は、Issue 側(上の GraphQL の `issueFieldValues`)で行う
- `gh project item-list` は、既定で 30 件までしか取らない。多いときは `-L` で増やす

## 書き終えたときのセルフチェック

- [ ] タイトルがプレフィックスで始まり、人の名前が入っていないか
- [ ] ラベルが、プレフィックスと同じ 1 つか
- [ ] 本文に、背景・具体的にやること・今回は対象外・完了条件があるか。やることは、終わったかを判断できる粒度か
- [ ] マイルストーン・担当者が付いているか
- [ ] Priority・Estimate・Target date が揃っているか(`item-list` ではなく、Issue 側の GraphQL で確認したか)
- [ ] workflow の変更を含むのに、実装を先にしていないか
