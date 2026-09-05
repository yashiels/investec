package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/yashiels/investec/internal/api"
	"github.com/yashiels/investec/internal/config"
	"github.com/yashiels/investec/internal/output"
)

// Version is injected at build time via -ldflags.
var Version = "dev"

type globalFlags struct {
	json    bool
	plain   bool
	sandbox bool
	noInput bool
	noColor bool
	quiet   bool
	verbose bool
	timeout time.Duration
}

// app holds shared state resolved once per invocation.
type app struct {
	g   globalFlags
	cfg *config.Config
	cli *api.Client
}

// outMode validates the mutually-exclusive output flags and returns the mode.
func (a *app) outMode() (output.Mode, error) {
	if a.g.json && a.g.plain {
		return output.ModeAuto, &api.CodedError{Code: api.ExitUsage, Msg: "--json and --plain are mutually exclusive"}
	}
	return output.Resolve(a.g.json, a.g.plain), nil
}

// rejectFormatFlags is used by commands whose output cannot be JSON/plain
// (binary documents, completion scripts).
func (a *app) rejectFormatFlags(what string) error {
	if a.g.json || a.g.plain {
		return &api.CodedError{Code: api.ExitUsage, Msg: fmt.Sprintf("--json/--plain not supported for %s", what)}
	}
	return nil
}

// client builds the API client, requiring complete credentials (exit 3).
func (a *app) client() (*api.Client, error) {
	if a.cli != nil {
		return a.cli, nil
	}
	cfg, err := config.Resolve(a.g.sandbox, a.g.noInput)
	if err != nil {
		return nil, &api.CodedError{Code: api.ExitAuth, Msg: err.Error()}
	}
	if !cfg.Creds.Complete() {
		return nil, &api.CodedError{Code: api.ExitAuth,
			Msg: "missing credentials: set " + joinMissing(cfg.Creds.Missing())}
	}
	a.cfg = cfg
	a.cli = api.New(cfg, a.g.timeout)
	return a.cli, nil
}

func (a *app) renderOut() output.Options {
	m, _ := a.outMode()
	return output.Options{Mode: m, NoColor: a.g.noColor || os.Getenv("NO_COLOR") != "", Out: os.Stdout}
}

func joinMissing(m []string) string {
	out := ""
	for i, s := range m {
		if i > 0 {
			out += ", "
		}
		out += s
	}
	return out
}

// Execute is the CLI entry point. It returns a process exit code.
func Execute() int {
	a := &app{}
	root := &cobra.Command{
		Use:           "investec",
		Short:         "Read-only client for the Investec SA Private Bank API",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	pf := root.PersistentFlags()
	pf.BoolVar(&a.g.json, "json", false, "output the full native API envelope as JSON")
	pf.BoolVar(&a.g.plain, "plain", false, "output stable tab-separated lines")
	pf.BoolVar(&a.g.sandbox, "sandbox", false, "target the Investec sandbox environment")
	pf.BoolVar(&a.g.noInput, "no-input", false, "never prompt; fail if credentials are missing")
	pf.BoolVar(&a.g.noColor, "no-color", false, "disable colored output")
	pf.BoolVarP(&a.g.quiet, "quiet", "q", false, "suppress optional diagnostics")
	pf.BoolVarP(&a.g.verbose, "verbose", "v", false, "verbose diagnostics on stderr")
	pf.DurationVar(&a.g.timeout, "timeout", 30*time.Second, "per-request timeout")

	root.AddCommand(
		accountsCmd(a),
		beneficiariesCmd(a),
		profilesCmd(a),
		documentsCmd(a),
		authCmd(a),
	)
	// cobra provides `completion` automatically.

	ctx, stop := signalContext()
	defer stop()
	root.SetContext(ctx)

	if err := root.Execute(); err != nil {
		reportError(err)
		return api.ExitCode(err)
	}
	return api.ExitOK
}

func reportError(err error) {
	fmt.Fprintln(os.Stderr, "investec: "+err.Error())
	if ce, ok := err.(*api.CodedError); ok && ce.Retry != "" {
		fmt.Fprintln(os.Stderr, "  retry after: "+ce.Retry)
	}
}

// run wraps a command function so credential/context plumbing stays uniform.
func (a *app) run(fn func(ctx context.Context, c *api.Client) (*api.Raw, error), table func(io.Writer, *api.Raw) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		if _, err := a.outMode(); err != nil {
			return err
		}
		c, err := a.client()
		if err != nil {
			return err
		}
		raw, err := fn(cmd.Context(), c)
		if err != nil {
			return err
		}
		return output.Envelope(a.renderOut(), raw, table)
	}
}
