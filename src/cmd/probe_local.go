package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/5uck1ess/devkit/runners"
	"github.com/spf13/cobra"
)

var errProbeFailed = errors.New("probe failed")

func formatHuman(r runners.ProbeResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "endpoint:    %s\n", r.Endpoint)
	fmt.Fprintf(&b, "model:       %s\n", r.Model)
	if !r.Enabled {
		fmt.Fprintln(&b, "enabled:     no — disabled (set DEVKIT_LOCAL_ENABLED=1 to enable)")
		return b.String()
	}
	fmt.Fprintln(&b, "enabled:     yes")

	switch r.Status {
	case runners.ProbeHealthy:
		fmt.Fprintf(&b, "reachable:   yes (HTTP %d, %dms)\n", r.HTTPStatus, r.LatencyMS)
		fmt.Fprintf(&b, "models seen: %s\n", strings.Join(r.ModelsSeen, ", "))
		fmt.Fprintln(&b, "model match: OK (configured model present in /models)")
	case runners.ProbeModelMissing:
		fmt.Fprintf(&b, "reachable:   yes (HTTP %d, %dms)\n", r.HTTPStatus, r.LatencyMS)
		if len(r.ModelsSeen) > 0 {
			fmt.Fprintf(&b, "models seen: %s\n", strings.Join(r.ModelsSeen, ", "))
		} else {
			fmt.Fprintln(&b, "models seen: (none returned)")
		}
		fmt.Fprintln(&b, "model match: MISSING")
		if r.Hint != "" {
			fmt.Fprintf(&b, "hint:        %s\n", r.Hint)
		}
	default:
		if r.HTTPStatus > 0 {
			fmt.Fprintf(&b, "reachable:   NO (HTTP %d in %dms)\n", r.HTTPStatus, r.LatencyMS)
		} else {
			fmt.Fprintf(&b, "reachable:   NO (%dms)\n", r.LatencyMS)
		}
		if r.Hint != "" {
			fmt.Fprintf(&b, "hint:        %s\n", r.Hint)
		}
		if r.ErrorMsg != "" {
			fmt.Fprintf(&b, "body:        %s\n", r.ErrorMsg)
		}
	}
	return b.String()
}

func formatJSON(r runners.ProbeResult) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

var probeLocalJSON bool

var probeLocalCmd = &cobra.Command{
	Use:   "probe-local",
	Short: "Probe the configured local inference endpoint",
	Long: `Probe the OpenAI-compatible endpoint configured via DEVKIT_LOCAL_* env vars.

Reports endpoint, model, reachability, and whether the configured model is
present in the server's /v1/models response. Exit 0 on healthy, 1 otherwise.`,
	// Override root's PersistentPreRunE — this probe doesn't need a git repo or DB.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	SilenceUsage:      true,
	SilenceErrors:     true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := runners.ProbeConfig{
			Enabled:  runners.LocalEnabled(),
			Endpoint: runners.LocalEndpoint(),
			Model:    runners.LocalModel(),
			APIKey:   runners.LocalAPIKey(),
			Timeout:  3 * time.Second,
		}
		result := runners.Probe(cmd.Context(), cfg)

		if probeLocalJSON {
			out, err := formatJSON(result)
			if err != nil {
				return fmt.Errorf("formatting JSON: %w", err)
			}
			fmt.Println(string(out))
		} else {
			fmt.Print(formatHuman(result))
		}

		if result.Status == runners.ProbeDisabled || result.Status == runners.ProbeHealthy {
			return nil
		}
		return errProbeFailed
	},
}

func init() {
	probeLocalCmd.Flags().BoolVar(&probeLocalJSON, "json", false, "emit structured JSON instead of human-readable text")
	rootCmd.AddCommand(probeLocalCmd)
}
