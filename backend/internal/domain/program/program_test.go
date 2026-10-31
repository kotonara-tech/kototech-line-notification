package program_test

import (
	"errors"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

var jst = time.FixedZone("JST", 9*60*60)

func newProgram(t *testing.T, startAt time.Time, mutate func(*program.Params)) *program.Program {
	t.Helper()
	p := program.Params{Name: "基調講演", StartAt: startAt, NotifyEnabled: true}
	if mutate != nil {
		mutate(&p)
	}
	got, err := program.New(p)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return got
}

func TestIsDueForNotice(t *testing.T) {
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, jst)
	lead, err := program.NewLeadTime(10)
	if err != nil {
		t.Fatal(err)
	}
	notifiedAt := now.Add(-time.Hour)

	tests := []struct {
		name   string
		start  time.Time
		mutate func(*program.Params)
		want   bool
	}{
		{name: "開始の 5 分前は対象", start: now.Add(5 * time.Minute), want: true},
		{name: "ちょうどリードタイム前は対象", start: now.Add(10 * time.Minute), want: true},
		{name: "リードタイムより 1 秒先は対象外", start: now.Add(10*time.Minute + time.Second), want: false},
		{name: "開始の 1 秒前は対象", start: now.Add(time.Second), want: true},
		{name: "ちょうど開始時刻は対象外", start: now, want: false},
		{name: "開始済みは対象外", start: now.Add(-time.Minute), want: false},
		{name: "通知済みは対象外", start: now.Add(5 * time.Minute), mutate: func(p *program.Params) { p.NotifiedAt = &notifiedAt }, want: false},
		{name: "通知が無効なら対象外", start: now.Add(5 * time.Minute), mutate: func(p *program.Params) { p.NotifyEnabled = false }, want: false},
		{name: "日付をまたぐ場合も判定できる", start: time.Date(2026, 11, 4, 0, 5, 0, 0, jst), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProgram(t, tt.start, tt.mutate)
			if got := p.IsDueForNotice(now, lead); got != tt.want {
				t.Errorf("IsDueForNotice = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsDueForNotice_AcrossMidnight(t *testing.T) {
	now := time.Date(2026, 11, 3, 23, 55, 0, 0, jst)
	lead, _ := program.NewLeadTime(10)
	p := newProgram(t, time.Date(2026, 11, 4, 0, 3, 0, 0, jst), nil)
	if !p.IsDueForNotice(now, lead) {
		t.Error("日付をまたいでも、8 分前なら対象になるはず")
	}
}

func TestMarkNotified(t *testing.T) {
	p := newProgram(t, time.Date(2026, 11, 3, 15, 0, 0, 0, jst), nil)
	at := time.Date(2026, 11, 3, 14, 55, 0, 0, jst)

	if err := p.MarkNotified(at); err != nil {
		t.Fatalf("1 回目: %v", err)
	}
	if !p.IsNotified() || !p.NotifiedAt().Equal(at) {
		t.Errorf("通知済みになっていない: %v", p.NotifiedAt())
	}
	if err := p.MarkNotified(at); !errors.Is(err, program.ErrAlreadyNotified) {
		t.Errorf("2 回目は ErrAlreadyNotified のはず: %v", err)
	}
}

func TestNew_Validation(t *testing.T) {
	start := time.Date(2026, 11, 3, 15, 0, 0, 0, jst)
	tests := []struct {
		name    string
		params  program.Params
		wantErr bool
	}{
		{name: "正常", params: program.Params{Name: "A", StartAt: start}},
		{name: "名前が空", params: program.Params{Name: "  ", StartAt: start}, wantErr: true},
		{name: "開始時刻が未設定", params: program.Params{Name: "A"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := program.New(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewID_Stable(t *testing.T) {
	start := time.Date(2026, 11, 3, 15, 0, 0, 0, jst)
	a := program.NewID(start, "Meetup", "基調講演", "ホール")
	if a != program.NewID(start.UTC(), " Meetup ", " 基調講演 ", "ホール ") {
		t.Error("同じ内容(タイムゾーン・前後の空白が違うだけ)なら同じ ID になるはず")
	}
	tests := []struct {
		name string
		id   program.ID
	}{
		{"開始時刻が違う", program.NewID(start.Add(time.Minute), "Meetup", "基調講演", "ホール")},
		{"イベント名が違う", program.NewID(start, "別のイベント", "基調講演", "ホール")},
		{"プログラム名が違う", program.NewID(start, "Meetup", "別の講演", "ホール")},
		{"場所が違う", program.NewID(start, "Meetup", "基調講演", "別室")},
	}
	for _, tt := range tests {
		if tt.id == a {
			t.Errorf("%sなら別の ID になるはず", tt.name)
		}
	}
}

func TestNewLeadTime(t *testing.T) {
	if _, err := program.NewLeadTime(0); err == nil {
		t.Error("0 分はエラーのはず")
	}
	l, err := program.NewLeadTime(15)
	if err != nil || l.Duration() != 15*time.Minute {
		t.Errorf("15 分: %v, %v", l, err)
	}
}
