package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/yashiels/investec/internal/api"
	"github.com/yashiels/investec/internal/output"
)

func accountsCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "Accounts, balances, and transactions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List accounts",
		Args:  cobra.NoArgs,
		RunE: a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
			return c.Accounts(ctx)
		}, output.Accounts),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "balance <accountId>",
		Short: "Get an account balance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.Balance(ctx, args[0])
			}, output.Balance)(cc, args)
		},
	})

	var (
		from, to, txType string
		includePending   bool
	)
	txns := &cobra.Command{
		Use:   "transactions <accountId>",
		Short: "List posted transactions (defaults to the last 180 days)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			if from != "" {
				if err := api.ValidateDate(from); err != nil {
					return err
				}
			}
			if to != "" {
				if err := api.ValidateDate(to); err != nil {
					return err
				}
			}
			if err := api.ValidateRange(from, to); err != nil {
				return err
			}
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.Transactions(ctx, args[0], api.TxOptions{
					From: from, To: to, TransactionType: txType, IncludePending: includePending,
				})
			}, output.Transactions)(cc, args)
		},
	}
	txns.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD)")
	txns.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD)")
	txns.Flags().StringVar(&txType, "transaction-type", "", "filter by transaction type")
	txns.Flags().BoolVar(&includePending, "include-pending", false, "include pending transactions")
	cmd.AddCommand(txns)

	cmd.AddCommand(&cobra.Command{
		Use:   "pending <accountId>",
		Short: "List pending transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.PendingTransactions(ctx, args[0])
			}, output.Transactions)(cc, args)
		},
	})

	return cmd
}
