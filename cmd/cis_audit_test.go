package cmd

import (
	"os"
	"testing"

	"github.com/aquasecurity/kube-bench/internal/cisaudit"
	"github.com/spf13/cobra"
)

func TestAuditPageSize(t *testing.T) {
	cases := []struct {
		name, env, flag string
		unset, bad      bool
		want            int
	}{
		{name: "default", unset: true, want: 500},
		{name: "environment", env: "250", want: 250},
		{name: "flag precedence", env: "250", flag: "100", want: 100},
		{name: "flag overrides bad environment", env: "bad", flag: "100", want: 100},
		{name: "zero", env: "0", bad: true},
		{name: "negative", env: "-1", bad: true},
		{name: "empty", env: "", bad: true},
		{name: "malformed", env: "abc", bad: true},
		{name: "overflow", env: "999999999999999999999999", bad: true},
		{name: "zero flag", env: "500", flag: "0", bad: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("KUBE_BENCH_API_PAGE_SIZE", tc.env)
			if tc.unset {
				os.Unsetenv("KUBE_BENCH_API_PAGE_SIZE")
			}
			cmd := &cobra.Command{}
			cmd.Flags().Int("page-size", cisaudit.DefaultPageSize, "")
			if tc.flag != "" {
				if err := cmd.Flags().Set("page-size", tc.flag); err != nil {
					t.Fatal(err)
				}
			}
			got, err := auditPageSize(cmd)
			if (err != nil) != tc.bad {
				t.Fatalf("value=%d err=%v", got, err)
			}
			if err == nil && got != tc.want {
				t.Fatalf("got=%d want=%d", got, tc.want)
			}
		})
	}
}
