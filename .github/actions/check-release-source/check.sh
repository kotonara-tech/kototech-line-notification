#!/usr/bin/env bash
# main に入る(入った)コミットが、develop で検証済みのものかを検査する。action.yml から呼ぶ。
# 判定の中身と場面の表は、docs/operations/01-operations.md の「リリース元の検査」にある。
#
# 環境変数(action.yml が渡す。値はシェルのコマンドに埋め込まず、変数として読む):
#   MODE      pull_request(main 宛ての PR の検査)か、deploy(prod のデプロイの検査)
#   SHA       検査するコミット(pull_request は PR の head、deploy はデプロイするコミット)。40 桁の小文字の 16 進数
#   HEAD_REF  pull_request のみ。PR の head のブランチ名。同じリポジトリのブランチのときだけ渡す(fork や不明なときは空)
#   REPO      owner/repo。空なら、ランナーが設定する GITHUB_REPOSITORY を使う
#   GH_TOKEN  gh が使うトークン(contents: read で足りる)
#
# 取得(gh api)に失敗したとき、値が読めないとき、想定外の値のときは、通さずに失敗する(fail-closed)。
# 使う API は branches / commits / compare で、どれも contents の read で読める。
set -euo pipefail

DEVELOP_BRANCH=develop
MODE="${MODE:-}"
SHA="${SHA:-}"
HEAD_REF="${HEAD_REF:-}"
REPO="${REPO:-${GITHUB_REPOSITORY:-}}"
# hotfix の切り元を探すとき、第 2 親から第 1 親の鎖を、最大この数だけたどる(超えたら失敗する)
WALK_LIMIT=30

say() {
  echo "$1"
  printf '%s\n' "$1" >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
}

pass() {
  say "リリース元の検査: 通った。$1"
  exit 0
}

warn() {
  echo "::warning title=リリース元の検査::$1"
  printf '%s\n' "注意: $1" >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
}

# メッセージには、固定の文と、検証済みの SHA だけを入れる(ブランチ名など、人が書ける値は入れない)
die() {
  echo "::error title=リリース元の検査に失敗::$1"
  printf '### リリース元の検査: 失敗\n\n%s\n' "$1" >> "${GITHUB_STEP_SUMMARY:-/dev/null}"
  exit 1
}

is_sha() {
  [[ "$1" =~ ^[0-9a-f]{40}$ ]]
}

fetch_develop_tip() {
  DEVELOP_TIP="$(gh api "repos/${REPO}/branches/${DEVELOP_BRANCH}" --jq '.commit.sha')" \
    || die "${DEVELOP_BRANCH} の先頭を取得できない(gh api が失敗した)。通さずに止めた。一時的な失敗なら、再実行する"
  is_sha "$DEVELOP_TIP" || die "${DEVELOP_BRANCH} の先頭の SHA を読めない。通さずに止めた"
}

# $2 のコミットが、$1 の履歴に含まれる($1 と同一か、$1 の祖先)なら 0、含まれないなら 1 を返す。
# 取得に失敗したときと、想定外の値のときは、返さずに失敗する。
# compare は、base=$1、head=$2 で呼ぶ。status は head から見た関係で、
# identical(同一)と behind(head が base の祖先)が「含まれる」、ahead と diverged が「含まれない」
in_history() {
  local status
  if [ "$1" = "$2" ]; then
    return 0
  fi
  status="$(gh api "repos/${REPO}/compare/${1}...${2}" --jq '.status')" \
    || die "${2:0:7} が ${1:0:7} の履歴に含まれるかを取得できない(gh api が失敗した)。通さずに止めた。一時的な失敗なら、再実行する"
  case "$status" in
    identical | behind) return 0 ;;
    ahead | diverged) return 1 ;;
    *) die "${2:0:7} と ${1:0:7} の関係(status)を読めない。通さずに止めた" ;;
  esac
}

# $1 のコミットが、develop の履歴に含まれるか
in_develop() {
  in_history "$DEVELOP_TIP" "$1"
}

# $1 のコミットの親を、PARENT_COUNT / FIRST_PARENT / SECOND_PARENT に入れる
fetch_parents() {
  local out p
  local -a parts
  out="$(gh api "repos/${REPO}/commits/${1}" --jq '[.parents[].sha] | join(" ")')" \
    || die "${1:0:7} の親を取得できない(gh api が失敗した)。通さずに止めた。一時的な失敗なら、再実行する"
  read -r -a parts <<< "$out" || true
  PARENT_COUNT="${#parts[@]}"
  FIRST_PARENT="${parts[0]:-}"
  SECOND_PARENT="${parts[1]:-}"
  for p in ${parts[@]+"${parts[@]}"}; do
    is_sha "$p" || die "${1:0:7} の親の SHA を読めない。通さずに止めた"
  done
}

# 切り元を探す。$2(マージされた側のブランチの先頭)から、第 1 親の鎖をたどり、
# $1(マージされた側ではない、それまでの main の先頭)の履歴に含まれる最初のコミットを、BRANCH_POINT に入れる。
# 第 1 親をたどるのは、ブランチの上で作ったコミットは第 1 親でつながり、main などの取り込み(Update branch など)は第 2 親になるため。
# つまり、ブランチが main を取り込んでいても、切り元は、ブランチを切ったコミットになる。
# 鎖が親のないコミットで終わるとき、WALK_LIMIT 個のコミット(第 2 親を含む)を調べても着かないとき、取得に失敗したときは、失敗する
find_branch_point() {
  local cur="$2" i
  for ((i = 0; i < WALK_LIMIT; i++)); do
    if in_history "$1" "$cur"; then
      BRANCH_POINT="$cur"
      return 0
    fi
    fetch_parents "$cur"
    if [ "$PARENT_COUNT" -lt 1 ]; then
      die "${2:0:7} の履歴をたどったが、${1:0:7} の履歴に着かなかった(親のないコミットに着いた)。通さずに止めた"
    fi
    cur="$FIRST_PARENT"
  done
  die "${2:0:7} から第 1 親の鎖を ${WALK_LIMIT} 個のコミット調べても、${1:0:7} の履歴に着かなかった(長いブランチ)。切り元を判定できないので、通さずに止めた"
}

# main 宛ての PR: PR の head の先頭が、develop の先頭と同一であること
check_pull_request() {
  # hotfix は main から切って main に出す。develop の履歴を経ないため、対象外にする
  # (同じリポジトリのブランチだけ。fork のブランチは、workflow が空にして渡す)
  if [[ "$HEAD_REF" == hotfix/?* ]]; then
    pass "PR の head が hotfix/* のため、対象外とした(hotfix は main から切り、develop の履歴を経ない)"
  fi

  fetch_develop_tip
  if [ "$SHA" = "$DEVELOP_TIP" ]; then
    pass "PR の head ${SHA:0:7} は、${DEVELOP_BRANCH} の先頭と同一"
  fi
  die "PR の head (${SHA:0:7}) が、${DEVELOP_BRANCH} の先頭 (${DEVELOP_TIP:0:7}) と同一でない。${DEVELOP_BRANCH} で検証されていないコミットを、main に入れない。リリースは ${DEVELOP_BRANCH} → main の PR で出す。${DEVELOP_BRANCH} が PR のあとに進んだときは、head を ${DEVELOP_BRANCH} の先頭に合わせる。hotfix は hotfix/* の名前にする"
}

# prod のデプロイ: デプロイするコミットが、develop で検証済みのものだけから成ること
check_deploy() {
  fetch_develop_tip
  say "デプロイするコミット: ${SHA:0:7}、${DEVELOP_BRANCH} の先頭: ${DEVELOP_TIP:0:7}"

  # 1. コミット自身が develop の履歴に含まれる(fast-forward で main を進めた場合など)
  if in_develop "$SHA"; then
    pass "デプロイするコミットは、${DEVELOP_BRANCH} の履歴に含まれる"
  fi

  # 2. リリースのマージ: 第 2 親(マージされた側)が develop の履歴に含まれる。
  #    develop がその後に進んでいても通る(第 2 親は、そのときの develop の先頭で、今は祖先)。
  #    第 1 親(それまでの main の先頭)は見ない(前のデプロイで検査済み)
  fetch_parents "$SHA"
  if [ "$PARENT_COUNT" -ne 2 ]; then
    die "デプロイするコミット (${SHA:0:7}) は、${DEVELOP_BRANCH} の履歴に含まれず、マージコミットでもない(親が ${PARENT_COUNT} 個)。main への直接のコミットや、スカッシュ・リベースのマージの可能性がある。リリースは ${DEVELOP_BRANCH} → main の PR を、マージコミットでマージする"
  fi
  if in_develop "$SECOND_PARENT"; then
    pass "マージコミットの第 2 親 ${SECOND_PARENT:0:7} が、${DEVELOP_BRANCH} の履歴に含まれる"
  fi

  # 3. hotfix のマージ: 第 2 親のブランチが、main から切られている。
  #    第 2 親から第 1 親の鎖をたどり、第 1 親(それまでの main の先頭)の履歴に含まれる最初のコミット(切り元)を探す。
  #    切り元が develop の履歴にあれば、develop から切ったブランチ。main を取り込んでいても(Update branch など)、切り元は変わらない。
  #    切り元が develop の履歴になければ、main から切ったブランチ(hotfix)とみなす。
  #    デプロイ時は、ブランチ名を確かめられない(PR の情報を読む権限がなく、マージコミットのメッセージにも名前が入らない)。
  #    そのため、名前ではなく、履歴の形で見る。故意に作った履歴(main から切ったブランチに、develop のコミットをマージするなど)は、見分けられない
  # (find_branch_point は fetch_parents を呼ぶので、FIRST_PARENT などが書き換わる。先に退避する)
  MAIN_PARENT="$FIRST_PARENT"
  BRANCH_TIP="$SECOND_PARENT"
  find_branch_point "$MAIN_PARENT" "$BRANCH_TIP"
  if in_develop "$BRANCH_POINT"; then
    die "マージコミット (${SHA:0:7}) の第 2 親 (${BRANCH_TIP:0:7}) は、${DEVELOP_BRANCH} の履歴に含まれず、${DEVELOP_BRANCH} から切ったブランチ(切り元 ${BRANCH_POINT:0:7} が ${DEVELOP_BRANCH} の履歴にある。${DEVELOP_BRANCH} で検証されていないコミットを含む)。リリースは ${DEVELOP_BRANCH} → main の PR で出す"
  fi
  warn "マージコミット ${SHA:0:7} の第 2 親 ${BRANCH_TIP:0:7} は ${DEVELOP_BRANCH} の履歴に含まれない。main から切ったブランチ(hotfix)のマージとみなして、通した。ブランチ名は、デプロイ時には確かめられない"
  pass "hotfix のマージとみなした(切り元 ${BRANCH_POINT:0:7} は ${DEVELOP_BRANCH} の履歴にない)"
}

[[ "$REPO" =~ ^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$ ]] || die "REPO(owner/repo)を読めない。通さずに止めた"
is_sha "$SHA" || die "検査するコミットの SHA(40 桁の小文字の 16 進数)を読めない。通さずに止めた"

case "$MODE" in
  pull_request) check_pull_request ;;
  deploy) check_deploy ;;
  *) die "mode は pull_request か deploy を指定する" ;;
esac
