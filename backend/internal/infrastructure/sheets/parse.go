package sheets

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Sheets のシリアル値は、1899-12-30 からの日数(小数部は 1 日の中の割合)。
var serialEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// 日付として妥当なシリアル値の範囲(2000-01-01 以上、2100-01-01 未満)。
// 時刻の欄に 1030 のような整数、日付の欄に 20261103 のような整数を入れた入力ミスを、
// 黙って 0 時や遠い未来として読まずに、エラーにするため。
const (
	minDateSerial = 36526
	maxDateSerial = 73051
)

func isPlausibleDateSerial(f float64) bool {
	return f >= minDateSerial && f < maxDateSerial
}

var dateLayouts = []string{"2006/1/2", "2006-1-2", "2006年1月2日"}

var clockLayouts = []string{"15:04", "15:04:05", "3:04 PM", "3:04:05 PM", "3:04PM"}

// widthReplacer は、全角の数字・記号を半角にする(日本語入力で打たれた値を読めるようにする)。
var widthReplacer = strings.NewReplacer(
	"０", "0", "１", "1", "２", "2", "３", "3", "４", "4", "５", "5", "６", "6", "７", "7", "８", "8", "９", "9",
	"：", ":", "／", "/", "－", "-", "　", " ",
)

// text はセルの値を文字列にする。空なら空文字列。
func text(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "TRUE"
		}
		return "FALSE"
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

// parseDate は日付セルを、年月日にする。
// 数値(シリアル値)と、よくある文字列の形式に対応する。
func parseDate(v any, loc *time.Location) (time.Time, error) {
	if f, ok := v.(float64); ok {
		if !isPlausibleDateSerial(f) {
			return time.Time{}, fmt.Errorf("日付として不正な数値: %v(日付型のセルか、2026/11/03 の形式で入力する)", f)
		}
		d := serialEpoch.AddDate(0, 0, int(math.Floor(f)))
		return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc), nil
	}
	s := widthReplacer.Replace(text(v))
	for _, layout := range dateLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("日付として読めない: %q(例: 2026/11/03)", s)
}

// parseClock は時刻セルを、0 時からの経過時間にする。
func parseClock(v any) (time.Duration, error) {
	if f, ok := v.(float64); ok {
		// 時刻型のセルは 0 以上 1 未満。日時型のセルは、整数部が日付として妥当なときだけ、小数部を時刻として使う
		if f < 0 || (f >= 1 && !isPlausibleDateSerial(f)) {
			return 0, fmt.Errorf("時刻として不正な数値: %v(時刻型のセルか、15:30 の形式で入力する)", f)
		}
		frac := f - math.Floor(f)
		secs := int(math.Round(frac * 24 * 60 * 60))
		return time.Duration(secs) * time.Second, nil
	}
	s := widthReplacer.Replace(text(v))
	for _, layout := range clockLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute + time.Duration(t.Second())*time.Second, nil
		}
	}
	return 0, fmt.Errorf("時刻として読めない: %q(例: 15:30)", s)
}

// combine は日付と、0 時からの経過時間から、開始日時を作る。
// 夏時間などで時刻が飛ぶ場合も、壁時計の時刻どおりに作るため、Add ではなく Date を使う。
func combine(date time.Time, clock time.Duration, loc *time.Location) time.Time {
	secs := int(clock / time.Second)
	return time.Date(date.Year(), date.Month(), date.Day(), secs/3600, secs%3600/60, secs%60, 0, loc)
}

// disabledNotify は、「通知する」列で通知を止める値かを返す。
func disabledNotify(v any) bool {
	if b, ok := v.(bool); ok {
		return !b
	}
	switch strings.ToLower(widthReplacer.Replace(text(v))) {
	case "false", "no", "off", "いいえ", "しない", "×", "x":
		return true
	default:
		return false
	}
}
