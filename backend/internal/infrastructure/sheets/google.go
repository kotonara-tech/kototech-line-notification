package sheets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/api/googleapi"
	impersonateapi "google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	sheetsapi "google.golang.org/api/sheets/v4"
)

const requestTimeout = 30 * time.Second

// googleGateway は、Google Sheets API を使った gateway の実装。
// 認証は Application Default Credentials(Cloud Run の実行用サービスアカウント)に任せる。
type googleGateway struct {
	svc           *sheetsapi.Service
	spreadsheetID string
	sheetName     string
}

// newGoogleGateway は Sheets API のクライアントを作る。
// impersonate が空でなければ、そのサービスアカウントのトークンを借りて呼び出す
// (CI のインテグレーションテストが、本番の Job と同じ「実行用サービスアカウント」として読むため)。
func newGoogleGateway(ctx context.Context, spreadsheetID, sheetName, impersonate string) (*googleGateway, error) {
	opts := []option.ClientOption{option.WithScopes(sheetsapi.SpreadsheetsScope)}
	if impersonate != "" {
		ts, err := impersonateapi.CredentialsTokenSource(ctx, impersonateapi.CredentialsConfig{
			TargetPrincipal: impersonate,
			Scopes:          []string{sheetsapi.SpreadsheetsScope},
		})
		if err != nil {
			return nil, fmt.Errorf("サービスアカウント %s の借用: %w", impersonate, err)
		}
		opts = []option.ClientOption{option.WithTokenSource(ts)}
	}
	svc, err := sheetsapi.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("sheets API クライアントの作成: %w", err)
	}
	return &googleGateway{svc: svc, spreadsheetID: spreadsheetID, sheetName: sheetName}, nil
}

// readAll はシート全体を行ごとに読む。
// 日付・時刻はロケールや表示形式に左右されないよう、シリアル値(数値)で受け取る。
func (g *googleGateway) readAll(ctx context.Context) ([][]any, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	resp, err := g.svc.Spreadsheets.Values.Get(g.spreadsheetID, quoteSheet(g.sheetName)).
		MajorDimension("ROWS").
		ValueRenderOption("UNFORMATTED_VALUE").
		DateTimeRenderOption("SERIAL_NUMBER").
		Context(ctx).Do()
	if err != nil {
		return nil, explain(err, g.sheetName)
	}
	return resp.Values, nil
}

// writeCell は 1 つのセルに文字列を書き込む(row は 1 始まり、col は 0 始まり)。
func (g *googleGateway) writeCell(ctx context.Context, row, col int, value string) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	cell := quoteSheet(g.sheetName) + "!" + columnName(col) + strconv.Itoa(row)
	_, err := g.svc.Spreadsheets.Values.Update(g.spreadsheetID, cell, &sheetsapi.ValueRange{
		Values: [][]any{{value}},
	}).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return explain(err, g.sheetName)
	}
	return nil
}

// explain は、運用で起きやすい失敗に、原因のヒントを足す。
func explain(err error, sheetName string) error {
	var apiErr *googleapi.Error
	if !errors.As(err, &apiErr) {
		return fmt.Errorf("sheets API: %w", err)
	}
	switch apiErr.Code {
	case http.StatusForbidden:
		return fmt.Errorf("sheets API: 権限がない。スプレッドシートを、実行用サービスアカウント(Cloud Run Job が使う。CI のテストも、これを借りて読む)に共有(編集者)しているか確認する: %w", err)
	case http.StatusNotFound:
		return fmt.Errorf("sheets API: スプレッドシート ID が違う、またはシート %q が存在しない: %w", sheetName, err)
	case http.StatusBadRequest:
		return fmt.Errorf("sheets API: リクエストが不正(シート名 %q やセルの指定を確認する): %w", sheetName, err)
	default:
		return fmt.Errorf("sheets API: %w", err)
	}
}

// quoteSheet は、A1 表記で使うシート名を引用符で囲む(シート名に空白や記号があっても使えるようにする)。
func quoteSheet(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

// columnName は 0 始まりの列番号を、A, B, ..., Z, AA の形式にする。
func columnName(col int) string {
	name := ""
	for n := col + 1; n > 0; n = (n - 1) / 26 {
		name = string(rune('A'+(n-1)%26)) + name
	}
	return name
}
