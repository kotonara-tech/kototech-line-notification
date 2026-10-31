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

func TestNewStartingSoon(t *testing.T) {
	p := mustProgram(t, program.Params{Name: "基調講演", StartAt: time.Date(2026, 11, 3, 15, 30, 0, 0, jst)})
	n := NewStartingSoon(p)
	if n.Kind != KindStartingSoon || n.Program != p {
		t.Errorf("種類と元のプログラムを持つはず: %+v", n)
	}
}

func TestNewStartingSoon_Key(t *testing.T) {
	start := time.Date(2026, 11, 3, 15, 30, 0, 0, jst)
	a := NewStartingSoon(mustProgram(t, program.Params{Name: "基調講演", StartAt: start}))
	b := NewStartingSoon(mustProgram(t, program.Params{Name: "基調講演", StartAt: start}))
	c := NewStartingSoon(mustProgram(t, program.Params{Name: "別のプログラム", StartAt: start}))

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

// 文面を、ドメインからテンプレートへ移しても、すでに送信した通知と同じキーになる(再送で、二重に配信されない)。
func TestNewStartingSoon_KeyIsStable(t *testing.T) {
	p := mustProgram(t, program.Params{ID: "program-1", Name: "基調講演", StartAt: time.Date(2026, 11, 3, 15, 30, 0, 0, jst)})
	want := uuidV5(namespace, "starting-soon:program-1")
	if got := NewStartingSoon(p).Key; got != want {
		t.Errorf("キーの作り方が変わった: %s, want %s", got, want)
	}
}

func TestKindsAreUnique(t *testing.T) {
	seen := map[Kind]bool{}
	for _, k := range AllKinds {
		if k == "" || seen[k] {
			t.Errorf("通知の種類が空、または重複している: %q", k)
		}
		seen[k] = true
	}
}
