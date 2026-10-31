// Package sheets は、スプレッドシートの工程表を program.Repository として扱う。
//
// 工程表のレイアウト(ヘッダー名による列の対応)はこのパッケージに閉じ込める。
// ドメインは、シートの形を知らない。
package sheets

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/program"
)

// ヘッダー(1 行目)の名前。列の順序は問わない。ここにない列は無視する。
const (
	headerDate     = "日付"
	headerStart    = "開始時刻"
	headerName     = "プログラム名"
	headerLocation = "場所"
	headerEvent    = "イベント名"
	headerNotify   = "通知する"
	headerNotified = "通知済み"
)

// gateway は Sheets API への最小限の窓口。テストでは偽物に差し替える。
type gateway interface {
	// readAll はシート全体を行ごとに返す。日付・時刻は数値(シリアル値)で返ってくる。
	readAll(ctx context.Context) ([][]any, error)
	// writeCell は 1 つのセルに書き込む(row は 1 始まり、col は 0 始まり)。
	writeCell(ctx context.Context, row, col int, value string) error
}

// Config は Repository の設定。
type Config struct {
	SpreadsheetID string
	SheetName     string
	Location      *time.Location
	Logger        *slog.Logger
}

// columns は、ヘッダーから求めた列の位置(0 始まり。なければ -1)。
type columns struct {
	date, start, name, location, event, notify, notified int
}

// Repository は program.Repository の実装。
// 行の位置は保持しない。書き込みのたびに読み直して、その時点の行を探す。
type Repository struct {
	gw     gateway
	loc    *time.Location
	logger *slog.Logger
}

var _ program.Repository = (*Repository)(nil)

// New は Google Sheets API を使う Repository を作る。
func New(ctx context.Context, cfg Config) (*Repository, error) {
	gw, err := newGoogleGateway(ctx, cfg.SpreadsheetID, cfg.SheetName)
	if err != nil {
		return nil, err
	}
	return newRepository(gw, cfg.Location, cfg.Logger), nil
}

func newRepository(gw gateway, loc *time.Location, logger *slog.Logger) *Repository {
	if loc == nil {
		loc = time.UTC
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Repository{gw: gw, loc: loc, logger: logger}
}

// snapshot は、ある時点の工程表の内容。
type snapshot struct {
	programs []*program.Program
	rows     map[program.ID]int // プログラムの行(1 始まり)
	cols     columns
}

// load は工程表を読んで、プログラムと、それぞれの行の位置を返す。
// 空行は黙って飛ばす。読めない行は、警告を出して飛ばす(1 行の入力ミスで全体を止めないため)。
func (r *Repository) load(ctx context.Context, logger *slog.Logger) (snapshot, error) {
	values, err := r.gw.readAll(ctx)
	if err != nil {
		return snapshot{}, err
	}
	if len(values) == 0 {
		return snapshot{}, errors.New("工程表が空(1 行目にヘッダーが必要)")
	}
	cols, warnings, err := findColumns(values[0])
	if err != nil {
		return snapshot{}, err
	}
	for _, w := range warnings {
		logger.Warn("工程表のヘッダーに注意", slog.String("detail", w))
	}

	snap := snapshot{
		programs: make([]*program.Program, 0, len(values)),
		rows:     make(map[program.ID]int, len(values)),
		cols:     cols,
	}
	for i := 1; i < len(values); i++ {
		sheetRow := i + 1
		p, err := r.parseRow(values[i], cols)
		if err != nil {
			if !errors.Is(err, errBlankRow) {
				logger.Warn("工程表の行を読めないため飛ばした",
					slog.Int("row", sheetRow), slog.String("error", err.Error()))
			}
			continue
		}
		if _, dup := snap.rows[p.ID()]; dup {
			logger.Warn("同じ内容の行が複数あるため飛ばした(イベント名・開始時刻・プログラム名・場所が同じ)",
				slog.Int("row", sheetRow), slog.String("program", p.Name()))
			continue
		}
		snap.rows[p.ID()] = sheetRow
		snap.programs = append(snap.programs, p)
	}
	return snap, nil
}

// ListAll は工程表のすべてのプログラムを返す。
func (r *Repository) ListAll(ctx context.Context) ([]*program.Program, error) {
	snap, err := r.load(ctx, r.logger)
	if err != nil {
		return nil, err
	}
	return snap.programs, nil
}

// Save は、通知済みの日時を「通知済み」列に書き込む。
//
// 書き込む直前に工程表を読み直し、その時点でのプログラムの行を探す。
// ListAll のあとに、運用者が行を挿入・削除・並べ替えしても、別のプログラムを通知済みにしないため。
// 読み直しから書き込みまでの間の編集は防げない(Sheets API に、条件つきの書き込みがない)が、
// その窓は 1 回の API 呼び出しの間だけで、数百ミリ秒に収まる。
func (r *Repository) Save(ctx context.Context, p *program.Program) error {
	at := p.NotifiedAt()
	if at == nil {
		return fmt.Errorf("通知済みではないプログラムは保存できない: %s", p.Name())
	}

	// 読み直しの警告は ListAll で出しているため、ここでは出さない(毎回の重複を避ける)
	snap, err := r.load(ctx, slog.New(slog.DiscardHandler))
	if err != nil {
		return err
	}
	row, ok := snap.rows[p.ID()]
	if !ok {
		return fmt.Errorf("%w: %s(読み取り後に、行が変更・削除された可能性がある)", program.ErrNotFound, p.Name())
	}
	for _, fresh := range snap.programs {
		if fresh.ID() == p.ID() && fresh.IsNotified() {
			// すでに通知済みが記録されている(別の実行が先に書いた、または手で入力された)。上書きしない
			return nil
		}
	}
	return r.gw.writeCell(ctx, row, snap.cols.notified, at.In(r.loc).Format(time.RFC3339))
}

var errBlankRow = errors.New("空行")

// knownHeaders は、アプリが読む列のヘッダー名。
var knownHeaders = []string{headerDate, headerStart, headerName, headerLocation, headerEvent, headerNotify, headerNotified}

// similarHints は、任意の列の打ち間違いを見つけるための手がかり。
// 見出しがこの語を含むのに、正しいヘッダー名と一致しないときに警告する。
var similarHints = []struct{ word, want string }{
	{"通知", headerNotify},
	{"場所", headerLocation},
	{"イベント", headerEvent},
}

func normalizeHeader(v any) string {
	return strings.TrimSpace(widthReplacer.Replace(text(v)))
}

// findColumns は、ヘッダー行から各列の位置を求める。必須の列がなければエラー。
// 実行は止めないが、運用で気づくべき点(見出しの重複、任意の列の打ち間違いの疑い)は警告として返す。
// 任意の列の見出しを打ち間違えると、その列は黙って無視されるため、とくに「通知する」の打ち間違いは、
// 通知しないつもりの行まで通知してしまう。
func findColumns(header []any) (columns, []string, error) {
	idx := map[string]int{}
	var warnings []string
	for i, h := range header {
		name := normalizeHeader(h)
		if name == "" {
			continue
		}
		if first, exists := idx[name]; exists {
			if slices.Contains(knownHeaders, name) {
				warnings = append(warnings, fmt.Sprintf("見出し %q が重複している(最初の列 %s だけを使う。重複は %s)",
					name, columnName(first), columnName(i)))
			}
			continue
		}
		idx[name] = i
	}
	for i, h := range header {
		name := normalizeHeader(h)
		if name == "" || slices.Contains(knownHeaders, name) {
			continue
		}
		for _, hint := range similarHints {
			if strings.Contains(name, hint.word) {
				warnings = append(warnings, fmt.Sprintf("見出し %q(%s 列)は読まれない。%q の打ち間違いなら直す", name, columnName(i), hint.want))
				break
			}
		}
	}

	find := func(name string) int {
		if i, ok := idx[name]; ok {
			return i
		}
		return -1
	}
	c := columns{
		date: find(headerDate), start: find(headerStart), name: find(headerName),
		location: find(headerLocation), event: find(headerEvent),
		notify: find(headerNotify), notified: find(headerNotified),
	}

	var missing []string
	for _, req := range []struct {
		col  int
		name string
	}{{c.date, headerDate}, {c.start, headerStart}, {c.name, headerName}, {c.notified, headerNotified}} {
		if req.col < 0 {
			missing = append(missing, req.name)
		}
	}
	if len(missing) > 0 {
		return columns{}, nil, fmt.Errorf("工程表の 1 行目に必須のヘッダーがない: %s", strings.Join(missing, "、"))
	}
	return c, warnings, nil
}

// cell は、行の指定した列の値を返す。列が行の長さを超えていれば(末尾の空セルは省略されるため)nil。
func cell(row []any, col int) any {
	if col < 0 || col >= len(row) {
		return nil
	}
	return row[col]
}

func (r *Repository) parseRow(row []any, c columns) (*program.Program, error) {
	dateCell, startCell, nameCell := cell(row, c.date), cell(row, c.start), cell(row, c.name)
	if text(dateCell) == "" && text(startCell) == "" && text(nameCell) == "" {
		return nil, errBlankRow
	}

	date, err := parseDate(dateCell, r.loc)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", headerDate, err)
	}
	clock, err := parseClock(startCell)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", headerStart, err)
	}

	var notifiedAt *time.Time
	if raw := text(cell(row, c.notified)); raw != "" {
		// 読めない値が入っていても、「通知済み」として扱う(二重送信を避けるため)
		t, perr := time.Parse(time.RFC3339, raw)
		if perr != nil {
			t = time.Time{}
		}
		notifiedAt = &t
	}

	return program.New(program.Params{
		EventName:     text(cell(row, c.event)),
		Name:          text(nameCell),
		Location:      text(cell(row, c.location)),
		StartAt:       combine(date, clock, r.loc),
		NotifyEnabled: !disabledNotify(cell(row, c.notify)),
		NotifiedAt:    notifiedAt,
	})
}
