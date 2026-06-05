package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	talonsandbox "x.xgit.pro/dark/talon-sandbox-sdk-go"
)

// 确认 SDK 路径(createSandbox 归因点)与直连路径的出站请求都带规范 User-Agent
// `talon-sandbox-cli/<version>`,后端据此把请求归类到 created_from=cli。
func TestUserAgent_EndToEnd(t *testing.T) {
	Version = "v9.9.9" // 模拟 ldflags 注入
	defer func() { Version = "dev" }()

	want := "talon-sandbox-cli/v9.9.9"

	var gotSDK, gotDirect string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		gotSDK = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"sb_1","state":"running"}`))
	})
	mux.HandleFunc("/v1/auth/me", func(w http.ResponseWriter, r *http.Request) {
		gotDirect = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tenant_id":"t","role":"owner"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// SDK 路径:经 Opts(含 withUserAgent)的选项调 package 级 Create。
	sdkClientOpts := []talonsandbox.Option{talonsandbox.WithBaseURL(srv.URL), withUserAgent(), talonsandbox.WithAPIKey("k")}
	if _, err := talonsandbox.Create(context.Background(), talonsandbox.Opts{}, sdkClientOpts...); err != nil {
		t.Fatalf("sdk create: %v", err)
	}
	if gotSDK != want {
		t.Fatalf("SDK path UA = %q, want %q", gotSDK, want)
	}

	// 直连路径:AuthClient.Me。
	ac := NewAuthClient(srv.URL, "k")
	if _, err := ac.Me(context.Background()); err != nil {
		t.Fatalf("direct me: %v", err)
	}
	if gotDirect != want {
		t.Fatalf("direct path UA = %q, want %q", gotDirect, want)
	}
}
