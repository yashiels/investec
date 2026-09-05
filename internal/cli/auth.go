package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/yashiels/investec/internal/api"
)

func authCmd(a *app) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Token minting and inspection",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "token",
		Short: "Mint and print a bearer token",
		Args:  cobra.NoArgs,
		RunE: func(cc *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			tok, err := c.Auth().Refresh(cc.Context())
			if err != nil {
				return &api.CodedError{Code: api.ExitAuth, Msg: err.Error()}
			}
			fmt.Fprintln(os.Stdout, tok.AccessToken)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Mint/inspect a token; report endpoint, expiry, and reported scopes",
		Args:  cobra.NoArgs,
		RunE: func(cc *cobra.Command, _ []string) error {
			c, err := a.client()
			if err != nil {
				return err
			}
			tok, err := c.Auth().Get(cc.Context())
			if err != nil {
				return &api.CodedError{Code: api.ExitAuth, Msg: err.Error()}
			}
			if a.g.json {
				status := map[string]any{
					"token_url":       tok.TokenURL,
					"expires_at":      tok.ExpiresAt,
					"scopes":          tok.Scope,
					"token_type":      tok.TokenType,
					"seconds_to_live": int(time.Until(tok.ExpiresAt).Seconds()),
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(status)
			}
			fmt.Printf("token endpoint: %s\n", tok.TokenURL)
			fmt.Printf("expires at:     %s\n", tok.ExpiresAt.Format("2006-01-02 15:04:05 MST"))
			fmt.Printf("reported scopes: %s\n", emptyDash(tok.Scope))
			return nil
		},
	})

	return cmd
}

func emptyDash(s string) string {
	if s == "" {
		return "(none reported)"
	}
	return s
}
