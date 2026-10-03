package handlers

import (
	"fmt"
	"html"
	"strings"

	"github.com/vortanixapp/panel/internal/api/payments"
)

func renderReceiptIntl(d receiptData) string {
	esc := html.EscapeString
	loc := d.Company.Location
	method := payments.Name(d.Provider)
	rows := []string{
		row("Receipt No.", fmt.Sprintf("%d", d.Number)),
		row("Order No.", fmt.Sprintf("%d", d.Invoice)),
		row("Payment date", d.PaidAt.In(loc).Format("2006-01-02 15:04")),
		row("Amount", esc(invoiceMoney(d.Amount, d.Currency))),
		row("Payment method", esc(method)),
		row("Payer", esc(d.Payer.title())),
	}
	if d.Payer.TaxID != "" {
		rows = append(rows, row("Payer tax ID", esc(d.Payer.TaxID)))
	}
	if d.Payer.Email != "" && d.Payer.Email != d.Payer.title() {
		rows = append(rows, row("Email", esc(d.Payer.Email)))
	}
	if d.ExternalID != "" {
		rows = append(rows, row("Payment provider reference", esc(d.ExternalID)))
	}
	if d.Refunded > 0 {
		rows = append(rows, row("Refunded", esc(invoiceMoney(d.Refunded, d.Currency))))
	}

	c := d.Company
	company := []string{}
	for _, item := range []struct{ label, value string }{
		{"Recipient", firstNonEmpty(c.FullName, c.Name, c.AppName)},
		{"Registration no.", c.RegNumber},
		{"Tax ID", firstNonEmpty(c.TaxID, c.INN)},
		{"Address", c.Address},
		{"IBAN", c.IBAN},
		{"SWIFT/BIC", c.SWIFT},
		{"Email", c.Email},
		{"Phone", c.Phone},
		{"Website", c.Site},
	} {
		if item.value != "" {
			company = append(company, row(item.label, esc(item.value)))
		}
	}
	companyBlock := ""
	if len(company) > 0 {
		companyBlock = `<h2>Payee</h2><table>` + strings.Join(company, "") + `</table>`
	}

	note := "This document confirms that the funds were credited to your balance in the control panel. It is a payment confirmation, not a tax invoice: an invoice is issued for each service charge."
	return `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>Payment receipt ` + fmt.Sprint(d.Number) + `</title>
<style>
  body { font-family: system-ui, -apple-system, "Segoe UI", sans-serif; color: #111; margin: 40px auto; max-width: 640px; padding: 0 16px; }
  h1 { font-size: 20px; margin-bottom: 4px; }
  h2 { font-size: 14px; margin: 24px 0 8px; text-transform: uppercase; letter-spacing: .06em; color: #555; }
  table { width: 100%; border-collapse: collapse; }
  td { padding: 8px 0; border-bottom: 1px solid #eee; vertical-align: top; }
  td:first-child { color: #555; width: 45%; }
  td:last-child { text-align: right; font-variant-numeric: tabular-nums; }
  .note { margin-top: 24px; font-size: 12px; color: #777; line-height: 1.5; }
  @media print { body { margin: 0; } .noprint { display: none; } }
  .noprint button { padding: 8px 14px; font-size: 13px; cursor: pointer; }
</style></head>
<body>
  <div class="noprint" style="margin-bottom:16px"><button onclick="window.print()">Print / save as PDF</button></div>
  <h1>Payment receipt</h1>
  <div style="color:#555;font-size:13px">Issued ` + d.IssuedAt.In(loc).Format("2006-01-02 15:04") + `</div>
  <h2>Payment</h2>
  <table>` + strings.Join(rows, "") + `</table>
  ` + companyBlock + `
  <p class="note">` + esc(note) + `</p>
</body></html>`
}
