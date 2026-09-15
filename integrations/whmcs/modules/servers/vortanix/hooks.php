<?php

if (!defined('WHMCS')) {
    die('This file cannot be accessed directly');
}

require_once __DIR__ . '/lib/Api.php';

use WHMCS\Database\Capsule;

add_hook('ClientEdit', 1, function (array $vars) {
    $clientId = (int) ($vars['userid'] ?? 0);
    if ($clientId <= 0) {
        return;
    }
    try {
        $serverIds = Capsule::table('tblhosting as h')
            ->join('tblservers as s', 's.id', '=', 'h.server')
            ->where('h.userid', $clientId)
            ->where('s.type', 'vortanix')
            ->distinct()
            ->pluck('s.id');
    } catch (\Throwable $e) {
        logActivity('Vortanix: ' . $e->getMessage());
        return;
    }
    foreach ($serverIds as $serverId) {
        try {
            VortanixApi::fromServerId((int) $serverId)->put('/clients/' . $clientId, [
                'email' => (string) ($vars['email'] ?? ''),
                'first_name' => (string) ($vars['firstname'] ?? ''),
                'last_name' => (string) ($vars['lastname'] ?? ''),
            ]);
        } catch (\Throwable $e) {
            logActivity('Vortanix: ' . $e->getMessage());
        }
    }
});
