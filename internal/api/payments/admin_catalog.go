package payments

type AdminField struct {
	Label   string            `json:"label"`
	Type    string            `json:"type,omitempty"`
	Default string            `json:"default,omitempty"`
	Options map[string]string `json:"options,omitempty"`
}

type AdminProviderDef struct {
	Key    string                `json:"key"`
	Name   string                `json:"name"`
	Fields map[string]AdminField `json:"fields"`
}

func field(label, typ, def string) AdminField {
	f := AdminField{Label: label, Type: typ}
	if def != "" {
		f.Default = def
	}
	return f
}

func selectField(label, def string, options map[string]string) AdminField {
	return AdminField{Label: label, Type: "select", Default: def, Options: options}
}

func AdminCatalog() []AdminProviderDef {
	vatOptions := map[string]string{
		"1": "Без НДС", "2": "НДС 0%", "3": "НДС 10%", "4": "НДС 20%",
		"5": "НДС 10/110", "6": "НДС 20/120",
	}
	paypalModes := map[string]string{"sandbox": "Sandbox", "live": "Live"}
	hashAlgos := map[string]string{"md5": "MD5", "sha1": "SHA1"}

	return []AdminProviderDef{
		{"perfectmoney", "PerfectMoney", map[string]AdminField{
			"payee_account":       field("Payee Account", "text", ""),
			"alt_passphrase_hash": field("Alternate Passphrase Hash", "text", ""),
			"process_url":         field("Process URL", "text", "https://perfectmoney.com/api/step1.asp"),
		}},
		{"payeer", "Payeer", map[string]AdminField{
			"shop_id": field("Shop ID", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"process_url": field("Process URL", "text", "https://payeer.com/merchant/"),
		}},
		{"webmoney", "WebMoney", map[string]AdminField{
			"purse": field("Purse (WMZ/WMR/etc)", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"process_url": field("Process URL", "text", "https://merchant.webmoney.ru/lmi/payment.asp"),
		}},
		{"webpay", "WebPay.by", map[string]AdminField{
			"store_id": field("Store ID", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"test": field("Test Mode", "checkbox", "0"), "currency": field("Currency", "text", "BYN"),
			"process_url": field("Process URL", "text", "https://payment.webpay.by"),
		}},
		{"paymaster", "PayMaster.ru", map[string]AdminField{
			"merchant_id": field("Merchant ID", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"hash_algo":   selectField("Hash Algorithm", "md5", hashAlgos),
			"process_url": field("Process URL", "text", "https://paymaster.ru/payment/init"),
		}},
		{"advcash", "AdvCash (Volet.com)", map[string]AdminField{
			"account_email": field("Account Email", "email", ""), "sci_name": field("SCI Name", "text", ""),
			"secret_key":  field("Secret Key", "password", ""),
			"process_url": field("Process URL", "text", "https://wallet.advcash.com/sci/"),
		}},
		{"freekassa", "FreeKassa", map[string]AdminField{
			"merchant_id": field("Merchant ID", "text", ""), "secret1": field("Secret 1", "password", ""),
			"secret2": field("Secret 2", "password", ""), "currency": field("Currency", "text", "RUB"),
			"pay_url":      field("Pay URL", "text", "https://pay.fk.money/"),
			"methods_json": field("Methods JSON", "textarea", ""),
		}},
		{"nowpayments", "NowPayments", map[string]AdminField{
			"api_url": field("API URL", "text", "https://api.nowpayments.io"),
			"api_key": field("API Key", "password", ""), "ipn_secret": field("IPN Secret", "password", ""),
			"price_currency": field("Price Currency", "text", "USD"), "pay_currency": field("Pay Currency", "text", "btc"),
		}},
		{"stripe", "Stripe", map[string]AdminField{
			"secret": field("Secret Key", "password", ""), "webhook_secret": field("Webhook Secret", "password", ""),
			"currency": field("Currency", "text", "USD"),
		}},
		{"paypal", "PayPal", map[string]AdminField{
			"client_id": field("Client ID", "text", ""), "client_secret": field("Client Secret", "password", ""),
			"webhook_id": field("Webhook ID", "text", ""), "mode": selectField("Mode", "sandbox", paypalModes),
			"currency": field("Currency", "text", "USD"),
		}},
		{"cloudpayments", "CloudPayments", map[string]AdminField{
			"public_id": field("Public ID", "text", ""), "api_secret": field("API Secret", "password", ""),
			"currency": field("Currency", "text", "RUB"),
		}},
		{"robokassa", "Robokassa", map[string]AdminField{
			"merchant_login": field("Merchant Login", "text", ""), "password1": field("Password 1", "password", ""),
			"password2": field("Password 2", "password", ""), "test": field("Test Mode", "checkbox", "0"),
			"currency": field("Currency", "text", "RUB"),
		}},
		{"yoomoney", "YooMoney", map[string]AdminField{
			"wallet": field("Wallet Number", "text", ""), "notification_secret": field("Notification Secret", "password", ""),
			"access_token": field("Access Token (optional)", "password", ""),
		}},
		{"yookassa", "YooKassa", map[string]AdminField{
			"shop_id": field("Shop ID", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"currency": field("Currency", "text", "RUB"), "vat_code": selectField("VAT Code", "1", vatOptions),
		}},
		{"lava", "Lava", map[string]AdminField{
			"shop_id": field("Shop ID", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"secret_key_2": field("Secret Key 2", "password", ""),
			"api_url":      field("API URL", "text", "https://api.lava.ru"),
		}},
		{"razorpay", "Razorpay", map[string]AdminField{
			"key_id": field("Key ID", "text", ""), "key_secret": field("Key Secret", "password", ""),
			"webhook_secret": field("Webhook Secret", "password", ""),
			"api_url":        field("API URL", "text", "https://api.razorpay.com"), "currency": field("Currency", "text", "INR"),
		}},
		{"paystack", "Paystack", map[string]AdminField{
			"secret_key": field("Secret Key", "password", ""),
			"api_url":    field("API URL", "text", "https://api.paystack.co"), "currency": field("Currency", "text", "NGN"),
		}},
		{"mollie", "Mollie", map[string]AdminField{
			"api_key": field("API Key", "password", ""),
			"api_url": field("API URL", "text", "https://api.mollie.com"), "currency": field("Currency", "text", "EUR"),
		}},
		{"payfast", "PayFast", map[string]AdminField{
			"merchant_id": field("Merchant ID", "text", ""), "merchant_key": field("Merchant Key", "password", ""),
			"passphrase": field("Passphrase", "password", ""), "test": field("Test Mode", "checkbox", "0"),
		}},
		{"flutterwave", "Flutterwave", map[string]AdminField{
			"secret_key": field("Secret Key", "password", ""), "webhook_secret_hash": field("Webhook Secret Hash", "password", ""),
			"api_url": field("API URL", "text", "https://api.flutterwave.com"), "currency": field("Currency", "text", "NGN"),
		}},
		{"enot", "Enot.io", map[string]AdminField{
			"shop_id": field("Shop ID", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"additional_key": field("Additional Key", "password", ""),
			"api_url":        field("API URL", "text", "https://enot.io"),
		}},
		{"tkassa", "T-Kassa (Tinkoff)", map[string]AdminField{
			"terminal_key": field("Terminal Key", "text", ""), "password": field("Password", "password", ""),
			"api_url": field("API URL", "text", "https://securepay.tinkoff.ru/v2"),
		}},
		{"unitpay", "Unitpay", map[string]AdminField{
			"public_key": field("Public Key", "text", ""), "secret_key": field("Secret Key", "password", ""),
			"api_url": field("API URL", "text", "https://unitpay.money/api"),
		}},
		{"cryptocloud", "CryptoCloud", map[string]AdminField{
			"shop_id": field("Shop ID", "text", ""), "api_key": field("API Key", "password", ""),
			"api_url": field("API URL", "text", "https://api.cryptocloud.plus"),
		}},
		{"coinbase", "Coinbase Commerce", map[string]AdminField{
			"api_key": field("API Key", "password", ""), "webhook_secret": field("Webhook Secret", "password", ""),
			"api_url":     field("API URL", "text", "https://api.commerce.coinbase.com"),
			"api_version": field("API Version", "text", "2018-03-22"),
		}},
		{"paddle", "Paddle", map[string]AdminField{
			"vendor_id": field("Vendor ID", "text", ""), "vendor_auth_code": field("Vendor Auth Code", "password", ""),
			"public_key": field("Webhook Public Key", "textarea", ""), "sandbox": field("Sandbox Mode", "checkbox", "0"),
		}},
		{"cryptocom", "Crypto.com Pay", map[string]AdminField{
			"api_key": field("API Key", "password", ""), "secret_key": field("Secret Key (for webhooks)", "password", ""),
			"api_url": field("API URL", "text", "https://pay.crypto.com"),
		}},
		{"bank", "Bank Transfer (Manual)", map[string]AdminField{
			"bank_name": field("Bank Name", "text", ""), "account_number": field("Account Number", "text", ""),
			"account_holder": field("Account Holder", "text", ""), "swift_bic": field("SWIFT/BIC", "text", ""),
			"instructions": field("Instructions", "textarea", ""),
		}},
		{"square", "Square", map[string]AdminField{
			"access_token": field("Access Token", "password", ""), "location_id": field("Location ID", "text", ""),
			"webhook_signature_key":    field("Webhook Signature Key", "password", ""),
			"webhook_notification_url": field("Webhook Notification URL", "text", ""),
			"api_url":                  field("API URL", "text", "https://connect.squareup.com"),
		}},
		{"authorizenet", "Authorize.Net", map[string]AdminField{
			"login_id": field("API Login ID", "text", ""), "transaction_key": field("Transaction Key", "password", ""),
			"signature_key": field("Signature Key (for webhooks)", "password", ""),
			"api_url":       field("API URL", "text", "https://api2.authorize.net/xml/v1/request.api"),
			"hosted_url":    field("Hosted Payment URL", "text", "https://accept.authorize.net/payment/payment"),
		}},
	}
}

func AdminCatalogMap() map[string]AdminProviderDef {
	out := make(map[string]AdminProviderDef, len(AdminCatalog()))
	for _, p := range AdminCatalog() {
		out[p.Key] = p
	}
	return out
}
