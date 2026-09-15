# Legal documents, consents and hosting obligations

[English](compliance.md) · [Русский](compliance.ru.md)

This page covers panel features for a game hosting business operating officially in Russia:

- legal documents and the consent log;
- client identification;
- abuse cases and authority requests;
- balance refunds;
- prepayment offset receipts;
- payments from companies;
- personal data subject rights.

The panel provides the tools but does not replace a lawyer. Have a specialist review the document texts, deadlines and procedures.

## Documents and consents

In the admin area open Legal documents. The Documents tab manages four documents:

- terms of service;
- privacy policy;
- personal data processing consent;
- cookie policy.

Insert template fills a draft with the company details from the Accounting section. The markup is simple: a line starting with `## ` is a section heading, a line starting with `- ` is a list item, and an empty line starts a new paragraph.

Every publication creates a new revision, and earlier revisions stay in the history. Published documents are available at `/legal/offer`, `/legal/privacy`, `/legal/consent` and `/legal/cookies`. The site footer links to them and shows the seller's details.

### Client consents

- At sign-up a client accepts the terms of service and the privacy policy, and gives personal data processing consent with a separate checkbox. The account is not created without them.
- A client who signed up through social login, Telegram or WHMCS sees a dialog with the documents on first entry to the panel and cannot continue until accepting them.
- When publishing a new revision, the “Ask clients to accept the new revision” checkbox shows that dialog to every client. Clear it for edits that do not change the terms.
- The Consent log tab records every acceptance and withdrawal: document, revision, date, IP address and browser.

The cookie notice is turned on on the Identification and cookies tab.

## Client identification

Hosting provider rules prohibit serving unidentified customers. “Do not provide services without identification” blocks server rental, trial servers and web hosting until the client is identified. Topping up stays available because it is the identification method.

- **Automatically.** A client is identified after the first successful payment through one of the providers selected in the settings. By default these are YooKassa, T-Kassa, Robokassa, CloudPayments and bank transfer.
- **Manually.** On the user card a staff member marks identification and records the basis, for example a verified document or contract. Identification can be removed there too.

Every change is written to the identification history.

## Abuse cases and authority requests

Abuse cases track requests from Roskomnadzor, courts and law enforcement, copyright complaints and abuse reports. Each case records:

- the source and request number;
- the subject and the target address;
- the server or client;
- the deadline.

The default deadline is one day for authority and copyright requests and three days for others. New cases past their deadline are flagged in the list.

Actions:

- **Notify client** sends a message to the panel notifications and the client's delivery channels.
- **Block server** and **Unblock server** stop and block the server, and notify the client with the reason.
- **Close as resolved** or **Reject** requires a resolution.
- **Add note.**

All actions are stored in the case history with the date and staff member.

## Balance refunds

A client files a refund request on the balance page: the amount and the method, either the original payment method or a bank account with details. Only one pending request per currency is allowed, and the client can cancel it until it is processed.

On the Accounting page, Refunds tab, staff can:

- **Refund through the payment provider.** The amount is spread over the client's latest payments in providers with a refund API; YooKassa receives a refund receipt.
- **Record a bank transfer.** The payment order number is required; the amount is debited from the balance and appears in the income ledger and the refunds register.
- **Reject the request** with a reason the client sees in the notification.

## Prepayment offset receipts

If Accounting uses the Advance or 100% prepayment settlement type for receipts, a second receipt, the prepayment offset, is required once a service is rendered. Every minute the panel matches rent, renewal and plan-change charges against the client's payments with such receipts and sends offset receipts through YooKassa, T-Kassa and CloudPayments.

For Robokassa and other registers the receipts are listed as manual: issue them in the register's dashboard and mark them on the Offset receipts tab, which also shows delivery errors and a retry button. The Offset receipts register is available among the reports.

## Payments from companies

To accept bank transfers, enable the Bank transfer payment method. The client gets an invoice with the seller's and payer's details, the amount in words and a payment purpose that includes the invoice number.

Incoming payments are credited by uploading a bank statement in the 1CClientBankExchange format on the Bank statement tab. The panel picks the incoming payments to the seller's account, matches them to invoices by the number in the payment purpose and the amount, and credits balances after confirmation. Uploading the same statement again never credits a payment twice.

## Personal data

- **Data export.** A client downloads everything stored about them as JSON on the My data tab of the settings. Staff export any user's data from the user card, for example on a request from the data subject or an authority.
- **Account deletion.** A client deletes the account in the settings; staff delete it from the user card. Servers must be deleted and the balance refunded first.
- **What deletion does.** Name, phone, email, sessions, social links, notifications and support requests are removed or anonymized, sign-in becomes impossible and the processing consent is withdrawn. Payments, balance operations, acts and company details are kept for five years for accounting and tax purposes.

## Access

- Legal documents require the “Settings (edit)” permission.
- Abuse cases require “Servers (view)” and “Servers (edit)”.
- Refunds, offset receipts and the bank statement require “Billing (view)” and “Billing (edit)”.
