<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Third Party Services
    |--------------------------------------------------------------------------
    |
    | This file is for storing the credentials for third party services such
    | as Mailgun, Postmark, AWS and more. This file provides the de facto
    | location for this type of information, allowing packages to have
    | a conventional file to locate the various service credentials.
    |
    */

    'postmark' => [
        'key' => env('POSTMARK_API_KEY'),
    ],

    'resend' => [
        'key' => env('RESEND_API_KEY'),
    ],

    'ses' => [
        'key' => env('AWS_ACCESS_KEY_ID'),
        'secret' => env('AWS_SECRET_ACCESS_KEY'),
        'region' => env('AWS_DEFAULT_REGION', 'us-east-1'),
    ],

    'slack' => [
        'notifications' => [
            'bot_user_oauth_token' => env('SLACK_BOT_USER_OAUTH_TOKEN'),
            'channel' => env('SLACK_BOT_USER_DEFAULT_CHANNEL'),
        ],
    ],

    'monitoring' => [
        'token' => env('MONITORING_API_TOKEN'),
    ],

    'location_daemon' => [
        'port' => env('LOCATION_DAEMON_PORT', 9201),
        'token' => env('LOCATION_DAEMON_TOKEN', env('MONITORING_API_TOKEN')),
    ],

    'freekassa' => [
        'merchant_id' => env('FREEKASSA_MERCHANT_ID'),
        'secret1' => env('FREEKASSA_SECRET_1'),
        'secret2' => env('FREEKASSA_SECRET_2'),
        'currency' => env('FREEKASSA_CURRENCY', 'RUB'),
        'pay_url' => env('FREEKASSA_PAY_URL', 'https://pay.fk.money/'),
        'methods' => json_decode(env('FREEKASSA_METHODS', '[]'), true) ?: [],
        'check_ip' => env('FREEKASSA_CHECK_IP', false),
        'allowed_ips' => [
            '168.119.157.136',
            '168.119.60.227',
            '178.154.197.79',
            '51.250.54.238',
        ],
    ],

    'nowpayments' => [
        'api_key' => env('NOWPAYMENTS_API_KEY'),
        'ipn_secret' => env('NOWPAYMENTS_IPN_SECRET'),
        'api_url' => env('NOWPAYMENTS_API_URL', 'https://api.nowpayments.io'),
        'price_currency' => env('NOWPAYMENTS_PRICE_CURRENCY', 'RUB'),
        'pay_currency' => env('NOWPAYMENTS_PAY_CURRENCY', 'usdt'),
    ],

    'license_cloud' => [
        'url' => env('LICENSE_CLOUD_URL'),
        'panel_id' => env('LICENSE_CLOUD_PANEL_ID'),
        'hmac_secret' => env('LICENSE_CLOUD_HMAC_SECRET'),
        'server_ip' => env('LICENSE_CLOUD_SERVER_IP'),
        'timeout' => env('LICENSE_CLOUD_TIMEOUT', 5),
    ],

];
