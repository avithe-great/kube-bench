package cisaudit

import (
	"bytes"
	"net/url"
	"testing"
)

func TestCustomPageSizeAllModes(t *testing.T) {
	for _, mode := range []string{"roles", "serviceaccounts", "managed-wildcards", "managed-pod-tokens"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			fetch := func(path string) ([]byte, error) {
				calls++
				u, err := url.Parse(path)
				if err != nil {
					t.Fatal(err)
				}
				if u.Query().Get("limit") != "7" {
					t.Fatalf("wrong limit: %s", path)
				}
				if u.Query().Get("continue") == "" {
					return list(`[]`, "next"), nil
				}
				if u.Query().Get("continue") != "next" {
					t.Fatal(path)
				}
				return list(`[]`, ""), nil
			}
			var out bytes.Buffer
			if err := RunWithPageSize(mode, fetch, &out, 7); err != nil {
				t.Fatal(err)
			}
			want := 4
			if mode == "managed-pod-tokens" {
				want = 2
			}
			if calls != want {
				t.Fatalf("calls=%d want=%d", calls, want)
			}
		})
	}
}

func TestInvalidPageSizeMakesNoRequests(t *testing.T) {
	for _, size := range []int{0, -1} {
		err := RunWithPageSize("roles", func(string) ([]byte, error) { t.Fatal("unexpected request"); return nil, nil }, &bytes.Buffer{}, size)
		if err == nil {
			t.Fatalf("accepted %d", size)
		}
	}
}
