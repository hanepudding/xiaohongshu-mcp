package errors

import "errors"

var ErrNoFeeds = errors.New("没有捕获到 feeds 数据")

// ErrNotLoggedIn: the page reports a guest user. The text is what the MCP
// client reads, so it names the recovery step.
var ErrNotLoggedIn = errors.New("未登录：请调用 get_login_qrcode，把返回的二维码图片展示给用户，用户用小红书 App 扫码并确认后再重试")
var ErrNoFeedDetail = errors.New("没有捕获到 feed 详情数据")
