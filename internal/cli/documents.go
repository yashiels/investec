package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/yashiels/investec/internal/api"
	"github.com/yashiels/investec/internal/output"
)

func documentsCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "documents",
		Short: "List and download account documents",
	}

	var from, to string
	list := &cobra.Command{
		Use:   "list <accountId>",
		Short: "List available documents in a date range",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if from == "" || to == "" {
				return &api.CodedError{Code: api.ExitUsage, Msg: "--from and --to are required"}
			}
			if err := api.ValidateDate(from); err != nil {
				return err
			}
			if err := api.ValidateDate(to); err != nil {
				return err
			}
			if err := api.ValidateRange(from, to); err != nil {
				return err
			}
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.Documents(ctx, args[0], from, to)
			}, output.Documents)(cc, args)
		},
	}
	list.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD, required)")
	list.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD, required)")
	cmd.AddCommand(list)

	var outPath string
	var force bool
	get := &cobra.Command{
		Use:   "get <accountId> <documentType> <documentDate>",
		Short: "Download a document (raw PDF)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cc *cobra.Command, args []string) error {
			if err := a.rejectFormatFlags("documents get"); err != nil {
				return err
			}
			c, err := a.client()
			if err != nil {
				return err
			}
			data, _, err := c.Document(cc.Context(), args[0], args[1], args[2])
			if err != nil {
				return err
			}
			return writeDocument(data, outPath, force)
		},
	}
	get.Flags().StringVarP(&outPath, "output", "o", "", "write to file ('-' for stdout)")
	get.Flags().BoolVar(&force, "force", false, "overwrite an existing output file")
	cmd.AddCommand(get)

	return cmd
}

// writeDocument writes PDF bytes to a file (atomically, 0600, no clobber) or to
// stdout. A bare TTY without -o is refused to avoid dumping binary to a terminal.
func writeDocument(data []byte, outPath string, force bool) error {
	if outPath == "" {
		if output.IsTTY(os.Stdout) {
			return &api.CodedError{Code: api.ExitUsage,
				Msg: "refusing to write binary to a terminal; use -o <file> or -o -"}
		}
		_, err := os.Stdout.Write(data)
		return err
	}
	if outPath == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if !force {
		if _, err := os.Stat(outPath); err == nil {
			return &api.CodedError{Code: api.ExitUsage,
				Msg: fmt.Sprintf("%s already exists; pass --force to overwrite", outPath)}
		}
	}
	tmp, err := os.CreateTemp("", "investec-doc-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, outPath); err != nil {
		// Cross-device rename fallback.
		return os.WriteFile(outPath, data, 0o600)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", outPath, len(data))
	return nil
}
