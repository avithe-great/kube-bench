package cisaudit

import (
	"encoding/json"
	"fmt"
	"io"
)

// managedWildcards mirrors the provider jq predicate, not the CIS 1.12
// exact-array grep. Preserve jq's empty-generator/short-circuit behavior too.
func managedWildcards(raw json.RawMessage) (bool, error) {
	var rules []struct {
		Verbs     []string `json:"verbs"`
		Resources []string `json:"resources"`
		APIGroups []string `json:"apiGroups"`
	}
	if len(raw) == 0 {
		return false, nil
	}
	if err := json.Unmarshal(raw, &rules); err != nil {
		return false, err
	}
	for _, rule := range rules {
		for _, verb := range rule.Verbs {
			if verb == "*" {
				return true, nil
			}
			for _, resource := range rule.Resources {
				if resource == "*" {
					return true, nil
				}
				for _, group := range rule.APIGroups {
					if group == "*" {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// runManaged emits the same presence flags consumed by the provider benchmarks.
// All pages must succeed before writing even a single flag.
func runManaged(mode string, fetch Fetch, out io.Writer, pageSize int) error {
	found := false
	flag := "automountServiceAccountToken"
	if mode == "managed-wildcards" {
		flag = "wildcards_present"
		for _, kind := range []string{"roles", "clusterroles"} {
			if err := pages(fetch, "/apis/rbac.authorization.k8s.io/v1/"+kind, pageSize, func(o object) error {
				match, err := managedWildcards(o.Rules)
				found = found || match
				return err
			}); err != nil {
				return err
			}
		}
	} else {
		if err := pages(fetch, "/api/v1/pods", pageSize, func(o object) error {
			// Provider checks inspect only the Pod setting; do not substitute the
			// different CIS 1.12 Pod/ServiceAccount truth table.
			found = found || o.Spec.Automount == nil || *o.Spec.Automount
			return nil
		}); err != nil {
			return err
		}
	}
	if found {
		_, err := fmt.Fprintln(out, flag)
		return err
	}
	return nil
}
