<?php

namespace App\Providers;

use App\Models\Setting;
use Illuminate\Support\Facades\Schema;
use Illuminate\Support\ServiceProvider;

class AppServiceProvider extends ServiceProvider
{

    public function register(): void
    {
        //
    }

    public function boot(): void
    {
        $defaultConnection = (string) config('database.default', '');
        if ($defaultConnection === '') {
            return;
        }

        if ($defaultConnection === 'mysql') {
            $mysqlHost = (string) config('database.connections.mysql.host', '');
            $mysqlDatabase = (string) config('database.connections.mysql.database', '');

            if ($mysqlHost === '' || $mysqlDatabase === '') {
                return;
            }
        }

        try {
            if (! Schema::hasTable('settings')) {
                return;
            }
        } catch (\Throwable $e) {
            return;
        }

        $settings = Setting::allCached();

        $settings = array_merge([
            'app.name' => '',
            'app.links.telegram' => '',
            'app.links.discord' => '',
            'app.links.support' => '',
            'app.branding.logo' => '',
            'app.branding.icon' => '',
            'app.site.description' => '',
            'app.site.domain' => '',
            'app.site.ip' => '',
            'app.site.subnet' => '',
            'app.site.default_template' => '',
            'services.recaptcha.site_key' => '',
            'services.recaptcha.secret_key' => '',
            'mail.default' => '',
            'mail.mailers.smtp.host' => '',
            'mail.mailers.smtp.port' => '',
            'mail.mailers.smtp.username' => '',
            'mail.mailers.smtp.password' => '',
            'mail.mailers.smtp.scheme' => '',
            'mail.from.address' => '',
            'mail.from.name' => '',
            'services.freekassa.merchant_id' => '',
            'services.freekassa.secret1' => '',
            'services.freekassa.secret2' => '',
            'services.freekassa.currency' => '',
            'services.freekassa.pay_url' => '',
            'services.freekassa.methods' => '',
            'services.nowpayments.api_key' => '',
            'services.nowpayments.ipn_secret' => '',
            'services.nowpayments.api_url' => '',
            'services.nowpayments.price_currency' => '',
            'services.nowpayments.pay_currency' => '',
        ], (array) $settings);

        $appName = (string) $settings['app.name'];
        if ($appName !== '') {
            config()->set('app.name', $appName);
        }

        $telegram = (string) $settings['app.links.telegram'];
        $discord = (string) $settings['app.links.discord'];
        $support = (string) $settings['app.links.support'];
        if ($telegram !== '' || $discord !== '' || $support !== '') {
            $links = (array) config('app.links');
            if ($telegram !== '') {
                $links['telegram'] = $telegram;
            }
            if ($discord !== '') {
                $links['discord'] = $discord;
            }
            if ($support !== '') {
                $links['support'] = $support;
            }
            config()->set('app.links', $links);
        }

        $logo = (string) $settings['app.branding.logo'];
        $icon = (string) $settings['app.branding.icon'];
        if ($logo !== '' || $icon !== '') {
            $branding = (array) config('app.branding');
            if ($logo !== '') {
                $branding['logo'] = $logo;
            }
            if ($icon !== '') {
                $branding['icon'] = $icon;
            }
            config()->set('app.branding', $branding);
        }

        $siteDescription = (string) $settings['app.site.description'];
        $siteDomain = (string) $settings['app.site.domain'];
        $siteIp = (string) $settings['app.site.ip'];
        $siteSubnet = (string) $settings['app.site.subnet'];
        $defaultTemplate = (string) $settings['app.site.default_template'];
        if ($siteDescription !== '' || $siteDomain !== '' || $siteIp !== '' || $siteSubnet !== '' || $defaultTemplate !== '') {
            $site = (array) config('app.site');
            if ($siteDescription !== '') {
                $site['description'] = $siteDescription;
            }
            if ($siteDomain !== '') {
                $site['domain'] = $siteDomain;
            }
            if ($siteIp !== '') {
                $site['ip'] = $siteIp;
            }
            if ($siteSubnet !== '') {
                $site['subnet'] = $siteSubnet;
            }
            if ($defaultTemplate !== '') {
                $site['default_template'] = $defaultTemplate;
            }
            config()->set('app.site', $site);
        }

        $recaptchaSiteKey = (string) $settings['services.recaptcha.site_key'];
        $recaptchaSecretKey = (string) $settings['services.recaptcha.secret_key'];
        if ($recaptchaSiteKey !== '' || $recaptchaSecretKey !== '') {
            $recaptcha = (array) config('services.recaptcha');
            if ($recaptchaSiteKey !== '') {
                $recaptcha['site_key'] = $recaptchaSiteKey;
            }
            if ($recaptchaSecretKey !== '') {
                $recaptcha['secret_key'] = $recaptchaSecretKey;
            }
            config()->set('services.recaptcha', $recaptcha);
        }

        $mailDefault = (string) $settings['mail.default'];
        if ($mailDefault !== '') {
            config()->set('mail.default', $mailDefault);
        }

        $smtpHost = (string) $settings['mail.mailers.smtp.host'];
        $smtpPort = (string) $settings['mail.mailers.smtp.port'];
        $smtpUsername = (string) $settings['mail.mailers.smtp.username'];
        $smtpPassword = (string) $settings['mail.mailers.smtp.password'];
        $smtpScheme = (string) $settings['mail.mailers.smtp.scheme'];
        if ($smtpHost !== '' || $smtpPort !== '' || $smtpUsername !== '' || $smtpPassword !== '' || $smtpScheme !== '') {
            $mailers = (array) config('mail.mailers');
            $smtp = array_merge((array) $mailers['smtp'], []);
            if ($smtpHost !== '') {
                $smtp['host'] = $smtpHost;
            }
            if ($smtpPort !== '') {
                $smtp['port'] = (int) $smtpPort;
            }
            if ($smtpUsername !== '') {
                $smtp['username'] = $smtpUsername;
            }
            if ($smtpPassword !== '') {
                $smtp['password'] = $smtpPassword;
            }
            if ($smtpScheme !== '') {
                $smtp['scheme'] = $smtpScheme;
            }
            $mailers['smtp'] = $smtp;
            config()->set('mail.mailers', $mailers);
        }

        $mailFromAddress = (string) $settings['mail.from.address'];
        $mailFromName = (string) $settings['mail.from.name'];
        if ($mailFromAddress !== '' || $mailFromName !== '') {
            $from = (array) config('mail.from');
            if ($mailFromAddress !== '') {
                $from['address'] = $mailFromAddress;
            }
            if ($mailFromName !== '') {
                $from['name'] = $mailFromName;
            }
            config()->set('mail.from', $from);
        }

        $fkMerchant = (string) $settings['services.freekassa.merchant_id'];
        $fkSecret1 = (string) $settings['services.freekassa.secret1'];
        $fkSecret2 = (string) $settings['services.freekassa.secret2'];
        $fkCurrency = (string) $settings['services.freekassa.currency'];
        $fkPayUrl = (string) $settings['services.freekassa.pay_url'];
        $fkMethodsJson = (string) $settings['services.freekassa.methods'];

        if ($fkMerchant !== '') {
            config()->set('services.freekassa.merchant_id', $fkMerchant);
        }
        if ($fkSecret1 !== '') {
            config()->set('services.freekassa.secret1', $fkSecret1);
        }
        if ($fkSecret2 !== '') {
            config()->set('services.freekassa.secret2', $fkSecret2);
        }
        if ($fkCurrency !== '') {
            config()->set('services.freekassa.currency', $fkCurrency);
        }
        if ($fkPayUrl !== '') {
            config()->set('services.freekassa.pay_url', $fkPayUrl);
        }
        if ($fkMethodsJson !== '') {
            $decoded = json_decode($fkMethodsJson, true);
            if (is_array($decoded)) {
                config()->set('services.freekassa.methods', $decoded);
            }
        }

        $npApiKey = (string) $settings['services.nowpayments.api_key'];
        $npIpnSecret = (string) $settings['services.nowpayments.ipn_secret'];
        $npApiUrl = (string) $settings['services.nowpayments.api_url'];
        $npPriceCurrency = (string) $settings['services.nowpayments.price_currency'];
        $npPayCurrency = (string) $settings['services.nowpayments.pay_currency'];
        if ($npApiKey !== '' || $npIpnSecret !== '' || $npApiUrl !== '' || $npPriceCurrency !== '' || $npPayCurrency !== '') {
            $np = (array) config('services.nowpayments');
            if ($npApiKey !== '') {
                $np['api_key'] = $npApiKey;
            }
            if ($npIpnSecret !== '') {
                $np['ipn_secret'] = $npIpnSecret;
            }
            if ($npApiUrl !== '') {
                $np['api_url'] = $npApiUrl;
            }
            if ($npPriceCurrency !== '') {
                $np['price_currency'] = $npPriceCurrency;
            }
            if ($npPayCurrency !== '') {
                $np['pay_currency'] = $npPayCurrency;
            }
            config()->set('services.nowpayments', $np);
        }
    }
}
