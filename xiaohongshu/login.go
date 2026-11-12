package xiaohongshu

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-rod/rod"
	"github.com/pkg/errors"
	xhserrors "github.com/xpzouying/xiaohongshu-mcp/errors"
)

type LoginAction struct {
	page *rod.Page
}

func NewLogin(page *rod.Page) *LoginAction {
	return &LoginAction{page: page}
}

func (a *LoginAction) CheckLoginStatus(ctx context.Context) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := a.page.Context(ctx).Navigate(getBaseURL() + "/explore"); err != nil {
		return false, errors.Wrap(err, "navigate for login status failed")
	}

	time.Sleep(1 * time.Second)

	if _, err := a.waitCurrentUser(ctx); err != nil {
		if errors.Is(err, xhserrors.ErrNotLoggedIn) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CurrentUser 当前登录用户的基础信息。
type CurrentUser struct {
	Nickname string `json:"nickname"`
	UserID   string `json:"userId"`
}

type currentUserState struct {
	CurrentUser
	Guest *bool `json:"guest"`
}

func (a *LoginAction) readCurrentUserState(ctx context.Context) (*currentUserState, error) {
	res, err := a.page.Context(ctx).Eval(`() => {
		const u = window.__INITIAL_STATE__ && window.__INITIAL_STATE__.user;
		const info = u && u.userInfo && u.userInfo.value !== undefined ? u.userInfo.value : (u && u.userInfo);
		return JSON.stringify(info ? {guest: info.guest, nickname: info.nickname, userId: info.userId || info.user_id} : {});
	}`)
	if err != nil {
		return nil, errors.Wrap(err, "read current user state failed")
	}
	var state currentUserState
	if err := json.Unmarshal([]byte(res.Value.String()), &state); err != nil {
		return nil, errors.Wrap(err, "unmarshal current user failed")
	}
	return &state, nil
}

// CurrentUser 等待当前页给出明确用户状态，不导航。
func (a *LoginAction) CurrentUser(ctx context.Context) (*CurrentUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return a.waitCurrentUser(ctx)
}

func (a *LoginAction) waitCurrentUser(ctx context.Context) (*CurrentUser, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		state, err := a.readCurrentUserState(ctx)
		if err != nil {
			return nil, err
		}
		if state.Guest != nil {
			if *state.Guest {
				return nil, xhserrors.ErrNotLoggedIn
			}
			if state.UserID != "" {
				return &state.CurrentUser, nil
			}
		}
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "wait for current user state failed")
		case <-ticker.C:
		}
	}
}

func (a *LoginAction) Login(ctx context.Context) error {
	if err := a.page.Context(ctx).Navigate(loginPageURL()); err != nil {
		return errors.Wrap(err, "navigate for login failed")
	}
	time.Sleep(2 * time.Second)
	if !a.WaitForLogin(ctx) {
		return errors.Wrap(ctx.Err(), "wait for login failed")
	}
	return nil
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	pp := a.page.Context(ctx)
	if err := pp.Navigate(loginPageURL()); err != nil {
		return "", false, errors.Wrap(err, "navigate for qrcode failed")
	}
	time.Sleep(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		state, err := a.readCurrentUserState(ctx)
		if err != nil {
			return "", false, err
		}
		if state.Guest != nil && !*state.Guest && state.UserID != "" {
			return "", true, nil
		}
		exists, el, err := pp.Has(".login-container .qrcode-img")
		if err != nil {
			return "", false, errors.Wrap(err, "find qrcode failed")
		}
		if exists {
			src, err := el.Attribute("src")
			if err != nil {
				return "", false, errors.Wrap(err, "get qrcode src failed")
			}
			if src != nil && *src != "" {
				return *src, false, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", false, errors.Wrap(ctx.Err(), "wait for qrcode failed")
		case <-ticker.C:
		}
	}
}

func (a *LoginAction) WaitForLogin(ctx context.Context) bool {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if _, err := a.CurrentUser(ctx); err == nil {
				return true
			}
		}
	}
}
