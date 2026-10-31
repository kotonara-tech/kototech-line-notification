// Package template は、通知の文面(LINE に送るメッセージ)を、種類ごとのテンプレートで作る。
//
// テンプレートは、Go のファイルの中に、直接書く(text/template)。種類ごとに 1 ファイルで、
// このパッケージの外(ドメイン)は、文面を知らない。
//
// 設計上の判断(通常は、文面をドメインに置くが、ここでは意図的に外へ出している):
// このソフトは、複数のテンプレートメッセージ(text、将来は Flex など、送る形式が違うもの)を使う。
// 文面と、LINE の形式は、セットで変わるため、LINE に送る側(infrastructure/line)に置き、
// 送る側が、ドメイン(notice)を受け取って、テンプレートで描画する。依存は、infrastructure → domain の向きのまま。
//
// 通知の種類を足す手順:
//  1. domain/notice に、Kind と、NewXxx を足す(AllKinds にも足す)
//  2. このパッケージに、テンプレートのファイルを足す(starting_soon.go を参考にする)
//  3. renderers に登録する(登録漏れは、テストが見つける)
package template

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	texttemplate "text/template"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
)

// ErrUnknownKind は、テンプレートが登録されていない通知の種類。
var ErrUnknownKind = errors.New("テンプレートがない通知の種類")

// Message は、LINE の Messaging API に送るメッセージ 1 件。
// text のときは Text、flex のときは AltText と Contents を使う。
type Message struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	AltText  string          `json:"altText,omitempty"`
	Contents json.RawMessage `json:"contents,omitempty"`
}

type renderer func(n notice.Notice) (Message, error)

// renderers は、通知の種類ごとの描画。種類を足したら、ここに登録する。
var renderers = map[notice.Kind]renderer{
	notice.KindStartingSoon: renderStartingSoon,
}

// Render は、通知を、LINE に送るメッセージにする。
func Render(n notice.Notice) (Message, error) {
	r, ok := renderers[n.Kind]
	if !ok {
		return Message{}, fmt.Errorf("%w: %q", ErrUnknownKind, n.Kind)
	}
	if n.Program == nil {
		return Message{}, fmt.Errorf("通知 %q に、元のプログラムがない", n.Kind)
	}
	return r(n)
}

// funcs は、テンプレートで使える関数。
// json は、Flex Message のテンプレートで、値を JSON の文字列として差し込むために使う
// (引用符や改行を含む値でも、JSON が壊れないようにする)。
var funcs = texttemplate.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}

// newTemplate は、テンプレートを作る。書き方の誤りは、プログラムの誤りなので、起動時に panic する
// (テストで必ず見つかる)。
func newTemplate(name, text string) *texttemplate.Template {
	return texttemplate.Must(texttemplate.New(name).Funcs(funcs).Option("missingkey=error").Parse(text))
}

// execute は、テンプレートに値を当てはめる。
func execute(t *texttemplate.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("テンプレート %q の描画: %w", t.Name(), err)
	}
	return buf.String(), nil
}
