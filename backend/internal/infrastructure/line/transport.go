package line

import (
	"io"
	"net/http"
)

// authTransport は、リクエストのたびに有効なアクセストークンを Authorization に付ける。
// 呼び出し側(Notifier)は、トークンの存在を意識しない。
// 401 が返ったら、トークンを破棄して 1 回だけ発行し直して再試行する。
type authTransport struct {
	base   http.RoundTripper
	source *tokenSource
}

// RoundTrip は http.RoundTripper を実装する。
// RoundTripper は受け取ったリクエストを変更してはいけないため、複製して使う。
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.source.get(req.Context())
	if err != nil {
		closeBody(req)
		return nil, err
	}

	resp, err := t.base.RoundTrip(withBearer(req, token))
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	// ボディを再送できないリクエストは、やり直せない
	retryReq, ok := cloneForRetry(req)
	if !ok {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()

	t.source.invalidate(token)
	fresh, err := t.source.get(req.Context())
	if err != nil {
		return nil, err
	}
	return t.base.RoundTrip(withBearer(retryReq, fresh))
}

func withBearer(req *http.Request, token string) *http.Request {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

// cloneForRetry は、ボディを巻き戻した再送用のリクエストを返す。
func cloneForRetry(req *http.Request) (*http.Request, bool) {
	r := req.Clone(req.Context())
	if req.Body == nil || req.Body == http.NoBody {
		return r, true
	}
	if req.GetBody == nil {
		return nil, false
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, false
	}
	r.Body = body
	return r, true
}

// closeBody は、RoundTripper の規約に従い、エラー時にもリクエストのボディを閉じる。
func closeBody(req *http.Request) {
	if req.Body != nil {
		_ = req.Body.Close()
	}
}
