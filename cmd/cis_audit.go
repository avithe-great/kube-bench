package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/aquasecurity/kube-bench/internal/cisaudit"
	"github.com/spf13/cobra"
)

func init() {
	auditCmd := &cobra.Command{
		Use:           "cis-audit [roles|serviceaccounts|managed-wildcards|managed-pod-tokens]",
		Short:         "Collect Kubernetes policy evidence with paginated API queries",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageSize, err := auditPageSize(cmd)
			if err != nil {
				return err
			}
			fetch, err := cisaudit.APIClient()
			if err != nil {
				return err
			}
			return cisaudit.RunWithPageSize(args[0], fetch, cmd.OutOrStdout(), pageSize)
		},
	}
	auditCmd.Flags().Int("page-size", cisaudit.DefaultPageSize, "Maximum objects per API page (or KUBE_BENCH_API_PAGE_SIZE)")
	RootCmd.AddCommand(auditCmd)
}

// The environment option is inherited by audit subprocesses during a full scan.
// An explicit subcommand flag takes precedence over the environment.
func auditPageSize(cmd *cobra.Command) (int, error) {
	size, err := cmd.Flags().GetInt("page-size")
	if err != nil {
		return 0, err
	}
	if !cmd.Flags().Changed("page-size") {
		if raw, exists := os.LookupEnv("KUBE_BENCH_API_PAGE_SIZE"); exists {
			size, err = strconv.Atoi(raw)
			if err != nil {
				return 0, fmt.Errorf("KUBE_BENCH_API_PAGE_SIZE must be a positive integer")
			}
		}
	}
	if size <= 0 {
		return 0, fmt.Errorf("API page size must be a positive integer")
	}
	return size, nil
}
