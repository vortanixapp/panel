package handlers

import (
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/regions"
)

var legalTitlesIntl = map[string]string{
	"offer":   "Terms of Service",
	"privacy": "Privacy Policy",
	"consent": "Consent to Data Processing",
	"cookies": "Cookie Notice",
}

var legalRightsBlock = map[string]string{
	regions.EU: `## Your rights under the GDPR

You have the right to access your personal data, to have it corrected or erased, to restrict or object to processing, to data portability, and to withdraw consent at any time without affecting the lawfulness of processing before withdrawal. You may also lodge a complaint with the supervisory authority in the EU Member State where you live or work. To exercise your rights, write to {email}.

Our legal bases for processing are: performance of the contract with you (providing the services and billing), compliance with legal obligations (tax and accounting records), our legitimate interests (security, fraud prevention, service improvement) and, where required, your consent.`,
	regions.Europe: `## Your rights

You have the right to access your personal data, to have it corrected or erased, to restrict or object to processing, to data portability, and to withdraw consent. Where the UK GDPR or equivalent national law applies, you may complain to your national data protection authority (in the UK, the Information Commissioner's Office). To exercise your rights, write to {email}.

Our legal bases for processing are: performance of the contract with you, compliance with legal obligations, our legitimate interests (security, fraud prevention, service improvement) and, where required, your consent.`,
	regions.US: `## Your privacy rights

Depending on the state where you live (for example California under the CCPA/CPRA), you may have the right to know what personal information we collect and why, to request access to or deletion of it, to correct inaccurate information, and to opt out of the sale or sharing of personal information. We do not sell personal information. We will not discriminate against you for exercising these rights. To make a request, write to {email}; we may need to verify your identity first.`,
}

var legalGoverning = map[string]string{
	regions.EU:     "the law of the country where the Seller is registered, without depriving consumers of the mandatory consumer-protection rules of their country of residence",
	regions.Europe: "the law of the country where the Seller is registered, without depriving consumers of the mandatory consumer-protection rules of their country of residence",
	regions.US:     "the laws of the jurisdiction where the Seller is registered, without regard to conflict-of-law rules",
}

var legalTemplatesIntl = map[string]string{
	"offer": `Effective date: {date}

## 1. Agreement

These Terms of Service ("Terms") are an agreement between {seller} ("Seller", "we") and you ("Customer"). By registering in the control panel at {site} or by paying for services you accept these Terms.

## 2. Services

We provide game server hosting and related services through the control panel. The scope, resources and price of each service are shown when you order it.

## 3. Account and balance

You are responsible for the activity on your account and for keeping your credentials safe. Payments are credited to a prepaid balance in the currency of the payment; services are paid from the balance. The balance is not a bank deposit and does not earn interest.

## 4. Prices and taxes

Prices shown in the panel include the taxes applicable to the Customer's country (VAT, sales tax or similar), which are shown on the invoice. If you are a business customer in the European Union and provide a valid VAT identification number, the reverse-charge mechanism may apply and VAT is accounted for by you.

## 5. Acceptable use

You must not use the services for unlawful activity, attacks on other systems, spam, or to infringe the rights of others. We may suspend a service that violates these Terms or puts the infrastructure at risk.

## 6. Availability and backups

We aim for high availability but do not guarantee uninterrupted operation. You are responsible for keeping backups of your data.

## 7. Cancellation and refunds

You may stop using a service at any time. Unused prepaid balance can be refunded on request to the original payment method where possible. Consumers in the EU and UK keep the statutory rights that apply to them; where you ask for immediate delivery of a digital service, you acknowledge that the right of withdrawal may be lost once the service has been fully provided.

## 8. Termination

We may terminate or suspend your account for a material breach of these Terms or non-payment, with notice where reasonably possible.

## 9. Liability

To the extent permitted by law, our liability is limited to the amount you paid for the affected service in the three months before the event. Nothing in these Terms limits liability that cannot be limited by law.

## 10. Governing law

These Terms are governed by {governing}.

## 11. Seller details

{requisites}

## 12. Contact

{email}`,

	"privacy": `Effective date: {date}

## 1. Who we are

{seller} ("we") is the controller of personal data processed through the control panel at {site}. Contact: {email}. Address: {address}.

## 2. Data we process

- account data: name, email address, country, address, phone number if provided;
- billing data: payment records, invoices, tax identification number for business customers;
- technical data: IP address, device and browser information, session and security logs;
- service data: the configuration and content of the servers you run, as needed to provide the service.

## 3. Why we process it

To provide and bill for the services, to keep the platform secure, to comply with tax and accounting law, to communicate with you about your account, and to improve the service.

## 4. Recipients

Payment providers, infrastructure and hosting providers, email delivery providers, and authorities where required by law. We do not sell your personal data.

## 5. International transfers

Data may be processed in countries outside your own. Where required, we rely on appropriate safeguards such as standard contractual clauses.

## 6. Retention

We keep data while your account is active and afterwards for as long as required by tax, accounting and other legal obligations or to protect against legal claims.

## 7. Security

We apply technical and organisational measures to protect personal data, including access control and encryption of sensitive values.

{rights}

## 8. Cookies

See the Cookie Notice on {site}.

## 9. Changes

We may update this policy by publishing a new version; material changes are notified when you sign in.`,

	"consent": `I consent to {seller} processing my personal data (name, email address, country, address, payment and technical data) for the purposes of providing the services, billing, security and legal compliance, as described in the Privacy Policy at {site}. I can withdraw this consent at any time by writing to {email}; withdrawal does not affect the lawfulness of processing carried out before it.`,

	"cookies": `Effective date: {date}

## What are cookies

Cookies are small pieces of data a website stores in your browser. The site also uses the browser's local storage.

## What we store

- authentication keys, so you stay signed in;
- interface preferences such as language and theme;
- a note that you have seen this notice.

These are strictly necessary for the site to work. We do not use advertising or cross-site tracking cookies.

## How to control cookies

You can restrict or delete cookies in your browser settings. Without the necessary cookies you will not be able to sign in.

## Contact

{email}`,
}

func legalTemplateIntl(kind, region string, p accountingProfile, now time.Time) (string, string) {
	seller := firstNonEmpty(p.FullName, p.Name, p.AppName)
	requisites := []string{"- " + seller}
	for _, item := range []struct{ label, value string }{
		{"Registration no.", p.RegNumber},
		{"Tax ID", firstNonEmpty(p.TaxID, p.INN)},
		{"Address", p.Address},
		{"Email", p.Email},
		{"Phone", p.Phone},
	} {
		if item.value != "" {
			requisites = append(requisites, "- "+item.label+": "+item.value)
		}
	}
	replacer := strings.NewReplacer(
		"{rights}", legalRightsBlock[region],
		"{governing}", legalGoverning[region],
		"{seller}", seller,
		"{address}", firstNonEmpty(p.Address, "see the details on the site"),
		"{email}", firstNonEmpty(p.Email, "the email address shown on the site"),
		"{site}", firstNonEmpty(p.Site, "the control panel site"),
		"{date}", now.Format("2 January 2006"),
		"{requisites}", strings.Join(requisites, "\n"),
	)
	body := replacer.Replace(legalTemplatesIntl[kind])
	body = strings.NewReplacer("{email}", firstNonEmpty(p.Email, "the email address shown on the site")).Replace(body)
	return legalTitlesIntl[kind], body
}
