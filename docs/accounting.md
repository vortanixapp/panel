# Accounting

[English](accounting.md) · [Русский](accounting.ru.md)

The Accounting section of the admin area stores the seller's details and tax
settings, sends fiscal receipts through payment providers, gives clients
closing documents and prepares exports for tax reporting. Documents and
exports follow Russian accounting practice and are generated in Russian.

## Details and taxes

On the Details and taxes tab, fill in:

- legal form and short and full name;
- INN, KPP, and OGRN or OGRNIP;
- registered address and bank details;
- the document signatory.

The INN is validated by its check digits.

| Setting | Used in |
|---|---|
| Tax system | Fiscal receipts, summary hints |
| VAT rate on services | Receipts, acts, services and acts registers |
| Accounting time zone | Document dates and report period boundaries |
| Settlement type in receipts | Settlement type and subject in receipts |
| Receipt line name | The receipt line for a balance top-up |

Self-employed sellers cannot use VAT and issue receipts in the “My Tax” app.

## Fiscal receipts (54-FZ)

Sending a receipt is turned on with a checkbox in the payment provider
settings. YooKassa, T-Kassa, Robokassa and CloudPayments are supported. A
receipt carries the client's email and these settings from the Accounting
section: tax system, VAT rate, settlement type and line name.

- **Full payment.** The subject is “Service”, VAT uses the regular rate.
- **100% prepayment.** The subject is “Service”, VAT uses the calculated rate,
  for example 20/120.
- **Advance.** The subject is “Payment”, VAT uses the calculated rate.

Your accountant decides which settlement type applies to balance top-ups.

Receipt parameters are stored with the payment. A refund through YooKassa uses
them to send a refund receipt. The payments register marks the payments whose
receipt was sent.

## Client documents

On the balance page a client enters payer details: individual, sole
proprietor or company, legal name, INN, KPP, OGRN and address. The same page
provides:

- **A monthly service act.** It is issued after the month ends and keeps a
  permanent number. It lists rent, renewals and plan or resource changes with
  dates, the total, VAT and the amount in words.
- **A reconciliation statement for any period.** It shows the opening
  balance, payments, services, refunds, bonuses, turnover and the closing
  balance, with a conclusion about debt or prepayment held.

The payment receipt page shows:

- seller and payer details;
- the invoice number;
- refunded amount;
- whether a fiscal receipt was sent.

Administrators open the same documents for any client on the Client documents
tab, searching by email, INN, legal name or last name.

## Reports

The Reports tab takes a period and a currency. The period can be:

- a month, quarter, half year, nine months or year;
- custom dates.

The summary shows:

- money received and refunded;
- received minus refunds, with a hint for the selected tax system;
- services rendered and their VAT;
- bonuses;
- client prepayments at the start and end of the period;
- breakdowns by month and payment method.

| Export | Contents |
|---|---|
| Income ledger (KUDiR) | Section I: receipts and refunds by date with quarterly totals |
| Payments register | Date, invoice, payer, INN, method, payment ID, paid and credited amounts, refunds, receipt mark |
| Refunds register | Date, invoice, payer, amount, reason, refund number in the payment system |
| Services register | Date, buyer, service, amount, VAT |
| Turnover balance sheet | Client prepayments at the start and end, payments, bonuses, services, refunds |
| Acts register | Numbers and amounts of acts for closed months of the period |

Exports are CSV files with “;” as the separator and a decimal comma, ready for
Excel. Acts get their numbers on first issue, and repeated exports keep them.

The receipt date in reports is the moment the client paid. Payment providers
transfer funds to the bank account later and minus their fee, so reconcile
receipts with the bank statement and provider reports.

## Access

Viewing reports and documents requires the “Billing (view)” permission.
Changing the details requires “Billing (edit)”.
