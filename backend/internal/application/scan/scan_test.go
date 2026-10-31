package scan_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/application/scan"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

var jst = time.FixedZone("JST", 9*60*60)

type fakeClock struct{ now time.Time }

func (c fakeClock) Now() time.Time { return c.now }

type fakeRepo struct {
	programs []*program.Program
	listErr  error
	saveErr  map[program.ID]error
	saved    []program.ID
}

func (r *fakeRepo) ListAll(context.Context) ([]*program.Program, error) {
	return r.programs, r.listErr
}

func (r *fakeRepo) Save(_ context.Context, p *program.Program) error {
	if err := r.saveErr[p.ID()]; err != nil {
		return err
	}
	r.saved = append(r.saved, p.ID())
	return nil
}

type fakeNotifier struct {
	sent []notice.Notice
	fail map[string]error // プログラム名 → エラー
}

func (n *fakeNotifier) Notify(_ context.Context, msg notice.Notice) error {
	if err := n.fail[msg.Program.Name()]; err != nil {
		return err
	}
	n.sent = append(n.sent, msg)
	return nil
}

var now = time.Date(2026, 11, 3, 15, 0, 0, 0, jst)

func prog(t *testing.T, name string, startOffset time.Duration, mutate func(*program.Params)) *program.Program {
	t.Helper()
	p := program.Params{Name: name, StartAt: now.Add(startOffset), NotifyEnabled: true}
	if mutate != nil {
		mutate(&p)
	}
	got, err := program.New(p)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func newUseCase(repo *fakeRepo, n *fakeNotifier) *scan.UseCase {
	lead, _ := program.NewLeadTime(10)
	return &scan.UseCase{Programs: repo, Notifier: n, Clock: fakeClock{now}, Lead: lead}
}

func TestExecute_NotifiesDueProgramsAndSaves(t *testing.T) {
	due := prog(t, "対象", 5*time.Minute, nil)
	later := prog(t, "まだ先", 30*time.Minute, nil)
	started := prog(t, "開始済み", -time.Minute, nil)
	done := time.Now()
	notified := prog(t, "通知済み", 5*time.Minute, func(p *program.Params) { p.NotifiedAt = &done })
	disabled := prog(t, "通知なし", 5*time.Minute, func(p *program.Params) { p.NotifyEnabled = false })

	repo := &fakeRepo{programs: []*program.Program{later, started, notified, disabled, due}}
	n := &fakeNotifier{}

	if err := newUseCase(repo, n).Execute(context.Background()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(n.sent) != 1 || n.sent[0].Program.Name() != "対象" {
		t.Fatalf("通知されたのは「対象」だけのはず: %+v", n.sent)
	}
	if len(repo.saved) != 1 || repo.saved[0] != due.ID() {
		t.Errorf("保存されたのは「対象」だけのはず: %v", repo.saved)
	}
	if !due.IsNotified() || !due.NotifiedAt().Equal(now) {
		t.Errorf("通知済みの時刻が記録されていない: %v", due.NotifiedAt())
	}
}

func TestExecute_NoDuplicateOnSecondRun(t *testing.T) {
	p := prog(t, "対象", 5*time.Minute, nil)
	repo := &fakeRepo{programs: []*program.Program{p}}
	n := &fakeNotifier{}
	uc := newUseCase(repo, n)

	for i := 0; i < 2; i++ {
		if err := uc.Execute(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(n.sent) != 1 {
		t.Errorf("2 回実行しても通知は 1 回のはず: %d", len(n.sent))
	}
}

func TestExecute_OneFailureDoesNotStopOthers(t *testing.T) {
	bad := prog(t, "失敗する", 3*time.Minute, nil)
	good := prog(t, "成功する", 6*time.Minute, nil)
	sendErr := errors.New("LINE が 500 を返した")

	repo := &fakeRepo{programs: []*program.Program{good, bad}}
	n := &fakeNotifier{fail: map[string]error{"失敗する": sendErr}}

	err := newUseCase(repo, n).Execute(context.Background())
	if !errors.Is(err, sendErr) {
		t.Fatalf("送信の失敗が返るはず: %v", err)
	}
	if len(n.sent) != 1 || n.sent[0].Program.Name() != "成功する" {
		t.Errorf("成功する方は通知されるはず: %+v", n.sent)
	}
	if bad.IsNotified() {
		t.Error("送信に失敗したものは通知済みにしない(次回に再試行する)")
	}
}

func TestExecute_SaveFailureIsReported(t *testing.T) {
	p := prog(t, "対象", 5*time.Minute, nil)
	saveErr := errors.New("Sheets が 503 を返した")
	repo := &fakeRepo{programs: []*program.Program{p}, saveErr: map[program.ID]error{p.ID(): saveErr}}

	err := newUseCase(repo, &fakeNotifier{}).Execute(context.Background())
	if !errors.Is(err, saveErr) {
		t.Errorf("保存の失敗が返るはず: %v", err)
	}
}

func TestExecute_ListFailure(t *testing.T) {
	listErr := errors.New("権限がない")
	err := newUseCase(&fakeRepo{listErr: listErr}, &fakeNotifier{}).Execute(context.Background())
	if !errors.Is(err, listErr) {
		t.Errorf("取得の失敗が返るはず: %v", err)
	}
}

func TestExecute_StopsOnCancelledContext(t *testing.T) {
	p := prog(t, "対象", 5*time.Minute, nil)
	n := &fakeNotifier{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := newUseCase(&fakeRepo{programs: []*program.Program{p}}, n).Execute(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("キャンセルが返るはず: %v", err)
	}
	if len(n.sent) != 0 {
		t.Error("キャンセル後は送信しない")
	}
}
