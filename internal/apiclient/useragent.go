package apiclient

import (
	"net/http"
	"net/http/cookiejar"
	"runtime/debug"
	"strings"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"
)

// Version 是 CLI 的版本号,默认 "dev"。
//
// 真实版本由 commands 包在 init 时从 commands.BuildVersion(-ldflags 注入)
// 同步过来;commands.BuildVersion 仍是唯一的 ldflags 注入点,这里只是为了
// 让 apiclient 在不反向 import commands(会构成循环)的前提下拿到版本号。
var Version = "dev"

// userAgent 返回规范的 User-Agent 头值,格式为 `talon-sandbox-cli/<version>`。
//
// 后端「来源追踪」按 UA 前缀归因:含 "talon-sandbox-cli" 的请求归类为
// created_from=cli(而非 sdk-<lang>)。版本号优先取 Version(ldflags 注入),
// 回落到 go module 的 build info,最终回落到 "dev",发版后自动更新无需改码。
func userAgent() string {
	return "talon-sandbox-cli/" + resolveVersion()
}

// resolveVersion 解析真实版本号,绝不硬编码版本字符串。
//
// 取值顺序:
//  1. Version(由 commands.BuildVersion 经 -ldflags 注入)
//  2. runtime/debug build info 的主模块版本(go install module@vX 时存在)
//  3. "dev"(本地裸构建兜底)
func resolveVersion() string {
	if v := normalizeVersion(Version); v != "" {
		return v
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := normalizeVersion(bi.Main.Version); v != "" {
			return v
		}
	}
	return "dev"
}

// normalizeVersion 过滤掉无意义的占位版本(空串、go 的本地构建哨兵 "(devel)"、
// 既有默认值 "dev"),其余原样返回。
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	switch v {
	case "", "dev", "(devel)":
		return ""
	}
	return v
}

// userAgentTransport 是一个 http.RoundTripper 装饰器,为每个出站请求注入
// User-Agent 头(请求方未显式设置时)。底层 RoundTripper 为空时用
// http.DefaultTransport。
type userAgentTransport struct {
	ua   string
	base http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	// 浅拷贝请求,避免改动调用方持有的 *http.Request(RoundTripper 契约)。
	r := req.Clone(req.Context())
	if r.Header.Get("User-Agent") == "" {
		r.Header.Set("User-Agent", t.ua)
	}
	return base.RoundTrip(r)
}

// setUserAgent 给直连 HTTP 请求统一打上 CLI 的 User-Agent。
// internal/apiclient 下的 AuthClient/EnvClient/ProcessClient 出口都调它。
func setUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", userAgent())
}

// withUserAgent 返回一个 SDK Option,让 SDK 客户端的所有出站 HTTP 请求带上
// CLI 的 User-Agent。
//
// SDK v0.1.4 没有 WithUserAgent/WithHeader,只能通过 WithHTTPClient 替换整个
// http.Client。SDK 默认客户端自带 cookie jar(cookie 鉴权 + WebSocket 握手依赖
// 它),所以这里必须复刻一个带 jar 的客户端,只把 Transport 换成 UA 装饰器,
// 不能丢掉 jar。
func withUserAgent() talonsandbox.Option {
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{
		Jar: jar,
		Transport: &userAgentTransport{
			ua:   userAgent(),
			base: http.DefaultTransport,
		},
	}
	return talonsandbox.WithHTTPClient(hc)
}
