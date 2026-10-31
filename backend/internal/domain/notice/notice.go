// Package notice は、友だちへ送る通知を表す。
//
// 「何を通知するか」(種類と、元になるプログラム)だけを持ち、文面は持たない。
// 文面は、送る側(infrastructure/line/template)が、種類ごとのテンプレートで作る。
// 通知の種類(text や Flex など、送る形式が違うもの)が増えても、ドメインを変えずに済むようにするため。
package notice

import "github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"

// namespace は、通知のキー(UUID v5)を作るための、このアプリ固有の名前空間。
var namespace = mustParseUUID("7d0b6f0e-3c1a-4f55-9d3e-2a8f6c1b9e44")

// Kind は通知の種類。文字列は、キーの一部になる(変えると、同じ通知でも別のキーになる)。
type Kind string

// 通知の種類の一覧。種類を足したら、AllKinds にも足し、infrastructure/line/template にテンプレートを足す。
const (
	// KindStartingSoon は、プログラムがまもなく始まることを伝える通知。
	KindStartingSoon Kind = "starting-soon"
)

// AllKinds は、すべての通知の種類。テンプレートの登録漏れを、テストで見つけるために使う。
var AllKinds = []Kind{KindStartingSoon}

// Notice は送信する通知。送信先(誰に送るか)も、文面も持たない。
type Notice struct {
	// Key は、同じ通知を表す安定した UUID。送信先のサービスが重複を判定するために使う。
	Key string
	// Kind は通知の種類。
	Kind Kind
	// Program は、通知の元になるプログラム。文面に差し込む値は、ここから取る。
	Program *program.Program
}

// NewStartingSoon は、プログラムがまもなく始まることを伝える通知を作る。
func NewStartingSoon(p *program.Program) Notice {
	return Notice{
		Key:     newKey(KindStartingSoon, p),
		Kind:    KindStartingSoon,
		Program: p,
	}
}

// newKey は、種類とプログラムから、安定したキーを作る。
// 同じ種類・同じプログラムなら、いつ作っても同じキーになる。
func newKey(kind Kind, p *program.Program) string {
	return uuidV5(namespace, string(kind)+":"+string(p.ID()))
}
