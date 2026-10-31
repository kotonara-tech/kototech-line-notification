package line

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kotonara-tech/kototech-line-notification/backend/internal/domain/notice"
)

const (
	defaultBaseURL = "https://api.line.me"
	httpTimeout    = 15 * time.Second
)

// Config は Notifier の設定。
type Config struct {
	Credentials Credentials
	// BaseURL は API のベース URL。空なら本番の URL。テストで差し替える。
	BaseURL string
	// Transport は実際の通信を行う RoundTripper。nil なら標準のものを使う。
	Transport http.RoundTripper
	// Now は現在時刻。nil なら time.Now。トークンの期限判定に使う。
	Now func() time.Time
	// Logger は、重複として扱われた配信などを記録する。nil なら出力しない。
	Logger *slog.Logger
}

// Notifier は LINE の broadcast(全友だちへの一斉配信)で通知を送る。
type Notifier struct {
	client  *http.Client
	baseURL string
	logger  *slog.Logger
}

// NewNotifier は Notifier を作る。アクセストークンは、送信のたびに自動で発行・更新される。
func NewNotifier(cfg Config) *Notifier {
	base := cfg.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	source := &tokenSource{
		// トークンの発行は、認証を付けない別のクライアントで行う(循環を避けるため)
		client:   &http.Client{Transport: base, Timeout: httpTimeout},
		endpoint: baseURL + "/oauth2/v3/token",
		creds:    cfg.Credentials,
		now:      now,
	}
	return &Notifier{
		client:  &http.Client{Transport: &authTransport{base: base, source: source}, Timeout: httpTimeout},
		baseURL: baseURL,
		logger:  logger,
	}
}

type textMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type broadcastRequest struct {
	Messages []textMessage `json:"messages"`
}

// Notify は通知を全友だちへ broadcast する。
// 同じ Key のリクエストは、LINE 側で重複として扱われる(X-Line-Retry-Key)。
// 重複(409)は、すでに配信が受け付けられているため、成功として扱う。
func (n *Notifier) Notify(ctx context.Context, msg notice.Notice) error {
	body, err := json.Marshal(broadcastRequest{Messages: []textMessage{{Type: "text", Text: msg.Text}}})
	if err != nil {
		return fmt.Errorf("リクエストの作成: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.baseURL+"/v2/bot/message/broadcast", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("リクエストの作成: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Line-Retry-Key", msg.Key)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("broadcast の送信: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusConflict:
		// 同じリトライキーの配信が、すでに受け付けられている。二重配信を防ぐための正常な動作だが、
		// 再通知のために通知済みを消した場合は「配信されなかった」ことになるため、気づけるよう記録する
		n.logger.Warn("LINE が重複として受け付けず、今回は配信されなかった(同じリトライキーは最初の受理から約 24 時間有効)",
			slog.String("retry_key", msg.Key),
			slog.String("accepted_request_id", resp.Header.Get("X-Line-Accepted-Request-Id")))
		return nil
	default:
		return fmt.Errorf("broadcast に失敗: status %d: %s", resp.StatusCode, readSnippet(resp.Body))
	}
}
