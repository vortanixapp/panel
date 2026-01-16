<?php

return [
    'sources' => [
        'laravel' => [
            'label' => 'Laravel',
            'type' => 'directory',
            'base_path' => storage_path('logs'),
            'pattern' => '*.log',
        ],
        'nginx' => [
            'label' => 'Nginx',
            'type' => 'files',
            'files' => [
                '/var/log/nginx/access.log',
                '/var/log/nginx/error.log',
            ],
        ],
        'daemon' => [
            'label' => 'Vortanix Daemon',
            'type' => 'files',
            'files' => [],
        ],
        'docker' => [
            'label' => 'Docker',
            'type' => 'command',
            'enabled' => (bool) env('ADMIN_LOGS_ALLOW_SYSTEM_COMMANDS', false),
            'list_command' => ['docker', 'ps', '--format', '{{.Names}}'],
            'tail_command' => ['docker', 'logs', '--tail'],
        ],
        'journal' => [
            'label' => 'systemd journal',
            'type' => 'command',
            'enabled' => (bool) env('ADMIN_LOGS_ALLOW_SYSTEM_COMMANDS', false),
            'units' => [],
            'tail_command' => ['journalctl', '--no-pager', '-n'],
        ],
    ],
];
