//go:build integration

package xiaohongshu

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/xpzouying/xiaohongshu-mcp/browser"
	xhserrors "github.com/xpzouying/xiaohongshu-mcp/errors"
)

func newLoginTestPage(t *testing.T, body string) *rod.Page {
	t.Helper()
	pending := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pending" {
			w.Header().Set("Content-Type", "image/gif")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			select {
			case <-pending:
			case <-r.Context().Done():
			}
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!doctype html><html><body>%s<img src="/pending"></body></html>`, body)
	}))
	t.Cleanup(func() {
		close(pending)
		server.Close()
	})
	t.Setenv("XHS_BASE_URL", server.URL)

	bin := os.Getenv("ROD_BROWSER_BIN")
	if bin == "" {
		var err error
		bin, err = browser.EnsureBrowser()
		if err != nil {
			t.Skipf("浏览器不可用: %v", err)
		}
	}
	l := launcher.New().Bin(bin).Headless(true).NoSandbox(true)
	b := rod.New().ControlURL(l.MustLaunch()).MustConnect()
	t.Cleanup(func() {
		b.MustClose()
		l.Cleanup()
	})
	return b.MustPage("about:blank")
}

func TestCheckLoginStatusPendingLoad(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state string
		want  bool
	}{
		{"authenticated", `{guest:false,userId:"test-user",nickname:"test"}`, true},
		{"value_wrapped", `{value:{guest:false,user_id:"test-user",nickname:"test"}}`, true},
		{"guest", `{guest:true}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := newLoginTestPage(t, `<script>setTimeout(() => {window.__INITIAL_STATE__={user:{userInfo:`+tc.state+`}}}, 1500)</script>`)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got, err := NewLogin(page).CheckLoginStatus(ctx)
			if err != nil || got != tc.want {
				t.Fatalf("CheckLoginStatus() = %v, %v; want %v, nil", got, err, tc.want)
			}
			if page.MustEval(`() => document.readyState`).Str() == "complete" {
				t.Fatal("fixture must keep the load event pending")
			}
		})
	}
}

func TestCheckLoginStatusPreservesDeadline(t *testing.T) {
	page := newLoginTestPage(t, `<script>
		setTimeout(() => {window.__INITIAL_STATE__={user:{userInfo:{guest:false,userId:"test-user"}}}}, 12000);
	</script>`)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	loggedIn, err := NewLogin(page).CheckLoginStatus(ctx)
	if err != nil || !loggedIn {
		t.Fatalf("CheckLoginStatus() must use its full deadline, got %v, %v", loggedIn, err)
	}
	if page.MustEval(`() => document.readyState`).Str() == "complete" {
		t.Fatal("fixture must keep the load event pending")
	}
}

func TestCurrentUserRejectsUnknownState(t *testing.T) {
	for _, body := range []string{
		`<script>window.__INITIAL_STATE__={}</script>`,
		`<script>window.__INITIAL_STATE__={user:{userInfo:{guest:false}}}</script>`,
		`<script>window.__INITIAL_STATE__={user:{userInfo:{userId:"test-user"}}}</script>`,
	} {
		page := newLoginTestPage(t, body)
		page.MustNavigate(getBaseURL() + "/explore")
		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
		_, err := NewLogin(page).CurrentUser(ctx)
		cancel()
		if err == nil || errors.Is(err, xhserrors.ErrNotLoggedIn) {
			t.Fatalf("unknown user state must remain an error, got %v", err)
		}
	}
}

func TestCurrentUserGuestAndReadError(t *testing.T) {
	page := newLoginTestPage(t, `<script>window.__INITIAL_STATE__={user:{userInfo:{guest:true}}}</script>`)
	page.MustNavigate(getBaseURL() + "/explore")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := NewLogin(page).CurrentUser(ctx); !errors.Is(err, xhserrors.ErrNotLoggedIn) {
		t.Fatalf("explicit guest must return ErrNotLoggedIn, got %v", err)
	}
	page.MustClose()
	if _, err := NewLogin(page).CurrentUser(ctx); err == nil || errors.Is(err, xhserrors.ErrNotLoggedIn) {
		t.Fatalf("closed page must return a read error, got %v", err)
	}
}

func TestFetchQrcodeImagePendingLoad(t *testing.T) {
	const qr = "data:image/png;base64,dGVzdA=="
	for _, tc := range []struct {
		name         string
		body         string
		wantImage    string
		wantLoggedIn bool
	}{
		{"qrcode", `<script>window.__INITIAL_STATE__={user:{userInfo:{guest:true}}}</script><div class="login-container"><img class="qrcode-img" src="` + qr + `"></div>`, qr, false},
		{"authenticated", `<script>window.__INITIAL_STATE__={user:{userInfo:{guest:false,userId:"test-user"}}}</script>`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := newLoginTestPage(t, tc.body)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			img, loggedIn, err := NewLogin(page).FetchQrcodeImage(ctx)
			if err != nil || img != tc.wantImage || loggedIn != tc.wantLoggedIn {
				t.Fatalf("FetchQrcodeImage() = %q, %v, %v", img, loggedIn, err)
			}
			if page.MustEval(`() => document.readyState`).Str() == "complete" {
				t.Fatal("fixture must keep the load event pending")
			}
		})
	}
}

func TestWaitForLoginWithoutAvatar(t *testing.T) {
	page := newLoginTestPage(t, `<script>
		window.__INITIAL_STATE__={user:{userInfo:{guest:true}}};
		setTimeout(() => {window.__INITIAL_STATE__.user.userInfo={guest:false,userId:"test-user"}}, 1200);
	</script>`)
	page.MustNavigate(getBaseURL() + "/explore")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if !NewLogin(page).WaitForLogin(ctx) {
		t.Fatal("authenticated state must complete login while load and avatar remain absent")
	}
}
