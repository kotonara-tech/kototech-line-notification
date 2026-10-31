package template

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

var jst = time.FixedZone("JST", 9*60*60)

func newProgram(t *testing.T, p program.Params) *program.Program {
	t.Helper()
	p.NotifyEnabled = true
	got, err := program.New(p)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestRender_StartingSoon(t *testing.T) {
	start := time.Date(2026, 11, 3, 15, 30, 0, 0, jst)
	tests := []struct {
		name   string
		params program.Params
		want   string
	}{
		{
			name:   "すべての項目あり",
			params: program.Params{EventName: "ことならテック Meetup", Name: "基調講演", Location: "メインホール", StartAt: start},
			want:   "【ことならテック Meetup】\nまもなく開始します\n基調講演\n開始時刻: 15:30\n場所: メインホール",
		},
		{
			name:   "イベント名と場所なし",
			params: program.Params{Name: "休憩明けセッション", StartAt: start},
			want:   "まもなく開始します\n休憩明けセッション\n開始時刻: 15:30",
		},
		{
			name:   "場所だけなし",
			params: program.Params{EventName: "Meetup", Name: "LT", StartAt: start},
			want:   "【Meetup】\nまもなく開始します\nLT\n開始時刻: 15:30",
		},
		{
			name:   "日付をまたぐ 0 時台",
			params: program.Params{Name: "深夜の部", StartAt: time.Date(2026, 11, 4, 0, 5, 0, 0, jst)},
			want:   "まもなく開始します\n深夜の部\n開始時刻: 00:05",
		},
		{
			name:   "記号や引用符を含んでいても、そのまま出す(テキストなので、エスケープしない)",
			params: program.Params{Name: `A & B "C" <D>`, StartAt: start},
			want:   "まもなく開始します\nA & B \"C\" <D>\n開始時刻: 15:30",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := Render(notice.NewStartingSoon(newProgram(t, tt.params)))
			if err != nil {
				t.Fatal(err)
			}
			if msg.Type != "text" || msg.Text != tt.want {
				t.Errorf("Message = %+v\nwant text:\n%q", msg, tt.want)
			}
		})
	}
}

// 通知の種類を足したのに、テンプレートの登録を忘れたら、ここで見つかる。
func TestEveryKindHasATemplate(t *testing.T) {
	p := newProgram(t, program.Params{Name: "A", StartAt: time.Date(2026, 11, 3, 15, 30, 0, 0, jst)})
	for _, kind := range notice.AllKinds {
		t.Run(string(kind), func(t *testing.T) {
			if _, err := Render(notice.Notice{Key: "k", Kind: kind, Program: p}); err != nil {
				t.Errorf("テンプレートが登録されていない、または描画できない: %v", err)
			}
		})
	}
}

func TestRender_Errors(t *testing.T) {
	p := newProgram(t, program.Params{Name: "A", StartAt: time.Date(2026, 11, 3, 15, 30, 0, 0, jst)})

	if _, err := Render(notice.Notice{Kind: "no-such-kind", Program: p}); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("未知の種類は ErrUnknownKind のはず: %v", err)
	}
	if _, err := Render(notice.Notice{Kind: notice.KindStartingSoon}); err == nil {
		t.Error("元のプログラムがなければ、エラーのはず")
	}
}

// Flex Message のテンプレートで、値を JSON の文字列として差し込める(引用符や改行があっても、JSON が壊れない)。
func TestJSONFunc(t *testing.T) {
	tmpl := newTemplate("flex-example", `{"type":"bubble","body":{"text":{{json .Name}}}}`)
	name := "引用符\"と改行\nと バックスラッシュ\\"

	out, err := execute(tmpl, struct{ Name string }{name})
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Body struct{ Text string } `json:"body"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("JSON として読めない: %v\n%s", err, out)
	}
	if decoded.Body.Text != name {
		t.Errorf("値が変わった: %q", decoded.Body.Text)
	}
}

// 描画に必要な値が足りないと、黙って空にせず、エラーにする(missingkey=error)。
func TestExecute_MissingKeyIsError(t *testing.T) {
	tmpl := newTemplate("missing", `{{.NoSuchField}}`)
	if _, err := execute(tmpl, map[string]string{}); err == nil {
		t.Error("存在しない値の参照は、エラーのはず")
	}
}

func TestMessage_JSONShape(t *testing.T) {
	text, _ := json.Marshal(Message{Type: "text", Text: "こんにちは"})
	if string(text) != `{"type":"text","text":"こんにちは"}` {
		t.Errorf("text の形が違う: %s", text)
	}
	flex, _ := json.Marshal(Message{Type: "flex", AltText: "代替", Contents: json.RawMessage(`{"type":"bubble"}`)})
	for _, want := range []string{`"type":"flex"`, `"altText":"代替"`, `"contents":{"type":"bubble"}`} {
		if !strings.Contains(string(flex), want) {
			t.Errorf("flex に %s がない: %s", want, flex)
		}
	}
}
