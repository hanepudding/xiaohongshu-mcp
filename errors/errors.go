package errors

import "errors"

var ErrNoFeeds = errors.New("没有捕获到 feeds 数据")

// ErrNotLoggedIn: the page reports a guest user. The text is what the MCP
// client reads, so it names the recovery step.
var ErrNotLoggedIn = errors.New("未登录：请调用 get_login_qrcode，把返回的二维码图片展示给用户，用户用小红书 App 扫码并确认后再重试")
var ErrNoFeedDetail = errors.New("没有捕获到 feed 详情数据")

// ErrVerificationRequired 会话被风控判定可疑，请求被重定向到安全验证页。
//
// 这类页面上不会出现 __INITIAL_STATE__，若只是等超时，调用方只能看到
// "context deadline exceeded"，无法判断该做什么。
var ErrVerificationRequired = errors.New("小红书要求安全验证：当前会话被判定可疑，已重定向到安全验证页。" +
	"请在同一账号下重试触发验证的操作，并用手机小红书 App 扫码完成身份验证")

// ErrLoginRequired 未登录或登录态已失效，请求被重定向到登录页。
var ErrLoginRequired = errors.New("未登录或登录态已失效：请求被重定向到登录页，请重新扫码登录")
