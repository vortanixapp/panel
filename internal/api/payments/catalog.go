package payments

import "strings"

type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type AdminField struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Default  string   `json:"default,omitempty"`
	Options  []Option `json:"options,omitempty"`
	Required bool     `json:"required,omitempty"`
}

type AdminProviderDef struct {
	Key           string
	Name          string
	Fields        []AdminField
	FixedCurrency string
	CurrencyField string
	CurrencyFrom  func(cfg map[string]any) string
	Manual        bool
}

func textField(key, label, def string) AdminField {
	return AdminField{Key: key, Label: label, Type: "text", Default: def}
}

func secretField(key, label string) AdminField {
	return AdminField{Key: key, Label: label, Type: "password"}
}

func checkboxField(key, label string) AdminField {
	return AdminField{Key: key, Label: label, Type: "checkbox", Default: "0"}
}

func textareaField(key, label string) AdminField {
	return AdminField{Key: key, Label: label, Type: "textarea"}
}

func selectField(key, label, def string, options ...Option) AdminField {
	return AdminField{Key: key, Label: label, Type: "select", Default: def, Options: options}
}

func required(f AdminField) AdminField {
	f.Required = true
	return f
}

const receiptFieldLabel = "Передавать чек (54-ФЗ): система налогообложения, НДС и признак расчёта — в разделе «Бухгалтерия»"

func buildCatalog() []AdminProviderDef {
	return []AdminProviderDef{
		{
			Key: "yookassa", Name: "ЮKassa", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("shop_id", "Shop ID", "")),
				required(secretField("secret_key", "Secret Key")),
				textField("currency", "Валюта", "RUB"),
				checkboxField("receipt", receiptFieldLabel),
			},
		},
		{
			Key: "yoomoney", Name: "ЮMoney", FixedCurrency: "RUB",
			Fields: []AdminField{
				required(textField("wallet", "Номер кошелька", "")),
				required(secretField("notification_secret", "Секрет для уведомлений")),
				selectField("payment_type", "Способ оплаты", "AC", Option{"AC", "Банковская карта"}, Option{"PC", "Кошелёк ЮMoney"}),
			},
		},
		{
			Key: "tkassa", Name: "Т-Касса", FixedCurrency: "RUB",
			Fields: []AdminField{
				required(textField("terminal_key", "Terminal Key", "")),
				required(secretField("password", "Пароль терминала")),
				textField("api_url", "API URL", "https://securepay.tinkoff.ru/v2"),
				checkboxField("receipt", receiptFieldLabel),
			},
		},
		{
			Key: "robokassa", Name: "Robokassa", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("merchant_login", "Merchant Login", "")),
				required(secretField("password1", "Пароль №1")),
				required(secretField("password2", "Пароль №2")),
				checkboxField("test", "Тестовый режим (нужны тестовые пароли)"),
				textField("currency", "Валюта", "RUB"),
				checkboxField("receipt", receiptFieldLabel),
			},
		},
		{
			Key: "freekassa", Name: "FreeKassa", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("merchant_id", "ID магазина", "")),
				required(secretField("secret1", "Секретное слово 1")),
				required(secretField("secret2", "Секретное слово 2")),
				textField("currency", "Валюта", "RUB"),
				textField("pay_url", "Адрес платёжной формы", "https://pay.fk.money/"),
				textareaField("methods_json", "Способы оплаты (JSON: [{\"id\":4,\"name\":\"VISA\"}])"),
			},
		},
		{
			Key: "cloudpayments", Name: "CloudPayments", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("public_id", "Public ID", "")),
				required(secretField("api_secret", "API Secret")),
				textField("currency", "Валюта", "RUB"),
				checkboxField("receipt", receiptFieldLabel),
			},
		},
		{
			Key: "unitpay", Name: "Unitpay", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("public_key", "Публичный ключ", "")),
				required(secretField("secret_key", "Секретный ключ")),
				textField("currency", "Валюта", "RUB"),
				textField("api_url", "Адрес Unitpay", "https://unitpay.money/api"),
			},
		},
		{
			Key: "lava", Name: "Lava", FixedCurrency: "RUB",
			Fields: []AdminField{
				required(textField("shop_id", "Shop ID", "")),
				required(secretField("secret_key", "Секретный ключ")),
				secretField("secret_key_2", "Дополнительный ключ"),
				textField("api_url", "API URL", "https://api.lava.ru"),
			},
		},
		{
			Key: "enot", Name: "Enot.io", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("shop_id", "Shop ID", "")),
				required(secretField("secret_key", "Секретный ключ")),
				secretField("additional_key", "Дополнительный ключ"),
				textField("currency", "Валюта", "RUB"),
				textField("api_url", "API URL", "https://api.enot.io"),
			},
		},
		{
			Key: "paymaster", Name: "PayMaster", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("merchant_id", "Merchant ID", "")),
				required(secretField("secret_key", "Секретный ключ")),
				selectField("hash_algo", "Алгоритм подписи", "md5", Option{"md5", "MD5"}, Option{"sha1", "SHA1"}, Option{"sha256", "SHA256"}),
				textField("currency", "Валюта", "RUB"),
				textField("process_url", "Адрес оплаты", "https://paymaster.ru/payment/init"),
			},
		},
		{
			Key: "payeer", Name: "Payeer", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("shop_id", "ID магазина", "")),
				required(secretField("secret_key", "Секретный ключ")),
				textField("currency", "Валюта (USD, EUR, RUB)", "RUB"),
				textField("process_url", "Адрес оплаты", "https://payeer.com/merchant/"),
			},
		},
		{
			Key: "webmoney", Name: "WebMoney", CurrencyFrom: webmoneyCurrency,
			Fields: []AdminField{
				required(textField("purse", "Кошелёк (Z, E или P)", "")),
				required(secretField("secret_key", "Secret Key")),
				textField("process_url", "Адрес оплаты", "https://merchant.webmoney.ru/lmi/payment_utf.asp"),
			},
		},
		{
			Key: "webpay", Name: "WebPay.by", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("store_id", "Store ID", "")),
				required(secretField("secret_key", "Secret Key")),
				checkboxField("test", "Тестовый режим"),
				textField("currency", "Валюта", "BYN"),
				textField("process_url", "Адрес оплаты", "https://payment.webpay.by"),
			},
		},
		{
			Key: "advcash", Name: "AdvCash (Volet)", CurrencyField: "currency",
			Fields: []AdminField{
				required(AdminField{Key: "account_email", Label: "Email аккаунта", Type: "email"}),
				required(textField("sci_name", "Имя SCI", "")),
				required(secretField("secret_key", "Пароль SCI")),
				textField("currency", "Валюта", "USD"),
				textField("process_url", "Адрес SCI", "https://wallet.advcash.com/sci/"),
			},
		},
		{
			Key: "perfectmoney", Name: "Perfect Money", CurrencyFrom: perfectMoneyCurrency,
			Fields: []AdminField{
				required(textField("payee_account", "Счёт получателя (U… или E…)", "")),
				textField("payee_name", "Название магазина", ""),
				required(secretField("alt_passphrase_hash", "Альтернативная кодовая фраза")),
				textField("process_url", "Адрес SCI", "https://perfectmoney.com/api/step1.asp"),
			},
		},
		{
			Key: "stripe", Name: "Stripe", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("secret", "Secret Key")),
				required(secretField("webhook_secret", "Webhook Signing Secret")),
				textField("currency", "Валюта", "USD"),
			},
		},
		{
			Key: "paypal", Name: "PayPal", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("client_id", "Client ID", "")),
				required(secretField("client_secret", "Client Secret")),
				required(textField("webhook_id", "Webhook ID", "")),
				selectField("mode", "Режим", "sandbox", Option{"sandbox", "Sandbox"}, Option{"live", "Live"}),
				textField("currency", "Валюта", "USD"),
			},
		},
		{
			Key: "mollie", Name: "Mollie", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("api_key", "API Key")),
				textField("currency", "Валюта", "EUR"),
				textField("api_url", "API URL", "https://api.mollie.com"),
			},
		},
		{
			Key: "paystack", Name: "Paystack", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("secret_key", "Secret Key")),
				textField("currency", "Валюта", "NGN"),
				textField("api_url", "API URL", "https://api.paystack.co"),
			},
		},
		{
			Key: "flutterwave", Name: "Flutterwave", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("secret_key", "Secret Key")),
				required(secretField("webhook_secret_hash", "Secret Hash вебхука")),
				textField("currency", "Валюта", "NGN"),
				textField("api_url", "API URL", "https://api.flutterwave.com"),
			},
		},
		{
			Key: "razorpay", Name: "Razorpay", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("key_id", "Key ID", "")),
				required(secretField("key_secret", "Key Secret")),
				required(secretField("webhook_secret", "Webhook Secret")),
				textField("currency", "Валюта", "INR"),
				textField("api_url", "API URL", "https://api.razorpay.com"),
			},
		},
		{
			Key: "payfast", Name: "PayFast", FixedCurrency: "ZAR",
			Fields: []AdminField{
				required(textField("merchant_id", "Merchant ID", "")),
				required(secretField("merchant_key", "Merchant Key")),
				secretField("passphrase", "Passphrase"),
				checkboxField("test", "Sandbox"),
			},
		},
		{
			Key: "square", Name: "Square", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("access_token", "Access Token")),
				required(textField("location_id", "Location ID", "")),
				required(secretField("webhook_signature_key", "Signature Key вебхука")),
				textField("webhook_notification_url", "URL вебхука, как в кабинете Square (пусто — адрес панели)", ""),
				textField("currency", "Валюта локации", "USD"),
				textField("api_url", "API URL", "https://connect.squareup.com"),
			},
		},
		{
			Key: "authorizenet", Name: "Authorize.Net", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("login_id", "API Login ID", "")),
				required(secretField("transaction_key", "Transaction Key")),
				required(secretField("signature_key", "Signature Key")),
				textField("currency", "Валюта аккаунта", "USD"),
				textField("api_url", "API URL", "https://api2.authorize.net/xml/v1/request.api"),
				textField("hosted_url", "Адрес платёжной страницы", "https://accept.authorize.net/payment/payment"),
			},
		},
		{
			Key: "paddle", Name: "Paddle Classic", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("vendor_id", "Vendor ID", "")),
				required(secretField("vendor_auth_code", "Vendor Auth Code")),
				required(textareaField("public_key", "Публичный ключ для вебхуков")),
				checkboxField("sandbox", "Sandbox"),
				textField("currency", "Валюта", "USD"),
			},
		},
		{
			Key: "nowpayments", Name: "NOWPayments", CurrencyField: "price_currency",
			Fields: []AdminField{
				required(secretField("api_key", "API Key")),
				required(secretField("ipn_secret", "IPN Secret")),
				textField("price_currency", "Валюта счёта", "USD"),
				textField("pay_currency", "Криптовалюта оплаты (пусто — на выбор клиента)", ""),
				textField("api_url", "API URL", "https://api.nowpayments.io"),
			},
		},
		{
			Key: "cryptocloud", Name: "CryptoCloud", CurrencyField: "currency",
			Fields: []AdminField{
				required(textField("shop_id", "Shop ID", "")),
				required(secretField("api_key", "API Key")),
				textField("currency", "Валюта счёта", "USD"),
				textField("api_url", "API URL", "https://api.trybit.com"),
			},
		},
		{
			Key: "coinbase", Name: "Coinbase Commerce", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("api_key", "API Key")),
				required(secretField("webhook_secret", "Webhook Shared Secret")),
				textField("currency", "Валюта счёта", "USD"),
				textField("api_url", "API URL", "https://api.commerce.coinbase.com"),
				textField("api_version", "Версия API", "2018-03-22"),
			},
		},
		{
			Key: "cryptocom", Name: "Crypto.com Pay", CurrencyField: "currency",
			Fields: []AdminField{
				required(secretField("api_key", "Secret Key (sk_…)")),
				required(secretField("secret_key", "Signature Secret вебхука")),
				textField("currency", "Валюта счёта", "USD"),
				textField("api_url", "API URL", "https://pay.crypto.com"),
			},
		},
		{
			Key: "bank", Name: "Банковский перевод", Manual: true,
			Fields: []AdminField{
				textField("bank_name", "Банк", ""),
				required(textField("account_holder", "Получатель", "")),
				required(textField("account_number", "Номер счёта / IBAN", "")),
				textField("swift_bic", "SWIFT / БИК", ""),
				textareaField("instructions", "Инструкция для клиента"),
			},
		},
	}
}

var (
	catalog      = buildCatalog()
	catalogIndex = indexCatalog(catalog)
)

func indexCatalog(defs []AdminProviderDef) map[string]AdminProviderDef {
	out := make(map[string]AdminProviderDef, len(defs))
	for _, d := range defs {
		out[d.Key] = d
	}
	return out
}

func AdminCatalog() []AdminProviderDef {
	return catalog
}

func Definition(code string) (AdminProviderDef, bool) {
	d, ok := catalogIndex[code]
	return d, ok
}

func (d AdminProviderDef) Field(key string) (AdminField, bool) {
	for _, f := range d.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return AdminField{}, false
}

func (d AdminProviderDef) MissingFields(cfg map[string]any) []string {
	out := []string{}
	for _, f := range d.Fields {
		if f.Required && strCfg(cfg, f.Key) == "" {
			out = append(out, f.Label)
		}
	}
	return out
}

func (d AdminProviderDef) ChargeCurrency(cfg map[string]any, walletCurrency string) string {
	if d.FixedCurrency != "" {
		return d.FixedCurrency
	}
	if d.CurrencyFrom != nil {
		if c := d.CurrencyFrom(cfg); c != "" {
			return c
		}
	}
	if d.CurrencyField != "" {
		if c := strings.ToUpper(strCfg(cfg, d.CurrencyField)); c != "" {
			return c
		}
		if f, ok := d.Field(d.CurrencyField); ok && f.Default != "" {
			return strings.ToUpper(f.Default)
		}
	}
	return strings.ToUpper(strings.TrimSpace(walletCurrency))
}

func (d AdminProviderDef) CurrencyMode() string {
	switch {
	case d.FixedCurrency != "":
		return "fixed"
	case d.CurrencyFrom != nil:
		return "account"
	case d.CurrencyField != "":
		return "setting"
	}
	return "wallet"
}

func webmoneyCurrency(cfg map[string]any) string {
	purse := strings.ToUpper(strCfg(cfg, "purse"))
	if purse == "" {
		return ""
	}
	switch purse[0] {
	case 'Z':
		return "USD"
	case 'E':
		return "EUR"
	case 'P', 'R':
		return "RUB"
	}
	return ""
}

func perfectMoneyCurrency(cfg map[string]any) string {
	account := strings.ToUpper(strCfg(cfg, "payee_account"))
	if account == "" {
		return ""
	}
	switch account[0] {
	case 'U':
		return "USD"
	case 'E':
		return "EUR"
	}
	return ""
}
