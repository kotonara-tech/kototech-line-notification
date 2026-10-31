// Package line は LINE Messaging API への送信を実装する。
package line

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// refreshMargin は、有効期限のどれだけ前から新しいトークンを発行するか。
const refreshMargin = time.Minute

// Credentials はステートレスなチャネルアクセストークンの発行に使う認証情報。
type Credentials struct {
	ChannelID     string
	ChannelSecret string
}

// String は秘匿情報を含まない表現を返す(誤ってログに出ても漏れないようにする)。
func (c Credentials) String() string {
	return "line.Credentials{ChannelID:" + c.ChannelID + ", ChannelSecret:****}"
}

// GoString は %#v でも、秘匿情報を隠す。
func (c Credentials) GoString() string { return c.String() }

// MarshalJSON は JSON にしても、秘匿情報を出さない。
func (c Credentials) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ChannelID     string `json:"channel_id"`
		ChannelSecret string `json:"channel_secret"`
	}{c.ChannelID, "****"})
}

// LogValue は slog に出力されたときも、秘匿情報を隠す。
func (c Credentials) LogValue() slog.Value {
	return slog.GroupValue(slog.String("channel_id", c.ChannelID), slog.String("channel_secret", "****"))
}

// tokenSource は、ステートレスなチャネルアクセストークン(有効期間 約 15 分)を発行し、
// 期限が近づくまでメモリに保持する。人手による更新・ローテーションを不要にするための仕組み。
type tokenSource struct {
	client   *http.Client
	endpoint string
	creds    Credentials
	now      func() time.Time

	mu     sync.Mutex
	token  string
	expiry time.Time
}

// get は有効なトークンを返す。期限が近ければ、新しく発行する。
// 同時に呼ばれても、発行は 1 回にまとまる(ロックを持ったまま発行するため)。
func (s *tokenSource) get(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.token != "" && s.now().Before(s.expiry.Add(-refreshMargin)) {
		return s.token, nil
	}
	token, expiresIn, err := s.issue(ctx)
	if err != nil {
		return "", err
	}
	s.token = token
	s.expiry = s.now().Add(expiresIn)
	return s.token, nil
}

// invalidate は、stale として渡したトークンを破棄する。
// すでに別のトークンに入れ替わっていたら、何もしない(新しいトークンを捨てないため)。
func (s *tokenSource) invalidate(stale string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == stale {
		s.token = ""
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (s *tokenSource) issue(ctx context.Context) (string, time.Duration, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {s.creds.ChannelID},
		"client_secret": {s.creds.ChannelSecret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("トークン発行のリクエスト作成: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("トークンの発行: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("トークンの発行に失敗: status %d: %s", resp.StatusCode, readSnippet(resp.Body))
	}
	var tr tokenResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tr); err != nil {
		return "", 0, fmt.Errorf("トークンのレスポンスの解析: %w", err)
	}
	if tr.AccessToken == "" || tr.ExpiresIn <= 0 {
		return "", 0, errors.New("トークンのレスポンスが不正(access_token または expires_in がない)")
	}
	return tr.AccessToken, time.Duration(tr.ExpiresIn) * time.Second, nil
}

// readSnippet は、エラー表示用にレスポンス本文の先頭だけを読む。
func readSnippet(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 300))
	return strings.TrimSpace(string(b))
}
