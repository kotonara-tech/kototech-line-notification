package notice

import (
	"strings"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

var jst = time.FixedZone("JST", 9*60*60)

// RFC 4122 の名前空間 DNS と "python.org" から作る UUID v5 は、広く知られた値になる。
func TestUUIDv5_KnownVector(t *testing.T) {
	dns := mustParseUUID("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	if got, want := uuidV5(dns, "python.org"), "886313e1-3b8a-5372-9b90-0c9aee199e5d"; got != want {
		t.Errorf("uuidV5 = %s, want %s", got, want)
	}
}

func mustProgram(t *testing.T, p program.Params) *program.Program {
	t.Helper()
	p.NotifyEnabled = true
	got, err := program.New(p)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestStartingSoon_Text(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StartingSoon(mustProgram(t, tt.params)).Text; got != tt.want {
				t.Errorf("Text:\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}

func TestStartingSoon_Key(t *testing.T) {
	start := time.Date(2026, 11, 3, 15, 30, 0, 0, jst)
	a := StartingSoon(mustProgram(t, program.Params{Name: "基調講演", StartAt: start}))
	b := StartingSoon(mustProgram(t, program.Params{Name: "基調講演", StartAt: start}))
	c := StartingSoon(mustProgram(t, program.Params{Name: "別のプログラム", StartAt: start}))

	if a.Key != b.Key {
		t.Errorf("同じプログラムなら同じキーのはず: %s / %s", a.Key, b.Key)
	}
	if a.Key == c.Key {
		t.Error("別のプログラムなら別のキーのはず")
	}
	if len(a.Key) != 36 || strings.Count(a.Key, "-") != 4 || a.Key[14] != '5' {
		t.Errorf("UUID v5 の形式ではない: %s", a.Key)
	}
}
