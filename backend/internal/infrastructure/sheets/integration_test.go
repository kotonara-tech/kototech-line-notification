//go:build integration

package sheets_test

import (
	"os"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/infrastructure/sheets"
)

// 実際のスプレッドシートを読めることと、工程表の形式が読めることを確認する。
// 読み取りだけで、書き込みはしない。dev 環境のスプレッドシートに対してのみ実行すること。
//
// 必要なもの:
//   - SPREADSHEET_ID: dev の工程表のスプレッドシート ID
//   - SHEET_NAME(任意): シート名。省略すると「工程表」
//   - Application Default Credentials に、スプレッドシートを読める権限
//     (例: gcloud auth application-default login --scopes=https://www.googleapis.com/auth/spreadsheets,https://www.googleapis.com/auth/cloud-platform)
func TestIntegration_ListAll(t *testing.T) {
	id := os.Getenv("SPREADSHEET_ID")
	if id == "" {
		t.Skip("SPREADSHEET_ID が未設定")
	}
	sheetName := os.Getenv("SHEET_NAME")
	if sheetName == "" {
		sheetName = "工程表"
	}
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	repo, err := sheets.New(t.Context(), sheets.Config{SpreadsheetID: id, SheetName: sheetName, Location: jst})
	if err != nil {
		t.Fatal(err)
	}
	programs, err := repo.ListAll(t.Context())
	if err != nil {
		t.Fatalf("工程表を読めない: %v", err)
	}
	t.Logf("読めたプログラム: %d 件", len(programs))
	for _, p := range programs {
		t.Logf("  %s  %s  (場所: %s / 通知済み: %v)", p.StartAt().Format("2006-01-02 15:04"), p.Name(), p.Location(), p.IsNotified())
	}
}
