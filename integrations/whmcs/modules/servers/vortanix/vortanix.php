<?php

if (!defined('WHMCS')) {
    die('This file cannot be accessed directly');
}

require_once __DIR__ . '/lib/Api.php';

use WHMCS\Database\Capsule;

function vortanix_MetaData()
{
    return [
        'DisplayName' => 'Vortanix',
        'APIVersion' => '1.1',
        'RequiresServer' => true,
        'DefaultNonSSLPort' => '80',
        'DefaultSSLPort' => '443',
        'ServiceSingleSignOnLabel' => vortanix_t('sso_label'),
    ];
}

function vortanix_ConfigOptions()
{
    return [
        'tariff' => [
            'FriendlyName' => vortanix_t('option_tariff'),
            'Type' => 'text',
            'Size' => '40',
            'Loader' => 'vortanix_LoaderTariffs',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_tariff_hint'),
        ],
        'game' => [
            'FriendlyName' => vortanix_t('option_game'),
            'Type' => 'text',
            'Size' => '30',
            'Loader' => 'vortanix_LoaderGames',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_game_hint'),
        ],
        'location' => [
            'FriendlyName' => vortanix_t('option_location'),
            'Type' => 'text',
            'Size' => '30',
            'Loader' => 'vortanix_LoaderLocations',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_location_hint'),
        ],
        'version' => [
            'FriendlyName' => vortanix_t('option_version'),
            'Type' => 'text',
            'Size' => '30',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_version_hint'),
        ],
        'slots' => [
            'FriendlyName' => vortanix_t('option_slots'),
            'Type' => 'text',
            'Size' => '6',
            'Default' => '0',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_resource_hint'),
        ],
        'cpu_cores' => [
            'FriendlyName' => vortanix_t('option_cpu'),
            'Type' => 'text',
            'Size' => '6',
            'Default' => '0',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_resource_hint'),
        ],
        'ram_gb' => [
            'FriendlyName' => vortanix_t('option_ram'),
            'Type' => 'text',
            'Size' => '6',
            'Default' => '0',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_resource_hint'),
        ],
        'disk_gb' => [
            'FriendlyName' => vortanix_t('option_disk'),
            'Type' => 'text',
            'Size' => '6',
            'Default' => '0',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_resource_hint'),
        ],
        'server_name' => [
            'FriendlyName' => vortanix_t('option_name'),
            'Type' => 'text',
            'Size' => '40',
            'SimpleMode' => true,
            'Description' => vortanix_t('option_name_hint'),
        ],
    ];
}

function vortanix_catalog(array $params)
{
    static $catalog = null;
    if ($catalog === null) {
        $catalog = VortanixApi::fromParams($params)->get('/catalog');
    }
    return $catalog;
}

function vortanix_LoaderTariffs(array $params)
{
    $options = [];
    foreach ((array) (vortanix_catalog($params)['tariffs'] ?? []) as $tariff) {
        $game = trim((string) ($tariff['game_name'] ?? ''));
        $label = (string) ($tariff['name'] ?? '') . ' — ' . ($game !== '' ? $game : vortanix_t('loader_any_game'));
        $location = trim((string) ($tariff['location_name'] ?? ''));
        if ($location !== '') {
            $label .= ', ' . $location;
        }
        $options[(string) $tariff['id']] = $label;
    }
    return $options;
}

function vortanix_LoaderGames(array $params)
{
    $options = ['' => vortanix_t('loader_from_plan')];
    foreach ((array) (vortanix_catalog($params)['games'] ?? []) as $game) {
        $options[(string) $game['slug']] = (string) ($game['name'] ?? $game['slug']);
    }
    return $options;
}

function vortanix_LoaderLocations(array $params)
{
    $options = ['' => vortanix_t('loader_auto')];
    foreach ((array) (vortanix_catalog($params)['locations'] ?? []) as $location) {
        $label = (string) ($location['name'] ?? '');
        $country = trim((string) ($location['country'] ?? ''));
        if ($country !== '') {
            $label .= ' (' . $country . ')';
        }
        $options[(string) $location['id']] = $label;
    }
    return $options;
}

function vortanix_normalize_name($value)
{
    $parts = explode('|', (string) $value);
    return strtolower(str_replace(' ', '_', trim($parts[0])));
}

function vortanix_option(array $params, $name, $position)
{
    foreach (['configoptions', 'customfields'] as $group) {
        if (empty($params[$group]) || !is_array($params[$group])) {
            continue;
        }
        foreach ($params[$group] as $key => $value) {
            if (vortanix_normalize_name($key) !== $name) {
                continue;
            }
            $parts = explode('|', (string) $value);
            $value = trim($parts[0]);
            if ($value !== '' && $value !== '0') {
                return $value;
            }
        }
    }
    return trim((string) ($params['configoption' . $position] ?? ''));
}

function vortanix_server_name(array $params)
{
    if (!empty($params['customfields']) && is_array($params['customfields'])) {
        foreach ($params['customfields'] as $key => $value) {
            if (vortanix_normalize_name($key) === 'server_name' && trim((string) $value) !== '') {
                return trim((string) $value);
            }
        }
    }
    $template = trim((string) ($params['configoption9'] ?? ''));
    if ($template === '') {
        return '';
    }
    return trim(strtr($template, [
        '{service_id}' => (string) ($params['serviceid'] ?? ''),
        '{client_id}' => (string) ($params['userid'] ?? ''),
        '{domain}' => (string) ($params['domain'] ?? ''),
    ]));
}

function vortanix_hosting(array $params)
{
    try {
        return Capsule::table('tblhosting as h')
            ->leftJoin('tblproducts as p', 'p.id', '=', 'h.packageid')
            ->where('h.id', (int) $params['serviceid'])
            ->first(['h.billingcycle', 'h.nextduedate', 'p.name as product']);
    } catch (\Throwable $e) {
        return null;
    }
}

function vortanix_due_date($value)
{
    $value = substr(trim((string) $value), 0, 10);
    if ($value === '0000-00-00' || !preg_match('/^\d{4}-\d{2}-\d{2}$/', $value)) {
        return '';
    }
    return $value;
}

function vortanix_service_payload(array $params)
{
    $hosting = vortanix_hosting($params);
    return [
        'tariff_id' => vortanix_option($params, 'tariff', 1),
        'game' => vortanix_option($params, 'game', 2),
        'location' => vortanix_option($params, 'location', 3),
        'version' => vortanix_option($params, 'version', 4),
        'slots' => (int) vortanix_option($params, 'slots', 5),
        'cpu_cores' => (int) vortanix_option($params, 'cpu_cores', 6),
        'ram_gb' => (int) vortanix_option($params, 'ram_gb', 7),
        'disk_gb' => (int) vortanix_option($params, 'disk_gb', 8),
        'name' => vortanix_server_name($params),
        'product' => $hosting ? (string) $hosting->product : '',
        'billing_cycle' => $hosting ? (string) $hosting->billingcycle : '',
        'next_due_date' => $hosting ? vortanix_due_date($hosting->nextduedate) : '',
    ];
}

function vortanix_panel_locale($language)
{
    $map = [
        'russian' => 'ru',
        'english' => 'en',
        'ukrainian' => 'uk',
        'german' => 'de',
        'french' => 'fr',
        'spanish' => 'es',
        'italian' => 'it',
        'polish' => 'pl',
        'turkish' => 'tr',
        'portuguese-br' => 'pt',
        'portuguese-pt' => 'pt',
        'chinese' => 'zh',
    ];
    $language = strtolower(trim((string) $language));
    if ($language === '') {
        try {
            $language = strtolower((string) \WHMCS\Config\Setting::getValue('Language'));
        } catch (\Throwable $e) {
            $language = '';
        }
    }
    return $map[$language] ?? '';
}

function vortanix_client_payload(array $params)
{
    $client = (isset($params['clientsdetails']) && is_array($params['clientsdetails'])) ? $params['clientsdetails'] : [];
    return [
        'id' => (int) ($params['userid'] ?? 0),
        'email' => (string) ($client['email'] ?? ''),
        'first_name' => (string) ($client['firstname'] ?? ''),
        'last_name' => (string) ($client['lastname'] ?? ''),
        'locale' => vortanix_panel_locale($client['language'] ?? ''),
    ];
}

function vortanix_encrypt($value)
{
    if (function_exists('encrypt')) {
        return encrypt($value);
    }
    $result = localAPI('EncryptPassword', ['password2' => $value]);
    return (is_array($result) && isset($result['password'])) ? (string) $result['password'] : '';
}

function vortanix_store_service(array $params, array $info)
{
    $update = [];
    if (!empty($info['username'])) {
        $update['username'] = (string) $info['username'];
    }
    if (!empty($info['password'])) {
        $encrypted = vortanix_encrypt((string) $info['password']);
        if ($encrypted !== '') {
            $update['password'] = $encrypted;
        }
    }
    $server = (isset($info['server']) && is_array($info['server'])) ? $info['server'] : null;
    if ($server) {
        if (!empty($server['ip_address'])) {
            $update['dedicatedip'] = (string) $server['ip_address'];
        }
        if (trim((string) ($params['domain'] ?? '')) === '' && !empty($server['address'])) {
            $update['domain'] = (string) $server['address'];
        }
    }
    if (!$update) {
        return;
    }
    try {
        Capsule::table('tblhosting')->where('id', (int) $params['serviceid'])->update($update);
    } catch (\Throwable $e) {
        logActivity('Vortanix: ' . $e->getMessage());
    }
}

function vortanix_service_path(array $params, $suffix = '')
{
    return '/services/' . (int) $params['serviceid'] . $suffix;
}

function vortanix_e($value)
{
    return htmlspecialchars((string) $value, ENT_QUOTES, 'UTF-8');
}

function vortanix_TestConnection(array $params)
{
    try {
        VortanixApi::fromParams($params)->get('/info');
        return ['success' => true, 'error' => ''];
    } catch (\Throwable $e) {
        return ['success' => false, 'error' => $e->getMessage()];
    }
}

function vortanix_CreateAccount(array $params)
{
    try {
        $payload = vortanix_service_payload($params);
        $payload['client'] = vortanix_client_payload($params);
        $payload['password'] = (string) ($params['password'] ?? '');
        $info = VortanixApi::fromParams($params)->post(vortanix_service_path($params), $payload);
        vortanix_store_service($params, $info);
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_SuspendAccount(array $params)
{
    try {
        VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/suspend'), [
            'reason' => (string) ($params['suspendreason'] ?? ''),
        ]);
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_UnsuspendAccount(array $params)
{
    try {
        VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/unsuspend'));
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_TerminateAccount(array $params)
{
    try {
        VortanixApi::fromParams($params)->delete(vortanix_service_path($params));
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_ChangePackage(array $params)
{
    try {
        $info = VortanixApi::fromParams($params)->post(
            vortanix_service_path($params, '/package'),
            vortanix_service_payload($params)
        );
        vortanix_store_service($params, $info);
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_ChangePassword(array $params)
{
    try {
        VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/password'), [
            'password' => (string) ($params['password'] ?? ''),
        ]);
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_Renew(array $params)
{
    try {
        $hosting = vortanix_hosting($params);
        VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/renew'), [
            'next_due_date' => $hosting ? vortanix_due_date($hosting->nextduedate) : '',
        ]);
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_ServiceSingleSignOn(array $params)
{
    try {
        $result = VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/sso'));
        if (empty($result['url'])) {
            return ['success' => false, 'errorMsg' => vortanix_t('error_bad_response', ['code' => 200])];
        }
        return ['success' => true, 'redirectTo' => (string) $result['url']];
    } catch (\Throwable $e) {
        return ['success' => false, 'errorMsg' => $e->getMessage()];
    }
}

function vortanix_power(array $params, $action)
{
    try {
        VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/power'), ['action' => $action]);
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_start(array $params)
{
    return vortanix_power($params, 'start');
}

function vortanix_stop(array $params)
{
    return vortanix_power($params, 'stop');
}

function vortanix_restart(array $params)
{
    return vortanix_power($params, 'restart');
}

function vortanix_reinstall(array $params)
{
    try {
        VortanixApi::fromParams($params)->post(vortanix_service_path($params, '/reinstall'));
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_sync(array $params)
{
    try {
        $api = VortanixApi::fromParams($params);
        $hosting = vortanix_hosting($params);
        if ($hosting) {
            $api->post(vortanix_service_path($params, '/renew'), [
                'next_due_date' => vortanix_due_date($hosting->nextduedate),
            ]);
        }
        vortanix_store_service($params, $api->get(vortanix_service_path($params)));
        return 'success';
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
}

function vortanix_AdminCustomButtonArray()
{
    return [
        vortanix_t('button_start') => 'start',
        vortanix_t('button_stop') => 'stop',
        vortanix_t('button_restart') => 'restart',
        vortanix_t('button_reinstall') => 'reinstall',
        vortanix_t('button_sync') => 'sync',
    ];
}

function vortanix_ClientAreaCustomButtonArray()
{
    return [
        vortanix_t('button_start') => 'start',
        vortanix_t('button_restart') => 'restart',
        vortanix_t('button_stop') => 'stop',
    ];
}

function vortanix_state_label($state)
{
    $key = 'state_' . strtolower((string) $state);
    $label = vortanix_t($key);
    return $label === $key ? (string) $state : $label;
}

function vortanix_state_tone($state)
{
    switch (strtolower((string) $state)) {
        case 'running':
            return 'success';
        case 'error':
            return 'danger';
        case 'starting':
        case 'stopping':
        case 'installing':
        case 'reinstalling':
        case 'updating':
            return 'warning';
        default:
            return 'default';
    }
}

function vortanix_resources(array $server)
{
    $limits = (isset($server['limits']) && is_array($server['limits'])) ? $server['limits'] : [];
    $cpu = $limits['cpu'] ?? ($limits['cpu_cores'] ?? 0);
    $values = [
        'res_slots' => (int) ($limits['slots'] ?? 0),
        'res_cpu' => is_numeric($cpu) ? $cpu + 0 : 0,
        'res_ram' => (int) round(((float) ($limits['memory_mb'] ?? 0)) / 1024),
        'res_disk' => (int) round(((float) ($limits['disk_mb'] ?? 0)) / 1024),
    ];
    $parts = [];
    foreach ($values as $key => $value) {
        if ($value > 0) {
            $parts[] = vortanix_t($key, ['value' => $value]);
        }
    }
    return implode(' · ', $parts);
}

function vortanix_ClientArea(array $params)
{
    $vars = [
        'vtxLang' => vortanix_lang(),
        'vtxServiceId' => (int) $params['serviceid'],
        'vtxError' => false,
        'vtxServer' => null,
        'vtxSuspended' => false,
        'vtxBlocked' => '',
        'vtxCanPower' => false,
        'vtxStatusLabel' => '',
        'vtxStatusTone' => 'default',
        'vtxStatusBadge' => 'secondary',
        'vtxRows' => [],
    ];
    try {
        $info = VortanixApi::fromParams($params)->get(vortanix_service_path($params));
    } catch (\Throwable $e) {
        $vars['vtxError'] = true;
        return ['tabOverviewModuleOutputTemplate' => 'templates/clientarea.tpl', 'templateVariables' => $vars];
    }

    $vars['vtxSuspended'] = ($info['status'] ?? '') === 'suspended';
    $server = (isset($info['server']) && is_array($info['server'])) ? $info['server'] : null;
    if ($server) {
        $state = (string) ($server['status'] ?? '');
        $tone = vortanix_state_tone($state);
        $vars['vtxServer'] = ['name' => (string) ($server['name'] ?? '')];
        $vars['vtxStatusLabel'] = vortanix_state_label($state);
        $vars['vtxStatusTone'] = $tone;
        $vars['vtxStatusBadge'] = $tone === 'default' ? 'secondary' : $tone;
        if (!$vars['vtxSuspended'] && !empty($server['is_blocked'])) {
            $reason = trim((string) ($server['blocked_reason'] ?? ''));
            $vars['vtxBlocked'] = vortanix_t('client_blocked', ['reason' => $reason !== '' ? $reason : '—']);
        }
        $vars['vtxCanPower'] = !$vars['vtxSuspended'] && empty($server['is_blocked']);

        $rows = [];
        if (!empty($server['address'])) {
            $rows[] = ['label' => vortanix_t('client_address'), 'value' => (string) $server['address'], 'code' => true];
        }
        if (!empty($server['game']['name'])) {
            $rows[] = ['label' => vortanix_t('client_game'), 'value' => (string) $server['game']['name'], 'code' => false];
        }
        if (!empty($server['location']['name'])) {
            $rows[] = ['label' => vortanix_t('client_location'), 'value' => (string) $server['location']['name'], 'code' => false];
        }
        $resources = vortanix_resources($server);
        if ($resources !== '') {
            $rows[] = ['label' => vortanix_t('client_resources'), 'value' => $resources, 'code' => false];
        }
        $vars['vtxRows'] = $rows;
    }

    return ['tabOverviewModuleOutputTemplate' => 'templates/clientarea.tpl', 'templateVariables' => $vars];
}

function vortanix_AdminServicesTabFields(array $params)
{
    try {
        $info = VortanixApi::fromParams($params)->get(vortanix_service_path($params));
    } catch (\Throwable $e) {
        return [vortanix_t('admin_server') => '<span class="label label-danger">' . vortanix_e($e->getMessage()) . '</span>'];
    }

    $fields = [vortanix_t('admin_status') => vortanix_e(vortanix_t('status_' . ($info['status'] ?? 'pending')))];
    $server = (isset($info['server']) && is_array($info['server'])) ? $info['server'] : null;
    if (!$server) {
        $fields[vortanix_t('admin_server')] = vortanix_e(vortanix_t('server_missing'));
        return $fields;
    }

    $name = vortanix_e($server['name'] ?? '');
    $link = (string) ($server['admin_url'] ?? '');
    if ($link !== '') {
        $name = '<a href="' . vortanix_e($link) . '" target="_blank" rel="noopener">' . $name . '</a>';
    }
    $fields[vortanix_t('admin_server')] = $name . ' — ' . vortanix_e(vortanix_state_label($server['status'] ?? ''));
    if (!empty($server['address'])) {
        $fields[vortanix_t('admin_address')] = '<code>' . vortanix_e($server['address']) . '</code>';
    }
    if (!empty($server['game']['name'])) {
        $fields[vortanix_t('admin_game')] = vortanix_e($server['game']['name']);
    }
    if (!empty($server['location']['name'])) {
        $fields[vortanix_t('admin_location')] = vortanix_e($server['location']['name']);
    }
    if (!empty($server['tariff']['name'])) {
        $fields[vortanix_t('admin_plan')] = vortanix_e($server['tariff']['name']);
    }
    $resources = vortanix_resources($server);
    if ($resources !== '') {
        $fields[vortanix_t('admin_resources')] = vortanix_e($resources);
    }
    if (!empty($info['user']['email'])) {
        $fields[vortanix_t('admin_owner')] = vortanix_e($info['user']['email']);
    }
    return $fields;
}

function vortanix_AdminLink(array $params)
{
    try {
        $base = VortanixApi::fromParams($params)->baseUrl();
    } catch (\Throwable $e) {
        return '';
    }
    return '<a class="btn btn-default btn-sm" href="' . vortanix_e($base . '/admin/dashboard')
        . '" target="_blank" rel="noopener">' . vortanix_e(vortanix_t('admin_open_panel')) . '</a>';
}

function vortanix_UsageUpdate(array $params)
{
    try {
        $usage = VortanixApi::fromParams($params)->get('/usage');
    } catch (\Throwable $e) {
        return $e->getMessage();
    }
    $now = date('Y-m-d H:i:s');
    foreach ((array) ($usage['services'] ?? []) as $row) {
        try {
            Capsule::table('tblhosting')
                ->where('id', (int) ($row['service_id'] ?? 0))
                ->where('server', (int) ($params['serverid'] ?? 0))
                ->update([
                    'diskusage' => (int) ($row['disk_used_mb'] ?? 0),
                    'disklimit' => (int) ($row['disk_limit_mb'] ?? 0),
                    'bwusage' => (int) ($row['bandwidth_used_mb'] ?? 0),
                    'bwlimit' => (int) ($row['bandwidth_limit_mb'] ?? 0),
                    'lastupdate' => $now,
                ]);
        } catch (\Throwable $e) {
            logActivity('Vortanix: ' . $e->getMessage());
        }
    }
    return 'success';
}
