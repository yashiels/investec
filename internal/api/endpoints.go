package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// ValidateDate enforces strict YYYY-MM-DD and calendar validity.
func ValidateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return coded(ExitUsage, "invalid date %q: expected YYYY-MM-DD", s)
	}
	return nil
}

// ValidateRange checks from <= to when both are present.
func ValidateRange(from, to string) error {
	if from == "" || to == "" {
		return nil
	}
	f, _ := time.Parse("2006-01-02", from)
	t, _ := time.Parse("2006-01-02", to)
	if f.After(t) {
		return coded(ExitUsage, "--from (%s) is after --to (%s)", from, to)
	}
	return nil
}

// requireID validates an opaque path identifier: non-empty, no path separators.
func requireID(name, v string) error {
	if strings.TrimSpace(v) == "" {
		return coded(ExitUsage, "%s is required", name)
	}
	if strings.ContainsAny(v, "/?#") {
		return coded(ExitUsage, "%s contains invalid characters", name)
	}
	return nil
}

func (c *Client) Accounts(ctx context.Context) (*Raw, error) {
	return c.get(ctx, "/za/pb/v1/accounts", nil)
}

func (c *Client) Balance(ctx context.Context, accountID string) (*Raw, error) {
	if err := requireID("accountId", accountID); err != nil {
		return nil, err
	}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/accounts/%s/balance", url.PathEscape(accountID)), nil)
}

// TxOptions are the swagger-backed transaction query filters.
type TxOptions struct {
	From            string
	To              string
	TransactionType string
	IncludePending  bool
}

func (c *Client) Transactions(ctx context.Context, accountID string, o TxOptions) (*Raw, error) {
	if err := requireID("accountId", accountID); err != nil {
		return nil, err
	}
	q := url.Values{}
	if o.From != "" {
		q.Set("fromDate", o.From)
	}
	if o.To != "" {
		q.Set("toDate", o.To)
	}
	if o.TransactionType != "" {
		q.Set("transactionType", o.TransactionType)
	}
	if o.IncludePending {
		q.Set("includePending", "true")
	}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/accounts/%s/transactions", url.PathEscape(accountID)), q)
}

func (c *Client) PendingTransactions(ctx context.Context, accountID string) (*Raw, error) {
	if err := requireID("accountId", accountID); err != nil {
		return nil, err
	}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/accounts/%s/pending-transactions", url.PathEscape(accountID)), nil)
}

func (c *Client) Beneficiaries(ctx context.Context) (*Raw, error) {
	return c.get(ctx, "/za/pb/v1/accounts/beneficiaries", nil)
}

func (c *Client) BeneficiaryCategories(ctx context.Context) (*Raw, error) {
	return c.get(ctx, "/za/pb/v1/accounts/beneficiarycategories", nil)
}

func (c *Client) Profiles(ctx context.Context) (*Raw, error) {
	return c.get(ctx, "/za/pb/v1/profiles", nil)
}

func (c *Client) ProfileAccounts(ctx context.Context, profileID string) (*Raw, error) {
	if err := requireID("profileId", profileID); err != nil {
		return nil, err
	}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/profiles/%s/accounts", url.PathEscape(profileID)), nil)
}

func (c *Client) ProfileBeneficiaries(ctx context.Context, profileID, accountID string) (*Raw, error) {
	if err := requireID("profileId", profileID); err != nil {
		return nil, err
	}
	if err := requireID("accountId", accountID); err != nil {
		return nil, err
	}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/profiles/%s/accounts/%s/beneficiaries",
		url.PathEscape(profileID), url.PathEscape(accountID)), nil)
}

func (c *Client) AuthorisationSetup(ctx context.Context, profileID, accountID string) (*Raw, error) {
	if err := requireID("profileId", profileID); err != nil {
		return nil, err
	}
	if err := requireID("accountId", accountID); err != nil {
		return nil, err
	}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/profiles/%s/accounts/%s/authorisationsetupdetails",
		url.PathEscape(profileID), url.PathEscape(accountID)), nil)
}

func (c *Client) Documents(ctx context.Context, accountID, from, to string) (*Raw, error) {
	if err := requireID("accountId", accountID); err != nil {
		return nil, err
	}
	q := url.Values{"fromDate": {from}, "toDate": {to}}
	return c.get(ctx, fmt.Sprintf("/za/pb/v1/accounts/%s/documents", url.PathEscape(accountID)), q)
}

// Document streams a single document's bytes. documentType and documentDate are
// used verbatim (the exact values returned by Documents).
func (c *Client) Document(ctx context.Context, accountID, docType, docDate string) ([]byte, string, error) {
	if err := requireID("accountId", accountID); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(docType) == "" || strings.TrimSpace(docDate) == "" {
		return nil, "", coded(ExitUsage, "documentType and documentDate are required")
	}
	path := fmt.Sprintf("/za/pb/v1/accounts/%s/document/%s/%s",
		url.PathEscape(accountID), url.PathEscape(docType), url.PathEscape(docDate))
	return c.getBinary(ctx, path, "application/pdf")
}
