// Package program は、イベント内の 1 つの進行単位(プログラム)に関するビジネスルールを持つ。
// 標準ライブラリのみに依存し、外部サービスの事情を知らない。
package program

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrAlreadyNotified は、通知済みのプログラムを再度通知済みにしようとしたときのエラー。
	ErrAlreadyNotified = errors.New("すでに通知済み")
	// ErrNotFound は、保存対象のプログラムが見つからないときのエラー。
	ErrNotFound = errors.New("プログラムが見つからない")
)

// ID はプログラムの識別子。開始日時・名前・場所から決まるため、同じ内容なら同じ ID になる。
type ID string

// NewID は、プログラムの内容から安定した ID を作る。
// 行の位置に依存させないのは、工程表の行が挿入・並べ替えされても同一視できるようにするため。
// イベント名を含めるのは、複数のイベントを 1 つの工程表で扱うとき、別イベントの同名・同時刻・同場所の
// プログラムが、同じ ID(同じ LINE のリトライキー)になって落ちるのを防ぐため。
func NewID(startAt time.Time, eventName, name, location string) ID {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		startAt.UTC().Format(time.RFC3339),
		strings.TrimSpace(eventName),
		strings.TrimSpace(name),
		strings.TrimSpace(location),
	}, "\x00")))
	return ID(hex.EncodeToString(sum[:16]))
}

// LeadTime は、開始の何分前から通知するか。
type LeadTime time.Duration

// NewLeadTime は分から LeadTime を作る。
func NewLeadTime(minutes int) (LeadTime, error) {
	if minutes <= 0 {
		return 0, fmt.Errorf("リードタイムは 1 分以上にする: %d", minutes)
	}
	return LeadTime(time.Duration(minutes) * time.Minute), nil
}

// Duration は time.Duration として返す。
func (l LeadTime) Duration() time.Duration { return time.Duration(l) }

// Params は Program を作るための値。
type Params struct {
	ID            ID
	EventName     string
	Name          string
	Location      string
	StartAt       time.Time
	NotifyEnabled bool
	NotifiedAt    *time.Time
}

// Program はプログラム(集約ルート)。
type Program struct {
	id            ID
	eventName     string
	name          string
	location      string
	startAt       time.Time
	notifyEnabled bool
	notifiedAt    *time.Time
}

// New は Program を作る。名前と開始時刻は必須。
func New(p Params) (*Program, error) {
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("プログラム名が空")
	}
	if p.StartAt.IsZero() {
		return nil, errors.New("開始時刻が未設定")
	}
	id := p.ID
	if id == "" {
		id = NewID(p.StartAt, p.EventName, p.Name, p.Location)
	}
	return &Program{
		id:            id,
		eventName:     strings.TrimSpace(p.EventName),
		name:          strings.TrimSpace(p.Name),
		location:      strings.TrimSpace(p.Location),
		startAt:       p.StartAt,
		notifyEnabled: p.NotifyEnabled,
		notifiedAt:    p.NotifiedAt,
	}, nil
}

// ID を返す。
func (p *Program) ID() ID { return p.id }

// EventName を返す。未設定なら空。
func (p *Program) EventName() string { return p.eventName }

// Name を返す。
func (p *Program) Name() string { return p.name }

// Location を返す。未設定なら空。
func (p *Program) Location() string { return p.location }

// StartAt を返す。
func (p *Program) StartAt() time.Time { return p.startAt }

// IsNotified は通知済みかを返す。
func (p *Program) IsNotified() bool { return p.notifiedAt != nil }

// NotifiedAt は通知した日時を返す。未通知なら nil。
func (p *Program) NotifiedAt() *time.Time { return p.notifiedAt }

// IsDueForNotice は、いま通知すべきかを返す。
// 通知が有効で、未通知で、開始まで lead 以内(ちょうど lead も含む)かつ開始前のとき true。
// 開始済みのプログラムは、遅れて気づいても通知しない。
// 時刻は引数で受け取る(テストで固定できるようにするため)。
func (p *Program) IsDueForNotice(now time.Time, lead LeadTime) bool {
	if !p.notifyEnabled || p.IsNotified() {
		return false
	}
	untilStart := p.startAt.Sub(now)
	return untilStart > 0 && untilStart <= lead.Duration()
}

// MarkNotified は通知済みにする。すでに通知済みならエラー。
func (p *Program) MarkNotified(at time.Time) error {
	if p.IsNotified() {
		return ErrAlreadyNotified
	}
	p.notifiedAt = &at
	return nil
}

// Repository はプログラムの保管先。実装は infrastructure に置く。
type Repository interface {
	// ListAll はすべてのプログラムを返す。
	ListAll(ctx context.Context) ([]*Program, error)
	// Save は、通知済みの状態を保存する。ListAll で取得したプログラムだけを対象にする。
	Save(ctx context.Context, p *Program) error
}
