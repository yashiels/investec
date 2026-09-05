package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/yashiels/investec/internal/api"
)

// tw builds a tab writer for aligned, greppable columns.
func tw(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
}

// Accounts renders the accounts list.
func Accounts(w io.Writer, r *api.Raw) error {
	var accounts []struct {
		AccountID     string `json:"accountId"`
		AccountNumber string `json:"accountNumber"`
		AccountName   string `json:"accountName"`
		ProductName   string `json:"productName"`
	}
	if err := decodeList(r, "accounts", &accounts); err != nil {
		return writeJSON(w, r.Full)
	}
	if len(accounts) == 0 {
		fmt.Fprintln(w, "no accounts")
		return nil
	}
	t := tw(w)
	fmt.Fprintln(t, "ACCOUNT ID\tNUMBER\tNAME\tPRODUCT")
	for _, a := range accounts {
		fmt.Fprintf(t, "%s\t%s\t%s\t%s\n", a.AccountID, a.AccountNumber, a.AccountName, a.ProductName)
	}
	return t.Flush()
}

// Balance renders a single account balance.
func Balance(w io.Writer, r *api.Raw) error {
	var b struct {
		AccountID        string  `json:"accountId"`
		CurrentBalance   float64 `json:"currentBalance"`
		AvailableBalance float64 `json:"availableBalance"`
		Currency         string  `json:"currency"`
	}
	if len(r.Data) == 0 || json.Unmarshal(r.Data, &b) != nil {
		return writeJSON(w, r.Full)
	}
	t := tw(w)
	fmt.Fprintf(t, "Account\t%s\n", b.AccountID)
	fmt.Fprintf(t, "Current\t%s %.2f\n", b.Currency, b.CurrentBalance)
	fmt.Fprintf(t, "Available\t%s %.2f\n", b.Currency, b.AvailableBalance)
	return t.Flush()
}

// Transactions renders posted or pending transactions.
func Transactions(w io.Writer, r *api.Raw) error {
	var txns []struct {
		TransactionDate string  `json:"transactionDate"`
		PostingDate     string  `json:"postingDate"`
		Description     string  `json:"description"`
		Amount          float64 `json:"amount"`
		Type            string  `json:"type"`
		Status          string  `json:"status"`
	}
	if err := decodeList(r, "transactions", &txns); err != nil {
		return writeJSON(w, r.Full)
	}
	if len(txns) == 0 {
		fmt.Fprintln(w, "no transactions")
		return nil
	}
	t := tw(w)
	fmt.Fprintln(t, "DATE\tDESCRIPTION\tAMOUNT\tTYPE\tSTATUS")
	for _, x := range txns {
		date := x.TransactionDate
		if date == "" {
			date = x.PostingDate
		}
		fmt.Fprintf(t, "%s\t%s\t%.2f\t%s\t%s\n", date, x.Description, x.Amount, x.Type, x.Status)
	}
	return t.Flush()
}

// Documents renders the available-documents list.
func Documents(w io.Writer, r *api.Raw) error {
	var docs []map[string]any
	if err := decodeList(r, "documents", &docs); err != nil {
		return writeJSON(w, r.Full)
	}
	if len(docs) == 0 {
		fmt.Fprintln(w, "no documents")
		return nil
	}
	t := tw(w)
	fmt.Fprintln(t, "TYPE\tDATE\tDESCRIPTION")
	for _, d := range docs {
		fmt.Fprintf(t, "%v\t%v\t%v\n", pick(d, "documentType", "type"),
			pick(d, "documentDate", "date"), pick(d, "description", "documentDescription"))
	}
	return t.Flush()
}

// decodeList handles both `data: [...]` and `data: {"<key>": [...]}` shapes.
func decodeList(r *api.Raw, key string, out any) error {
	if len(r.Data) == 0 {
		return fmt.Errorf("no data")
	}
	if json.Unmarshal(r.Data, out) == nil {
		return nil
	}
	var wrapper map[string]json.RawMessage
	if err := json.Unmarshal(r.Data, &wrapper); err != nil {
		return err
	}
	if inner, ok := wrapper[key]; ok {
		return json.Unmarshal(inner, out)
	}
	return fmt.Errorf("unrecognized shape")
}

func pick(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return ""
}
