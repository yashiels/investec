package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/yashiels/investec/internal/api"
)

func profilesCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Profiles and their accounts",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List profiles",
		Args:  cobra.NoArgs,
		RunE: a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
			return c.Profiles(ctx)
		}, nil),
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "accounts <profileId>",
		Short: "List accounts for a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cc *cobra.Command, args []string) error {
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.ProfileAccounts(ctx, args[0])
			}, nil)(cc, args)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "beneficiaries <profileId> <accountId>",
		Short: "List beneficiaries for a profile account",
		Args:  cobra.ExactArgs(2),
		RunE: func(cc *cobra.Command, args []string) error {
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.ProfileBeneficiaries(ctx, args[0], args[1])
			}, nil)(cc, args)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "authorisation-setup <profileId> <accountId>",
		Short: "Get payment authorisation setup details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cc *cobra.Command, args []string) error {
			return a.run(func(ctx context.Context, c *api.Client) (*api.Raw, error) {
				return c.AuthorisationSetup(ctx, args[0], args[1])
			}, nil)(cc, args)
		},
	})

	return cmd
}
