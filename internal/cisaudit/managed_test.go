package cisaudit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestManagedWildcardParityWithJQ(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Fatal("jq is required for provider predicate parity tests")
	}
	states := [][]string{nil, {}, {"get"}, {"*"}, {"get", "*"}}
	for _, verbs := range states {
		for _, resources := range states {
			for _, groups := range states {
				rules, _ := json.Marshal([]map[string]any{{"verbs": verbs, "resources": resources, "apiGroups": groups}})
				got, err := managedWildcards(rules)
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command("jq", "-c", `.items[] | select(.rules[]? | (.verbs[]? == "*" or .resources[]? == "*" or .apiGroups[]? == "*"))`)
				cmd.Stdin = strings.NewReader(`{"items":[{"rules":` + string(rules) + `}]}`)
				output, err := cmd.Output()
				if err != nil {
					t.Fatal(err)
				}
				if got != (len(output) > 0) {
					t.Fatalf("predicate differs from jq for %s: got %v", rules, got)
				}
			}
		}
	}
}

func TestManagedPaginatedFlags(t *testing.T) {
	for _, mode := range []string{"managed-wildcards", "managed-pod-tokens"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			fetch := func(path string) ([]byte, error) {
				calls++
				if strings.Contains(path, "clusterroles") {
					return list(`[]`, ""), nil
				}
				if !strings.Contains(path, "continue=") {
					return list(`[{"metadata":{"name":"safe"},"rules":[{"verbs":["get"],"resources":["pods"],"apiGroups":[""]}],"spec":{"automountServiceAccountToken":false}}]`, "next"), nil
				}
				return list(`[{"metadata":{"name":"unsafe"},"rules":[{"verbs":["get","*"],"resources":["pods"]}],"spec":{}}]`, ""), nil
			}
			var out bytes.Buffer
			if err := Run(mode, fetch, &out); err != nil {
				t.Fatal(err)
			}
			want := "automountServiceAccountToken\n"
			count := 2
			if mode == "managed-wildcards" {
				want = "wildcards_present\n"
				count = 3
			}
			if out.String() != want || calls != count {
				t.Fatalf("output=%q calls=%d", out.String(), calls)
			}
		})
	}
}

func TestManagedFailuresDoNotPass(t *testing.T) {
	for _, mode := range []string{"managed-wildcards", "managed-pod-tokens"} {
		for _, failure := range []string{"Forbidden", "timeout", "expired continuation"} {
			calls := 0
			var out bytes.Buffer
			err := Run(mode, func(string) ([]byte, error) {
				calls++
				if calls == 1 {
					return list(`[{"metadata":{"name":"unsafe"},"rules":[{"verbs":["*"]}]}]`, "next"), nil
				}
				return nil, fmt.Errorf("%s", failure)
			}, &out)
			if err == nil || out.Len() != 0 {
				t.Fatalf("%s %s: err=%v output=%q", mode, failure, err, out.String())
			}
		}
	}
}

func TestManagedPodTokenStates(t *testing.T) {
	for _, state := range []string{"null", "false", "true"} {
		var out bytes.Buffer
		err := Run("managed-pod-tokens", func(string) ([]byte, error) {
			return list(`[{"metadata":{"name":"p"},"spec":{"automountServiceAccountToken":`+state+`}}]`, ""), nil
		}, &out)
		if err != nil {
			t.Fatal(err)
		}
		if (out.Len() > 0) != (state != "false") {
			t.Fatalf("state=%s output=%q", state, out.String())
		}
	}
}

func TestManagedEmptyCollections(t *testing.T) {
	for _, mode := range []string{"managed-wildcards", "managed-pod-tokens"} {
		var out bytes.Buffer
		if err := Run(mode, func(string) ([]byte, error) { return list(`[]`, ""), nil }, &out); err != nil || out.Len() != 0 {
			t.Fatalf("mode=%s err=%v output=%q", mode, err, out.String())
		}
	}
}
