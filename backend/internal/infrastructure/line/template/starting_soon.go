package template

import (
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
)

// startingSoonText は、プログラムがまもなく始まることを伝える、テキストメッセージのテンプレート。
// イベント名と場所は、あるときだけ出す。
const startingSoonText = `{{if .EventName}}【{{.EventName}}】
{{end}}まもなく開始します
{{.Name}}
開始時刻: {{.StartTime}}{{if .Location}}
場所: {{.Location}}{{end}}`

var startingSoon = newTemplate("starting-soon", startingSoonText)

// startingSoonData は、テンプレートに当てはめる値。ドメインの値を、表示用の文字列にしたもの。
type startingSoonData struct {
	EventName string
	Name      string
	StartTime string // 15:30 の形式
	Location  string
}

func renderStartingSoon(n notice.Notice) (Message, error) {
	p := n.Program
	text, err := execute(startingSoon, startingSoonData{
		EventName: p.EventName(),
		Name:      p.Name(),
		StartTime: p.StartAt().Format("15:04"),
		Location:  p.Location(),
	})
	if err != nil {
		return Message{}, err
	}
	return Message{Type: "text", Text: text}, nil
}
