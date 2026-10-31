package sheets

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

var jst = time.FixedZone("JST", 9*60*60)

type write struct {
	row, col int
	value    string
}

type fakeGateway struct {
	values   [][]any
	readErr  error
	writeErr error
	writes   []write
}

func (g *fakeGateway) readAll(context.Context) ([][]any, error) { return g.values, g.readErr }

func (g *fakeGateway) writeCell(_ context.Context, row, col int, value string) error {
	if g.writeErr != nil {
		return g.writeErr
	}
	g.writes = append(g.writes, write{row, col, value})
	return nil
}

// serialDate は日付のシリアル値(Sheets の内部表現)を返す。
func serialDate(y int, m time.Month, d int) float64 {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Sub(serialEpoch).Hours() / 24
}

// serialClock は時刻のシリアル値(1 日の中の割合)を返す。
func serialClock(h, m int) float64 { return float64(h*60+m) / (24 * 60) }

var header = []any{"日付", "開始時刻", "終了時刻", "プログラム名", "場所", "イベント名", "通知する", "通知済み", "メモ"}

func newRepo(values ...[]any) (*Repository, *fakeGateway, *bytes.Buffer) {
	gw := &fakeGateway{values: append([][]any{header}, values...)}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	return newRepository(gw, jst, logger), gw, &logs
}

func TestListAll_SerialValues(t *testing.T) {
	repo, _, _ := newRepo(
		[]any{serialDate(2026, 11, 3), serialClock(15, 30), serialClock(16, 30), "基調講演", "メインホール", "ことならテック Meetup", true, "", "メモ"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 {
		t.Fatalf("1 件のはず: %d", len(programs))
	}
	p := programs[0]
	if want := time.Date(2026, 11, 3, 15, 30, 0, 0, jst); !p.StartAt().Equal(want) {
		t.Errorf("StartAt = %v, want %v", p.StartAt(), want)
	}
	if p.Name() != "基調講演" || p.Location() != "メインホール" || p.EventName() != "ことならテック Meetup" {
		t.Errorf("項目が違う: %q %q %q", p.Name(), p.Location(), p.EventName())
	}
	if p.IsNotified() {
		t.Error("未通知のはず")
	}
}

func TestListAll_ColumnOrderAndExtraColumnsDoNotMatter(t *testing.T) {
	gw := &fakeGateway{values: [][]any{
		{"メモ", "通知済み", "プログラム名", "開始時刻", "日付"},
		{"x", "", "並べ替えた列", serialClock(9, 0), serialDate(2026, 11, 3)},
	}}
	programs, err := newRepository(gw, jst, nil).ListAll(t.Context())
	if err != nil || len(programs) != 1 {
		t.Fatalf("読めるはず: %v, %d", err, len(programs))
	}
	if programs[0].Name() != "並べ替えた列" || programs[0].StartAt().Hour() != 9 {
		t.Errorf("列の対応が違う: %v", programs[0].StartAt())
	}
}

func TestListAll_StringValues(t *testing.T) {
	tests := []struct {
		name        string
		date, clock any
		want        time.Time
	}{
		{"スラッシュ区切り", "2026/11/03", "15:30", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"ゼロ埋めなし", "2026/1/5", "9:05", time.Date(2026, 1, 5, 9, 5, 0, 0, jst)},
		{"ハイフン区切り", "2026-11-03", "15:30:00", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"年月日", "2026年11月3日", "15:30", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"全角", "２０２６／１１／０３", "１５：３０", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"午後(12 時間表記)", "2026/11/03", "3:30 PM", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"前後の空白", " 2026/11/03 ", " 15:30 ", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"日付が数値で時刻が文字列", serialDate(2026, 11, 3), "15:30", time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"日付が文字列で時刻が数値", "2026/11/03", serialClock(15, 30), time.Date(2026, 11, 3, 15, 30, 0, 0, jst)},
		{"日付をまたぐ 0 時台", "2026/11/04", "0:05", time.Date(2026, 11, 4, 0, 5, 0, 0, jst)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, _ := newRepo([]any{tt.date, tt.clock, "", "A"})
			programs, err := repo.ListAll(t.Context())
			if err != nil || len(programs) != 1 {
				t.Fatalf("読めるはず: %v, %d", err, len(programs))
			}
			if !programs[0].StartAt().Equal(tt.want) {
				t.Errorf("StartAt = %v, want %v", programs[0].StartAt(), tt.want)
			}
		})
	}
}

func TestSerialClock_RoundsToExactMinute(t *testing.T) {
	// 浮動小数点の誤差があっても、分単位の時刻がずれない
	for h := 0; h < 24; h++ {
		for m := 0; m < 60; m += 5 {
			d, err := parseClock(serialClock(h, m))
			if err != nil {
				t.Fatal(err)
			}
			if want := time.Duration(h)*time.Hour + time.Duration(m)*time.Minute; d != want {
				t.Fatalf("%02d:%02d → %v", h, m, d)
			}
		}
	}
	if d, _ := parseClock(math.Nextafter(serialClock(15, 30), 1)); d != 15*time.Hour+30*time.Minute {
		t.Errorf("わずかな誤差でずれた: %v", d)
	}
}

func TestListAll_SkipsBlankAndBadRows(t *testing.T) {
	repo, _, logs := newRepo(
		[]any{},
		[]any{serialDate(2026, 11, 3), serialClock(10, 0), "", "正常な行"},
		[]any{"", "", "", ""},
		[]any{"日付ではない", serialClock(11, 0), "", "日付が不正"},
		[]any{serialDate(2026, 11, 3), "時刻ではない", "", "時刻が不正"},
		[]any{serialDate(2026, 11, 3), serialClock(12, 0), "", ""},
		[]any{serialDate(2026, 11, 3), serialClock(13, 0), "", "末尾の行"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range programs {
		names = append(names, p.Name())
	}
	if got := strings.Join(names, ","); got != "正常な行,末尾の行" {
		t.Errorf("読めた行 = %s", got)
	}
	// 空行は警告にしない。読めない行は、行番号つきで警告する
	for _, want := range []string{"row=5", "row=6", "row=7"} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("警告に %s がない:\n%s", want, logs.String())
		}
	}
	if strings.Contains(logs.String(), "row=2 ") || strings.Contains(logs.String(), "row=4") {
		t.Errorf("空行・正常な行は警告しないはず:\n%s", logs.String())
	}
}

func TestListAll_NotifiedColumn(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)
	repo, _, _ := newRepo(
		[]any{d, c, "", "未通知", "", "", "", ""},
		[]any{d, c, "", "通知済み", "", "", "", "2026-11-03T09:55:00+09:00"},
		[]any{d, c, "", "読めない値", "", "", "", "なにか手で書いた"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range programs {
		got[p.Name()] = p.IsNotified()
	}
	want := map[string]bool{"未通知": false, "通知済み": true, "読めない値": true}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s の通知済み = %v, want %v", name, got[name], w)
		}
	}
}

func TestListAll_NotifyColumn(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)
	now := time.Date(2026, 11, 3, 9, 55, 0, 0, jst)
	lead, _ := program.NewLeadTime(10)

	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"チェックあり(TRUE)", true, true},
		{"チェックなし(FALSE)", false, false},
		{"空欄は通知する", "", true},
		{"文字列の FALSE", "FALSE", false},
		{"×", "×", false},
		{"しない", "しない", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _, _ := newRepo([]any{d, c, "", "A", "", "", tt.value, ""})
			programs, err := repo.ListAll(t.Context())
			if err != nil || len(programs) != 1 {
				t.Fatalf("%v, %d", err, len(programs))
			}
			if got := programs[0].IsDueForNotice(now, lead); got != tt.want {
				t.Errorf("通知対象 = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListAll_OptionalColumnsMayBeMissing(t *testing.T) {
	gw := &fakeGateway{values: [][]any{
		{"日付", "開始時刻", "プログラム名", "通知済み"},
		{serialDate(2026, 11, 3), serialClock(10, 0), "最小構成", ""},
	}}
	programs, err := newRepository(gw, jst, nil).ListAll(t.Context())
	if err != nil || len(programs) != 1 {
		t.Fatalf("%v, %d", err, len(programs))
	}
	lead, _ := program.NewLeadTime(10)
	if !programs[0].IsDueForNotice(time.Date(2026, 11, 3, 9, 55, 0, 0, jst), lead) {
		t.Error("「通知する」列がなければ、通知するはず")
	}
}

func TestListAll_MissingRequiredHeaders(t *testing.T) {
	tests := []struct {
		name   string
		values [][]any
		want   string
	}{
		{"空のシート", nil, "ヘッダー"},
		{"必須の列がない", [][]any{{"日付", "プログラム名"}}, "開始時刻、通知済み"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newRepository(&fakeGateway{values: tt.values}, jst, nil).ListAll(t.Context())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want %q を含む", err, tt.want)
			}
		})
	}
}

func TestListAll_ReadError(t *testing.T) {
	boom := errors.New("boom")
	_, err := newRepository(&fakeGateway{readErr: boom}, jst, nil).ListAll(t.Context())
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestListAll_DuplicateRowsAreSkipped(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)
	repo, _, logs := newRepo(
		[]any{d, c, "", "同じ", "ホール"},
		[]any{d, c, "", "同じ", "ホール"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil || len(programs) != 1 {
		t.Fatalf("%v, %d", err, len(programs))
	}
	if !strings.Contains(logs.String(), "row=3") {
		t.Errorf("重複の警告がない:\n%s", logs.String())
	}
}

func TestSave_WritesNotifiedAtToTheRightCell(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)
	repo, gw, _ := newRepo(
		[]any{d, c, "", "A"},
		[]any{d, c, "", "B"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	b := programs[1]
	at := time.Date(2026, 11, 3, 9, 55, 7, 0, time.UTC) // UTC で渡しても、JST で書かれる
	if err := b.MarkNotified(at); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	// 「通知済み」は 0 始まりで 7 列目、プログラム B は 3 行目
	want := write{row: 3, col: 7, value: "2026-11-03T18:55:07+09:00"}
	if len(gw.writes) != 1 || gw.writes[0] != want {
		t.Errorf("writes = %+v, want %+v", gw.writes, want)
	}
}

func TestSave_Errors(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)

	t.Run("ListAll していないプログラムは保存できない", func(t *testing.T) {
		repo, _, _ := newRepo()
		p, _ := program.New(program.Params{Name: "A", StartAt: time.Date(2026, 11, 3, 10, 0, 0, 0, jst), NotifyEnabled: true})
		_ = p.MarkNotified(time.Now())
		if err := repo.Save(t.Context(), p); !errors.Is(err, program.ErrNotFound) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("通知済みでなければ保存しない", func(t *testing.T) {
		repo, gw, _ := newRepo([]any{d, c, "", "A"})
		programs, _ := repo.ListAll(t.Context())
		if err := repo.Save(t.Context(), programs[0]); err == nil {
			t.Error("エラーのはず")
		}
		if len(gw.writes) != 0 {
			t.Error("書き込まないはず")
		}
	})
	t.Run("書き込みの失敗が返る", func(t *testing.T) {
		repo, gw, _ := newRepo([]any{d, c, "", "A"})
		programs, _ := repo.ListAll(t.Context())
		_ = programs[0].MarkNotified(time.Now())
		gw.writeErr = errors.New("quota")
		if err := repo.Save(t.Context(), programs[0]); !errors.Is(err, gw.writeErr) {
			t.Errorf("err = %v", err)
		}
	})
}

func TestColumnName(t *testing.T) {
	for col, want := range map[int]string{0: "A", 1: "B", 25: "Z", 26: "AA", 27: "AB", 51: "AZ", 52: "BA", 701: "ZZ", 702: "AAA"} {
		if got := columnName(col); got != want {
			t.Errorf("columnName(%d) = %s, want %s", col, got, want)
		}
	}
}

func TestQuoteSheet(t *testing.T) {
	for in, want := range map[string]string{"工程表": "'工程表'", "Sheet 1": "'Sheet 1'", "it's": "'it''s'"} {
		if got := quoteSheet(in); got != want {
			t.Errorf("quoteSheet(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestSave_FollowsRowsThatMovedAfterListAll(t *testing.T) {
	d := serialDate(2026, 11, 3)
	repo, gw, _ := newRepo(
		[]any{d, serialClock(10, 0), "", "A"},
		[]any{d, serialClock(13, 0), "", "B"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	a := programs[0]

	// 通知している間に、運用者が A の上へ行を挿入した(A は 2 行目から 3 行目へ)
	gw.values = append([][]any{gw.values[0], {d, serialClock(9, 0), "", "挿入された行"}}, gw.values[1:]...)

	if err := a.MarkNotified(time.Date(2026, 11, 3, 9, 55, 0, 0, jst)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	if len(gw.writes) != 1 || gw.writes[0].row != 3 {
		t.Errorf("挿入後の A の行(3 行目)に書くはず: %+v", gw.writes)
	}
}

func TestSave_DoesNotWriteWhenRowWasDeleted(t *testing.T) {
	d := serialDate(2026, 11, 3)
	repo, gw, _ := newRepo(
		[]any{d, serialClock(10, 0), "", "A"},
		[]any{d, serialClock(13, 0), "", "B"},
	)
	programs, _ := repo.ListAll(t.Context())
	a := programs[0]
	gw.values = append([][]any{gw.values[0]}, gw.values[2:]...) // A の行を削除(B が 2 行目に繰り上がる)

	_ = a.MarkNotified(time.Now())
	if err := repo.Save(t.Context(), a); !errors.Is(err, program.ErrNotFound) {
		t.Errorf("行がなければ ErrNotFound のはず: %v", err)
	}
	if len(gw.writes) != 0 {
		t.Errorf("別のプログラム(B)に書いてはいけない: %+v", gw.writes)
	}
}

func TestSave_DoesNotOverwriteWhenAlreadyRecorded(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)
	repo, gw, _ := newRepo([]any{d, c, "", "A"})
	programs, _ := repo.ListAll(t.Context())
	a := programs[0]

	// 別の実行が先に通知済みを書き込んだ
	gw.values[1] = []any{d, c, "", "A", "", "", "", "2026-11-03T09:54:00+09:00"}

	_ = a.MarkNotified(time.Now())
	if err := repo.Save(t.Context(), a); err != nil {
		t.Errorf("記録済みなら、何もせず成功のはず: %v", err)
	}
	if len(gw.writes) != 0 {
		t.Errorf("上書きしてはいけない: %+v", gw.writes)
	}
}

func TestListAll_SameProgramInDifferentEventsAreKept(t *testing.T) {
	d, c := serialDate(2026, 11, 3), serialClock(10, 0)
	repo, _, logs := newRepo(
		[]any{d, c, "", "オープニング", "ホール", "イベント A"},
		[]any{d, c, "", "オープニング", "ホール", "イベント B"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 2 {
		t.Errorf("イベント名が違えば別のプログラムとして残るはず: %d\n%s", len(programs), logs.String())
	}
}

func TestFindColumns_Warnings(t *testing.T) {
	tests := []struct {
		name   string
		header []any
		want   string // 警告に含まれるべき文字列。空なら警告なし
	}{
		{"警告なし", []any{"日付", "開始時刻", "プログラム名", "通知済み", "場所", "通知する", "メモ", "終了時刻"}, ""},
		{"通知するの打ち間違い", []any{"日付", "開始時刻", "プログラム名", "通知済み", "通知する?"}, "通知する"},
		{"場所の打ち間違い", []any{"日付", "開始時刻", "プログラム名", "通知済み", "開催場所"}, "場所"},
		{"イベント名の打ち間違い", []any{"日付", "開始時刻", "プログラム名", "通知済み", "イベント"}, "イベント名"},
		{"必須の見出しの重複", []any{"日付", "開始時刻", "プログラム名", "通知済み", "通知済み"}, "重複"},
		{"メモ列の重複は警告しない", []any{"日付", "開始時刻", "プログラム名", "通知済み", "メモ", "メモ"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, warnings, err := findColumns(tt.header)
			if err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(warnings, "\n")
			if tt.want == "" && len(warnings) != 0 {
				t.Errorf("警告は出ないはず: %v", warnings)
			}
			if tt.want != "" && !strings.Contains(joined, tt.want) {
				t.Errorf("%q を含む警告が出るはず: %v", tt.want, warnings)
			}
		})
	}
}

func TestListAll_LogsHeaderWarnings(t *testing.T) {
	gw := &fakeGateway{values: [][]any{{"日付", "開始時刻", "プログラム名", "通知済み", "通知する?"}}}
	var logs bytes.Buffer
	repo := newRepository(gw, jst, slog.New(slog.NewTextHandler(&logs, nil)))
	if _, err := repo.ListAll(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "通知する") {
		t.Errorf("見出しの警告が出るはず:\n%s", logs.String())
	}
}

func TestParse_RejectsImplausibleNumbers(t *testing.T) {
	if _, err := parseClock(1030.0); err == nil {
		t.Error("開始時刻に整数 1030 を入れた入力ミスは、エラーのはず")
	}
	if _, err := parseClock(-0.5); err == nil {
		t.Error("負の時刻は、エラーのはず")
	}
	if _, err := parseDate(20261103.0, jst); err == nil {
		t.Error("日付に整数 20261103 を入れた入力ミスは、エラーのはず")
	}
	if _, err := parseDate(0.0, jst); err == nil {
		t.Error("0 は日付として不正のはず")
	}

	// 日時型のセル(整数部が日付、小数部が時刻)は、時刻として読める
	dt := serialDate(2026, 11, 3) + serialClock(10, 30)
	d, err := parseClock(dt)
	if err != nil || d != 10*time.Hour+30*time.Minute {
		t.Errorf("日時型のセルから 10:30 を読めるはず: %v, %v", d, err)
	}
}

func TestListAll_ImplausibleNumbersAreSkippedWithWarning(t *testing.T) {
	d := serialDate(2026, 11, 3)
	repo, _, logs := newRepo(
		[]any{d, 1030.0, "", "開始時刻に整数"},
		[]any{20261103.0, serialClock(10, 0), "", "日付に整数"},
		[]any{d, serialClock(10, 0), "", "正常"},
	)
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(programs) != 1 || programs[0].Name() != "正常" {
		t.Errorf("正常な行だけ読めるはず: %d", len(programs))
	}
	if !strings.Contains(logs.String(), "row=2") || !strings.Contains(logs.String(), "row=3") {
		t.Errorf("読めない行は、行番号つきで警告するはず:\n%s", logs.String())
	}
}
