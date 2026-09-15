package xiaohongshu

import (
	"os"
	"strings"
)

// getBaseURL is the web host every page URL is built on. XHS_BASE_URL
// switches it, e.g. https://www.rednote.com for accounts that are not
// registered with a mainland phone number: those are served by rednote.com
// and never get an authenticated web_session on xiaohongshu.com.
func getBaseURL() string {
	if url := os.Getenv("XHS_BASE_URL"); url != "" {
		return strings.TrimRight(url, "/")
	}
	return "https://www.xiaohongshu.com"
}

// loginPageURL is where the QR code is fetched from. xiaohongshu.com pops
// the login modal on /explore; rednote.com does not, but has a standalone
// /login page with the same DOM (.login-container .qrcode-img).
func loginPageURL() string {
	base := getBaseURL()
	if strings.Contains(base, "rednote.com") {
		return base + "/login"
	}
	return base + "/explore"
}

// BaseURL exposes the configured web host to the main package.
func BaseURL() string { return getBaseURL() }
