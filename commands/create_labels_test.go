package commands

import "testing"

func TestParseLabels(t *testing.T) {
	t.Run("nil 输入 → nil", func(t *testing.T) {
		got, err := parseLabels(nil)
		if err != nil || got != nil {
			t.Fatalf("got %v, %v; want nil,nil", got, err)
		}
	})
	t.Run("正常 KV", func(t *testing.T) {
		got, err := parseLabels([]string{"end_user_id=u_8821", "plan=pro"})
		if err != nil {
			t.Fatal(err)
		}
		if got["end_user_id"] != "u_8821" || got["plan"] != "pro" {
			t.Fatalf("bad map: %v", got)
		}
	})
	t.Run("value 含等号(只切首个=)", func(t *testing.T) {
		got, _ := parseLabels([]string{"token=a=b=c"})
		if got["token"] != "a=b=c" {
			t.Fatalf("got %q", got["token"])
		}
	})
	t.Run("空 value 合法", func(t *testing.T) {
		got, err := parseLabels([]string{"k="})
		if err != nil || got["k"] != "" {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("缺等号 → 错误", func(t *testing.T) {
		if _, err := parseLabels([]string{"noequals"}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("空 key → 错误", func(t *testing.T) {
		if _, err := parseLabels([]string{"=v"}); err == nil {
			t.Fatal("expected error")
		}
	})
}
