<?php

if (!defined('WHMCS')) {
    die('This file cannot be accessed directly');
}

if (!function_exists('vortanix_lang')) {
    function vortanix_lang()
    {
        static $lang = null;
        if ($lang !== null) {
            return $lang;
        }
        $dir = dirname(__DIR__) . '/lang/';
        $fallback = require $dir . 'english.php';
        $lang = is_array($fallback) ? $fallback : [];
        $code = vortanix_lang_code($dir);
        if ($code !== 'english') {
            $loaded = require $dir . $code . '.php';
            if (is_array($loaded)) {
                $lang = array_merge($lang, $loaded);
            }
        }
        return $lang;
    }

    function vortanix_lang_code($dir)
    {
        $candidates = [];
        if (defined('ADMINAREA') && !empty($_SESSION['adminid'])) {
            try {
                $candidates[] = \WHMCS\Database\Capsule::table('tbladmins')
                    ->where('id', (int) $_SESSION['adminid'])
                    ->value('language');
            } catch (\Throwable $e) {
            }
        }
        if (!empty($_SESSION['Language'])) {
            $candidates[] = $_SESSION['Language'];
        }
        try {
            $candidates[] = \WHMCS\Config\Setting::getValue('Language');
        } catch (\Throwable $e) {
        }
        foreach ($candidates as $candidate) {
            $candidate = strtolower(preg_replace('/[^a-z_-]/i', '', (string) $candidate));
            if ($candidate !== '' && is_file($dir . $candidate . '.php')) {
                return $candidate;
            }
        }
        return 'english';
    }

    function vortanix_t($key, array $replace = [])
    {
        $lang = vortanix_lang();
        $text = isset($lang[$key]) ? (string) $lang[$key] : (string) $key;
        foreach ($replace as $name => $value) {
            $text = str_replace('{' . $name . '}', (string) $value, $text);
        }
        return $text;
    }
}

if (!class_exists('VortanixApi')) {
    class VortanixApiException extends \Exception
    {
    }

    class VortanixApi
    {
        private $baseUrl;
        private $key;
        private $timeout;

        public function __construct($baseUrl, $key, $timeout = 30)
        {
            $this->baseUrl = rtrim((string) $baseUrl, '/');
            $this->key = trim((string) $key);
            $this->timeout = (int) $timeout;
        }

        public static function fromParams(array $params)
        {
            $host = trim((string) (isset($params['serverhostname']) ? $params['serverhostname'] : ''));
            if ($host === '') {
                $host = trim((string) (isset($params['serverip']) ? $params['serverip'] : ''));
            }
            if ($host === '') {
                throw new VortanixApiException(vortanix_t('error_no_host'));
            }
            if (!preg_match('~^https?://~i', $host)) {
                $secureRaw = isset($params['serversecure']) ? $params['serversecure'] : '';
                $secure = !empty($secureRaw) && $secureRaw !== 'off';
                $port = (int) (isset($params['serverport']) ? $params['serverport'] : 0);
                $host = ($secure ? 'https://' : 'http://') . $host;
                if ($port > 0 && !(($secure && $port === 443) || (!$secure && $port === 80))) {
                    $host .= ':' . $port;
                }
            }
            $host = preg_replace('~/v1/?$~', '', rtrim($host, '/'));
            $key = trim((string) (isset($params['serveraccesshash']) ? $params['serveraccesshash'] : ''));
            if ($key === '') {
                $key = trim((string) (isset($params['serverpassword']) ? $params['serverpassword'] : ''));
            }
            if ($key === '') {
                throw new VortanixApiException(vortanix_t('error_no_key'));
            }
            return new self($host, $key);
        }

        public static function fromServerId($serverId)
        {
            $row = \WHMCS\Database\Capsule::table('tblservers')->where('id', (int) $serverId)->first();
            if (!$row) {
                throw new VortanixApiException(vortanix_t('error_no_host'));
            }
            $password = '';
            if (!empty($row->password)) {
                $decrypted = localAPI('DecryptPassword', ['password2' => $row->password]);
                if (is_array($decrypted) && isset($decrypted['password'])) {
                    $password = (string) $decrypted['password'];
                }
            }
            return self::fromParams([
                'serverhostname' => $row->hostname,
                'serverip' => $row->ipaddress,
                'serversecure' => $row->secure,
                'serverport' => isset($row->port) ? $row->port : 0,
                'serveraccesshash' => $row->accesshash,
                'serverpassword' => $password,
            ]);
        }

        public function baseUrl()
        {
            return $this->baseUrl;
        }

        public function get($path)
        {
            return $this->request('GET', $path, null);
        }

        public function post($path, array $body = [])
        {
            return $this->request('POST', $path, $body);
        }

        public function put($path, array $body = [])
        {
            return $this->request('PUT', $path, $body);
        }

        public function delete($path)
        {
            return $this->request('DELETE', $path, null);
        }

        private function request($method, $path, $body)
        {
            $url = $this->baseUrl . '/v1/admin/whmcs' . $path;
            $headers = ['Accept: application/json', 'X-API-Key: ' . $this->key];
            $payload = null;
            $ch = curl_init($url);
            if ($body !== null) {
                $payload = json_encode($body, JSON_UNESCAPED_UNICODE);
                $headers[] = 'Content-Type: application/json';
                curl_setopt($ch, CURLOPT_POSTFIELDS, $payload);
            }
            curl_setopt_array($ch, [
                CURLOPT_CUSTOMREQUEST => $method,
                CURLOPT_RETURNTRANSFER => true,
                CURLOPT_HTTPHEADER => $headers,
                CURLOPT_TIMEOUT => $this->timeout,
                CURLOPT_CONNECTTIMEOUT => 10,
                CURLOPT_FOLLOWLOCATION => false,
            ]);
            $raw = curl_exec($ch);
            $error = curl_error($ch);
            $code = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
            curl_close($ch);

            if (function_exists('logModuleCall')) {
                logModuleCall('vortanix', $method . ' ' . $path, $payload, $raw, null, [$this->key]);
            }
            if ($raw === false) {
                throw new VortanixApiException(vortanix_t('error_unreachable', ['error' => $error]));
            }
            $data = json_decode((string) $raw, true);
            if ($code >= 400) {
                if (is_array($data) && !empty($data['error'])) {
                    throw new VortanixApiException((string) $data['error'], $code);
                }
                throw new VortanixApiException(vortanix_t('error_bad_response', ['code' => $code]), $code);
            }
            if (!is_array($data)) {
                throw new VortanixApiException(vortanix_t('error_bad_response', ['code' => $code]), $code);
            }
            return $data;
        }
    }
}
