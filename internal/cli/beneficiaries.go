package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/yashiels/investec/internal/api"
)

func beneficiariesCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "beneficiaries",
		Short: "Beneficiaries and beneficiary categories",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List beneficiaries",
		Args:  cobra.NoArgs,
		RunE: a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
			return c.Beneficiaries(ctx)
		}, nil),
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "categories <profileId>",
		Short: "List beneficiary categories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.BeneficiaryCategories(ctx, args[0])
			}, nil)(cc, args)
		},
	})
	return cmd
}
