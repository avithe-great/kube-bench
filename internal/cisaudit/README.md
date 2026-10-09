# CIS 1.12 paginated audit collectors

Checks 5.1.3 and 5.1.6 now invoke the internal `kube-bench cis-audit`
command. Existing result fields, control IDs, and compliance predicates are
preserved, including the upstream wildcard-array matching and token truth table.
The managed-provider extensions are listed below. Existing Dockerfiles include `internal/`
and `cmd/`, so no extra image dependency is needed.

Each API LIST request asks for 500 items, follows the server continuation
token, and has a 60-second timeout. The client-go transport uses explicit
KUBECONFIG when set, otherwise in-cluster service-account credentials, or the
standard local kubeconfig outside a pod. It preserves CA verification and token
file refresh, rejects redirects, and never defaults to localhost:8080.
An expired continuation token or any API/decode error aborts the check instead
of silently starting a new snapshot. Roles and ClusterRoles are evaluated directly
from their pages. ServiceAccount token settings are stored in a private temporary
directory and reused across Pods; files and staged findings are removed on normal
return. Abrupt process termination can leave temporary files until container cleanup.

Required access: list Roles and ClusterRoles cluster-wide, and list Pods and
ServiceAccounts cluster-wide. Listing ServiceAccounts is an additional requirement
compared with fetching individual ServiceAccounts. Missing accounts, forbidden
requests, malformed responses, and timeouts return nonzero; the existing unscored
CIS evaluator records WARN with an error reason, not PASS. A check warning does
not necessarily cause the entire kube-bench process to exit nonzero.

No cross-run cache is used. Individual paginated collections use the API's list
snapshot semantics; separate collections are not an atomic cluster-wide snapshot.
Run comparisons on a quiescent test cluster. Accounts deleted/created between
collections can produce a visible missing-account error.

Collector memory scales with a page rather than all cluster resources. This is
NOT an OOM guarantee: the parent kube-bench evaluator still stores audit output
and the JSON report in memory. Disk usage scales with accounts and findings.
The API request count is pages(Roles)+pages(ClusterRoles) for 5.1.3 and
pages(ServiceAccounts)+pages(Pods) for 5.1.6, rather than one GET per object/Pod.




## Managed-provider policy collectors

The automated wildcard and Pod token checks now use paginated collection in:

| Benchmark | Wildcards | Pod tokens |
| --- | --- | --- |
| eks-1.7.0, eks-1.8.0 | 4.1.3 | 4.1.6 |
| gke-1.8.0, gke-1.9.0 | 4.1.3 | 4.1.5 |
| aks-1.7 | 4.1.3 | 4.1.6 |

`managed-wildcards` preserves the provider jq predicate, including mixed wildcard
arrays and empty-generator behavior. `managed-pod-tokens` inspects the Pod flag
alone, preserving the provider rule that missing/true is a finding. These modes
do not reuse the different CIS 1.12 compliance predicates. Existing flag names,
scoring, headings and remediation are preserved. All pages must succeed before
emitting flags; collection errors reach the evaluator as failures, not false passes.
These scored checks report FAIL on errors (the CIS 1.12 checks are unscored WARN).

AKS 1.8 and older/manual-only provider checks remain manual; this change does not
convert manual controls into automated checks. Other provider audits are unchanged.
Existing list permissions suffice for these modes. Unlike CIS 1.12, these provider
scripts already used bulk requests: expected benefits are bounded page processing,
removal of kubectl/jq processes and explicit authentication/error handling, not a
reduction from one request per resource. Pagination can increase request count
relative to one unpaginated response. No managed-cluster speedup is claimed yet.

Tests compare wildcard results against the original jq expression (jq required),
check Pod token states, pagination, empty lists and collection failures. Validation
on actual EKS, GKE and AKS clusters remains required before a production rollout.

### Existing provider heading caveat

The GKE 1.8/1.9 and AKS 1.7 pod-token audit headings contain the literal
`automountServiceAccountToken`, which their existing absence tests also match.
Consequently those controls can FAIL even when every Pod explicitly disables
mounting. This pre-existing reporting behavior is preserved here, not silently
changed as part of the performance work. A separate correctness fix should remove
or reword the heading and update the expected report comparison.


## Configuring API page size

The default page size is 500 objects per request. To configure all optimized
audit subprocesses in a full scan, set the environment variable on the scanner
container (the kube-bench init container, not the uploader):

```yaml
env:
  - name: KUBE_BENCH_API_PAGE_SIZE
    value: "250"
```

For a local full scan:

```sh
KUBE_BENCH_API_PAGE_SIZE=250 kube-bench run --benchmark cis-1.12 --config-dir ./cfg
```

For a direct collector invocation:

```sh
kube-bench cis-audit roles --page-size 250
```

Precedence is explicit `cis-audit --page-size`, then `KUBE_BENCH_API_PAGE_SIZE`,
then 500. Values must be positive integers; zero, negative, empty environment
values and malformed values produce an error before API client setup. The flag
belongs to the internal subcommand; use the environment variable for `run`.
The setting applies to all four optimized audit modes, including managed-provider
checks. It controls objects per page, not requests per second or the total number
of resources scanned. Smaller pages trade more API requests for less per-page
memory; very large values can increase memory use. The API may return fewer items
than requested, and the collector still follows all continuation tokens.
