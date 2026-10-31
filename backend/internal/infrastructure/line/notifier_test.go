package line_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
	"github.com/kotonara-tech/kototech-line-notification/backend/internal/infrastructure/line"
)

// fakeLINE は LINE のトークン発行と broadcast を模したサーバ。
type fakeLINE struct {
	t *testing.T

	mu          sync.Mutex
	issued      int             // 発行したトークンの数
	valid       map[string]bool // 有効なトークン
	broadcasts  []broadcastCall
	broadcastFn func(call int, r *http.Request) (status int, body string) // nil なら常に 200
	tokenStatus int                                                       // 0 なら 200
	expiresIn   int
}

type broadcastCall struct {
	Authorization string
	RetryKey      string
	Texts         []string
}

func newFakeLINE(t *testing.T) (*fakeLINE, *httptest.Server) {
	t.Helper()
	f := &fakeLINE{t: t, valid: map[string]bool{}, expiresIn: 900}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/v3/token", f.handleToken)
	mux.HandleFunc("/v2/bot/message/broadcast", f.handleBroadcast)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakeLINE) handleToken(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if f.tokenStatus != 0 {
		w.WriteHeader(f.tokenStatus)
		_, _ = io.WriteString(w, `{"error":"invalid_client","error_description":"bad credentials"}`)
		return
	}
	if r.Form.Get("grant_type") != "client_credentials" || r.Form.Get("client_id") != "chan-id" || r.Form.Get("client_secret") != "chan-secret" {
		f.t.Errorf("トークン発行のパラメータが不正: %v", r.Form)
	}
	f.issued++
	token := fmt.Sprintf("token-%d", f.issued)
	f.valid[token] = true
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "expires_in": f.expiresIn, "token_type": "Bearer"})
}

func (f *fakeLINE) handleBroadcast(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !f.valid[token] {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"message":"Authentication failed"}`)
		return
	}
	var req struct {
		Messages []struct{ Type, Text string } `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	call := broadcastCall{Authorization: r.Header.Get("Authorization"), RetryKey: r.Header.Get("X-Line-Retry-Key")}
	for _, m := range req.Messages {
		if m.Type != "text" {
			f.t.Errorf("text 以外のメッセージ: %s", m.Type)
		}
		call.Texts = append(call.Texts, m.Text)
	}
	f.broadcasts = append(f.broadcasts, call)

	if f.broadcastFn != nil {
		status, body := f.broadcastFn(len(f.broadcasts), r)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
		return
	}
	_, _ = io.WriteString(w, "{}")
}

func (f *fakeLINE) revokeAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for k := range f.valid {
		f.valid[k] = false
	}
}

func newNotifier(srv *httptest.Server, now func() time.Time) *line.Notifier {
	return line.NewNotifier(line.Config{
		Credentials: line.Credentials{ChannelID: "chan-id", ChannelSecret: "chan-secret"},
		BaseURL:     srv.URL,
		Now:         now,
	})
}

var sampleNotice = notice.Notice{Key: "11111111-2222-5333-8444-555555555555", Text: "まもなく開始します"}

func TestNotify_SendsBroadcastWithTokenAndRetryKey(t *testing.T) {
	f, srv := newFakeLINE(t)
	n := newNotifier(srv, nil)

	if err := n.Notify(t.Context(), sampleNotice); err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if len(f.broadcasts) != 1 {
		t.Fatalf("broadcast は 1 回のはず: %d", len(f.broadcasts))
	}
	got := f.broadcasts[0]
	if got.Authorization != "Bearer token-1" {
		t.Errorf("Authorization = %q", got.Authorization)
	}
	if got.RetryKey != sampleNotice.Key {
		t.Errorf("X-Line-Retry-Key = %q", got.RetryKey)
	}
	if len(got.Texts) != 1 || got.Texts[0] != sampleNotice.Text {
		t.Errorf("本文 = %v", got.Texts)
	}
}

func TestNotify_ReusesTokenUntilExpiry(t *testing.T) {
	f, srv := newFakeLINE(t)
	now := time.Date(2026, 11, 3, 15, 0, 0, 0, time.UTC)
	n := newNotifier(srv, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if err := n.Notify(t.Context(), sampleNotice); err != nil {
			t.Fatal(err)
		}
	}
	if f.issued != 1 {
		t.Errorf("期限内ならトークンは 1 回だけ発行するはず: %d", f.issued)
	}

	// 有効期限(900 秒)の 1 分前を過ぎたら、新しいトークンを発行する
	now = now.Add(900*time.Second - 30*time.Second)
	if err := n.Notify(t.Context(), sampleNotice); err != nil {
		t.Fatal(err)
	}
	if f.issued != 2 {
		t.Errorf("期限が近ければ再発行するはず: %d", f.issued)
	}
	if got := f.broadcasts[len(f.broadcasts)-1].Authorization; got != "Bearer token-2" {
		t.Errorf("新しいトークンで送るはず: %q", got)
	}
}

func TestNotify_RefreshesTokenOnUnauthorized(t *testing.T) {
	f, srv := newFakeLINE(t)
	n := newNotifier(srv, nil)

	if err := n.Notify(t.Context(), sampleNotice); err != nil {
		t.Fatal(err)
	}
	f.revokeAll() // トークンが LINE 側で無効になった状況

	if err := n.Notify(t.Context(), sampleNotice); err != nil {
		t.Fatalf("401 のあとに、発行し直して成功するはず: %v", err)
	}
	if f.issued != 2 {
		t.Errorf("再発行は 1 回のはず: %d", f.issued)
	}
	last := f.broadcasts[len(f.broadcasts)-1]
	if last.Authorization != "Bearer token-2" || last.Texts[0] != sampleNotice.Text {
		t.Errorf("再試行のリクエストが不正(ボディが再送されていない可能性): %+v", last)
	}
}

func TestNotify_DoesNotRetryUnauthorizedForever(t *testing.T) {
	f, srv := newFakeLINE(t)
	f.broadcastFn = func(int, *http.Request) (int, string) { return http.StatusUnauthorized, `{"message":"nope"}` }
	// 有効なトークンでも 401 を返し続けるサーバ
	n := newNotifier(srv, nil)

	err := n.Notify(t.Context(), sampleNotice)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("401 のエラーが返るはず: %v", err)
	}
	if f.issued > 2 {
		t.Errorf("再発行は多くても 1 回のはず(合計 2 回まで): %d", f.issued)
	}
}

func TestNotify_ConflictIsSuccessButLogged(t *testing.T) {
	f, srv := newFakeLINE(t)
	f.broadcastFn = func(int, *http.Request) (int, string) {
		return http.StatusConflict, `{"message":"The retry key is already accepted"}`
	}
	var logs bytes.Buffer
	n := line.NewNotifier(line.Config{
		Credentials: line.Credentials{ChannelID: "chan-id", ChannelSecret: "chan-secret"},
		BaseURL:     srv.URL,
		Logger:      slog.New(slog.NewTextHandler(&logs, nil)),
	})

	if err := n.Notify(t.Context(), sampleNotice); err != nil {
		t.Errorf("409(受付済み)は成功として扱うはず: %v", err)
	}
	if !strings.Contains(logs.String(), sampleNotice.Key) || !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("409 は、リトライキーつきの警告として記録するはず:\n%s", logs.String())
	}
}

func TestNotify_ErrorStatuses(t *testing.T) {
	tests := []struct {
		status int
		body   string
	}{
		{http.StatusBadRequest, `{"message":"invalid"}`},
		{http.StatusTooManyRequests, `{"message":"rate limit"}`},
		{http.StatusInternalServerError, `oops`},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			f, srv := newFakeLINE(t)
			f.broadcastFn = func(int, *http.Request) (int, string) { return tt.status, tt.body }
			err := newNotifier(srv, nil).Notify(t.Context(), sampleNotice)
			if err == nil || !strings.Contains(err.Error(), fmt.Sprint(tt.status)) {
				t.Errorf("status %d のエラーが返るはず: %v", tt.status, err)
			}
		})
	}
}

func TestNotify_TokenIssueFailure(t *testing.T) {
	f, srv := newFakeLINE(t)
	f.tokenStatus = http.StatusBadRequest

	err := newNotifier(srv, nil).Notify(t.Context(), sampleNotice)
	if err == nil || !strings.Contains(err.Error(), "トークンの発行に失敗") {
		t.Fatalf("トークン発行の失敗が返るはず: %v", err)
	}
	if len(f.broadcasts) != 0 {
		t.Error("トークンが取れなければ、送信しない")
	}
	if strings.Contains(err.Error(), "chan-secret") {
		t.Error("エラーにチャネルシークレットが含まれている")
	}
}

func TestNotify_ConcurrentCallsIssueOneToken(t *testing.T) {
	f, srv := newFakeLINE(t)
	n := newNotifier(srv, nil)

	var wg sync.WaitGroup
	var failures atomic.Int32
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.Notify(t.Context(), sampleNotice); err != nil {
				failures.Add(1)
			}
		}()
	}
	wg.Wait()

	if failures.Load() != 0 {
		t.Errorf("失敗: %d", failures.Load())
	}
	if f.issued != 1 {
		t.Errorf("同時に呼んでも、発行は 1 回のはず: %d", f.issued)
	}
}

func TestCredentials_DoesNotLeakSecret(t *testing.T) {
	c := line.Credentials{ChannelID: "chan-id", ChannelSecret: "super-secret"}
	j, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	slog.New(slog.NewJSONHandler(&logs, nil)).Info("確認", "creds", c)

	outputs := map[string]string{
		"書式化":  fmt.Sprintf("%v %+v %#v %s", c, c, c, c),
		"JSON": string(j),
		"slog": logs.String(),
	}
	for name, out := range outputs {
		if strings.Contains(out, "super-secret") {
			t.Errorf("%s にシークレットが含まれている: %s", name, out)
		}
	}
}
