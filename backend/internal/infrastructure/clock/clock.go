// Package clock は port.Clock の実装。
package clock

import "time"

// System は実際の現在時刻を、指定したタイムゾーンで返す。
type System struct {
	loc *time.Location
}

// New は System を作る。loc が nil なら UTC を使う。
func New(loc *time.Location) System {
	if loc == nil {
		loc = time.UTC
	}
	return System{loc: loc}
}

// Now は現在時刻を返す。
func (s System) Now() time.Time { return time.Now().In(s.loc) }
