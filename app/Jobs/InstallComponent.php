<?php

namespace App\Jobs;

use App\Models\Location;
use App\Models\LocationSetupStatus;
use App\Models\LocationDaemon;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Queue\Queueable;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Crypt;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;
use phpseclib3\Net\SSH2;

class InstallComponent implements ShouldQueue
{
    use Queueable;

    public $timeout = 600;

    private const SETUP_CACHE_TTL_SEC = 21600;

    protected Location $location;
    protected string $component;
    protected int $userId;

    private function updateSetupCache(string $cacheKey, array $patch): void
    {
        $state = Cache::get($cacheKey, []);
        foreach ($patch as $k => $v) {
            $state[$k] = $v;
        }
        Cache::put($cacheKey, $state, self::SETUP_CACHE_TTL_SEC);
    }

    private function appendSetupLog(string $cacheKey, string $message): void
    {
        $state = Cache::get($cacheKey, []);
        $state['log'] = (string) $state['log'] . $message;
        $state['completed'] = (bool) $state['completed'];
        Cache::put($cacheKey, $state, self::SETUP_CACHE_TTL_SEC);
    }

    public function __construct(Location $location, string $component, int $userId)
    {
        $this->location = $location;
        $this->component = $component;
        $this->userId = $userId;
    }

    public function handle(): void
    {
        $location = $this->location;
        $component = $this->component;
        $userId = $this->userId;

        Log::info("Starting InstallComponent job", ['component' => $component, 'location_id' => $location->id, 'user_id' => $userId]);

        try {
            LocationSetupStatus::updateOrCreate(
                ['location_id' => $location->id, 'component' => $component],
                ['status' => LocationSetupStatus::STATUS_INSTALLING]
            );

            $cacheKey = "setup_status_{$location->id}_{$userId}";

            $state = Cache::get($cacheKey, []);
            $state['log'] = (string) $state['log'];
            $state['completed'] = false;
            $state['component'] = $component;
            Cache::put($cacheKey, $state, self::SETUP_CACHE_TTL_SEC);
            Log::info("Cache initial", ['key' => $cacheKey, 'value' => $state]);

            $commands = $this->getCommandsForComponent($component);

            $mysqlRootUsername = null;
            $mysqlRootPassword = null;
            if ($component === 'mysql') {
                $mysqlRootUsername = 'root';
                $mysqlRootPassword = Str::password(16, true, true, false, false);
                $mysqlRootPassword = preg_replace('/[^a-zA-Z0-9]/', 'A', (string) $mysqlRootPassword);

                $commands[] = 'sudo sh -c "CONF=/etc/mysql/mysql.conf.d/mysqld.cnf; [ -f \"$CONF\" ] || CONF=/etc/mysql/mariadb.conf.d/50-server.cnf; if [ -f \"$CONF\" ]; then (grep -q \"^bind-address\" \"$CONF\" && sed -i \"s/^bind-address.*/bind-address = 0.0.0.0/\" \"$CONF\") || echo \"bind-address = 0.0.0.0\" >> \"$CONF\"; fi"';
                $commands[] = 'sudo systemctl restart mysql || sudo systemctl restart mariadb || true';

                $sql = "CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED WITH mysql_native_password BY '" . $mysqlRootPassword . "';";
                $sql .= "ALTER USER 'root'@'%' IDENTIFIED WITH mysql_native_password BY '" . $mysqlRootPassword . "';";
                $sql .= "GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION;";
                $sql .= "FLUSH PRIVILEGES;";

                $commands[] = 'sudo mysql -e ' . escapeshellarg($sql);
            }

            Log::info("Commands for {$component}", ['commands' => $commands]);

            $this->runSSHCommands($location, $commands, $cacheKey);

            LocationSetupStatus::updateOrCreate(
                ['location_id' => $location->id, 'component' => $component],
                ['status' => LocationSetupStatus::STATUS_INSTALLED, 'installed_at' => now()]
            );

            if ($component === 'daemon') {
                $this->refreshDaemonInfo($location);
            }

            if ($component === 'mysql') {
                $publicIp = (string) ($location->ip_address ?: $location->ssh_host);
                $data = [
                    'mysql_host' => $publicIp,
                    'mysql_port' => 3306,
                ];

                if ($mysqlRootUsername !== null && $mysqlRootPassword !== null) {
                    $data['mysql_root_username'] = $mysqlRootUsername;
                    $data['mysql_root_password'] = Crypt::encryptString($mysqlRootPassword);
                }

                $location->update($data);
            }

            if ($component === 'phpmyadmin') {
                $location->update([
                    'phpmyadmin_port' => 8081,
                ]);
            }

            $state = Cache::get($cacheKey, ['log' => '', 'component' => $component]);
            $state['log'] = (string) $state['log'] . "\n✅ {$component} установлен.";
            $state['completed'] = true;
            $state['component'] = $component;
            Cache::put($cacheKey, $state, self::SETUP_CACHE_TTL_SEC);
            Log::info("Cache final", ['key' => $cacheKey, 'value' => $state]);

            Log::info("InstallComponent completed", ['component' => $component]);

        } catch (\Exception $e) {
            LocationSetupStatus::updateOrCreate(
                ['location_id' => $location->id, 'component' => $component],
                ['status' => LocationSetupStatus::STATUS_FAILED, 'error_message' => $e->getMessage()]
            );

            $state = Cache::get($cacheKey, ['log' => '', 'component' => $component]);
            $state['log'] = (string) $state['log'] . "\n❌ Ошибка: " . $e->getMessage();
            $state['completed'] = true;
            $state['component'] = $component;
            Cache::put($cacheKey, $state, self::SETUP_CACHE_TTL_SEC);

            Log::error("InstallComponent failed: {$component}", ['error' => $e->getMessage()]);
        }
    }

    private function getCommandsForComponent(string $component): array
    {
        return match ($component) {
            'packages' => [
                'sudo apt update',
                'sudo apt install -y curl wget git htop vim nano ufw fail2ban unzip',
                'sudo dpkg --add-architecture i386',
                'sudo apt update',
                'sudo apt install -y libc6-i386 libc6:i386 lib32gcc-s1 libstdc++6:i386 libgcc-s1:i386 zlib1g:i386 libuuid1:i386',
                'sudo sh -c "apt-get install -y libtinfo6:i386 libncurses6:i386 2>/dev/null || apt-get install -y libtinfo5:i386 libncurses5:i386"',
                'sudo sh -c "apt-get install -y libcurl4t64:i386 2>/dev/null || apt-get install -y libcurl4:i386"',
            ],
            'docker' => [
                'curl -fsSL https://get.docker.com | sudo sh',
                'sudo systemctl enable docker',
                'sudo systemctl start docker',
            ],
            'mysql' => [
                'sudo apt install -y mysql-server',
                'sudo systemctl enable mysql',
                'sudo systemctl start mysql',
                'sudo ufw allow 3306/tcp',
                'sudo mysql -e "DELETE FROM mysql.user WHERE User=\'\'"',
                'sudo mysql -e "DROP DATABASE IF EXISTS test;"',
                'sudo mysql -e "DELETE FROM mysql.db WHERE Db=\'test\' OR Db LIKE \'test\\_%\';"',
                'sudo mysql -e "FLUSH PRIVILEGES;"',
                'echo "MySQL установлен, выполнена базовая настройка безопасности (удалены анонимные пользователи и тестовая БД)."',
            ],
            'phpmyadmin' => [
                'sudo systemctl start docker || true',
                'sudo docker rm -f vortanix-phpmyadmin 2>/dev/null || true',
                'sudo docker pull phpmyadmin/phpmyadmin:latest',
                'sudo docker run -d --name vortanix-phpmyadmin --restart unless-stopped --add-host=host.docker.internal:host-gateway -e PMA_HOST=host.docker.internal -e PMA_PORT=3306 -e UPLOAD_LIMIT=256M -p 8081:80 phpmyadmin/phpmyadmin:latest',
                'sudo ufw allow 8081/tcp',
                'echo "phpMyAdmin установлен и доступен на порту 8081"',
            ],
            'ftp' => [
                'sudo apt install -y vsftpd',
                'sudo systemctl enable vsftpd',
                'sudo systemctl start vsftpd',
                'sudo ufw allow 21',
                'sudo ufw allow 21100:21110/tcp',
            ],
            'daemon' => $this->getDaemonCommands(),
            'images' => $this->getImagesBuildCommands(),
            default => [],
        };
    }

    private function getRuntimeImages(): array
    {
        return [
            [
                'key' => 'samp',
                'env' => 'SAMP_DOCKER_IMAGE',
                'tag' => 'vortanix/samp:0.3.7-r3',
            ],
            [
                'key' => 'crmp',
                'env' => 'CRMP_DOCKER_IMAGE',
                'tag' => 'vortanix/crmp:latest',
            ],
            [
                'key' => 'cs16',
                'env' => 'CS16_DOCKER_IMAGE',
                'tag' => 'vortanix/cs16:latest',
            ],
            [
                'key' => 'tf2',
                'env' => 'TF2_DOCKER_IMAGE',
                'tag' => 'vortanix/tf2:latest',
            ],
            [
                'key' => 'gmod',
                'env' => 'GMOD_DOCKER_IMAGE',
                'tag' => 'vortanix/gmod:latest',
            ],
            [
                'key' => 'css',
                'env' => 'CSS_DOCKER_IMAGE',
                'tag' => 'vortanix/css:latest',
            ],
            [
                'key' => 'cs2',
                'env' => 'CS2_DOCKER_IMAGE',
                'tag' => 'vortanix/cs2:latest',
            ],
            [
                'key' => 'rust',
                'env' => 'RUST_DOCKER_IMAGE',
                'tag' => 'vortanix/rust:latest',
            ],
            [
                'key' => 'mta',
                'env' => 'MTA_DOCKER_IMAGE',
                'tag' => 'vortanix/mta:latest',
            ],
            [
                'key' => 'unturned',
                'env' => 'UNTURNED_DOCKER_IMAGE',
                'tag' => 'vortanix/unturned:latest',
            ],
        ];
    }

    private function getImagesBuildCommands(): array
    {
        $commands = [
            'sudo systemctl start docker || true',
            'sudo mkdir -p /opt/vortanix-daemon/docker || true',
        ];

        $runtimeImages = $this->getRuntimeImages();
        foreach ($runtimeImages as $image) {
            $dir = '/opt/vortanix-daemon/docker/' . $image['key'];
            $commands[] = 'test -d ' . $dir . ' || (echo "Missing ' . $dir . ' (run Daemon install first)" && exit 1)';
            $commands[] = 'sudo docker build -t ' . $image['tag'] . ' ' . $dir;
        }

        return $commands;
    }

    private function getDaemonCommands(): array
    {
        $runtimeImages = $this->getRuntimeImages();
        $daemonFiles = [
            [
                'src' => base_path('vortanix-daemon/vortanix-daemon.py'),
                'dest' => '/opt/vortanix-daemon/vortanix-daemon.py',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/samp/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/samp/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/samp/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/samp/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/crmp/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/crmp/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/crmp/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/crmp/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/cs16/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/cs16/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/cs16/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/cs16/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/unturned/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/unturned/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/unturned/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/unturned/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/tf2/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/tf2/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/tf2/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/tf2/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/gmod/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/gmod/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/gmod/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/gmod/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/mta/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/mta/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/mta/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/mta/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/css/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/css/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/css/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/css/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/cs2/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/cs2/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/cs2/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/cs2/entrypoint.sh',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/rust/Dockerfile'),
                'dest' => '/opt/vortanix-daemon/docker/rust/Dockerfile',
            ],
            [
                'src' => base_path('vortanix-daemon/docker/rust/entrypoint.sh'),
                'dest' => '/opt/vortanix-daemon/docker/rust/entrypoint.sh',
            ],
        ];

        $daemonModuleDir = base_path('vortanix-daemon/vtxdaemon');
        if (is_dir($daemonModuleDir)) {
            $moduleFiles = glob($daemonModuleDir . '/*.py') ?: [];
            sort($moduleFiles);
            foreach ($moduleFiles as $src) {
                $base = basename($src);
                $daemonFiles[] = [
                    'src' => $src,
                    'dest' => '/opt/vortanix-daemon/vtxdaemon/' . $base,
                ];
            }
        }

        $daemonContentCommands = [
            'sudo mkdir -p /opt/vortanix-daemon/vtxdaemon',
        ];

        foreach ($runtimeImages as $image) {
            $daemonContentCommands[] = 'sudo mkdir -p /opt/vortanix-daemon/docker/' . $image['key'];
        }

        foreach ($daemonFiles as $file) {
            if (! file_exists($file['src'])) {
                continue;
            }

            $content = file_get_contents($file['src']);
            if ($content === false) {
                continue;
            }

            $encoded = base64_encode($content);
            $daemonContentCommands[] = 'echo "' . $encoded . '" | base64 -d | sudo tee ' . $file['dest'] . ' > /dev/null';
        }

        $daemonContentCommands[] = 'sudo chmod +x /opt/vortanix-daemon/docker/*/entrypoint.sh 2>/dev/null || true';
        $daemonContentCommands[] = 'test -f /opt/vortanix-daemon/vtxdaemon/servers_common.py || (echo "Missing vtxdaemon/servers_common.py (daemon files incomplete)" && exit 1)';

        $serviceUnitLines = [
            '[Unit]',
            'Description=Vortanix Location Daemon',
            'After=network.target',
            '',
            '[Service]',
            'User=root',
            'WorkingDirectory=/opt/vortanix-daemon',
            'ExecStart=/opt/vortanix-daemon/venv/bin/python3 /opt/vortanix-daemon/vortanix-daemon.py',
            'Restart=always',
            'Environment=MONITORING_LOCATION_CODE=' . $this->location->code,
            'Environment=LOCATION_DAEMON_PORT=9201',
            'Environment=LOCATION_DAEMON_TOKEN=' . config('services.location_daemon.token'),
            'Environment=MONITORING_PANEL_URL=' . config('app.url'),
            'Environment=STATUS_REPORT_INTERVAL_SEC=20',
        ];
        foreach ($runtimeImages as $image) {
            $serviceUnitLines[] = 'Environment=' . $image['env'] . '=' . $image['tag'];
        }
        $serviceUnitLines[] = '';
        $serviceUnitLines[] = '[Install]';
        $serviceUnitLines[] = 'WantedBy=multi-user.target';

        $serviceUnit = implode("\n", $serviceUnitLines) . "\n";

        $serviceEncoded = base64_encode($serviceUnit);
        $serviceCommand = 'echo "' . $serviceEncoded . '" | base64 -d | sudo tee /etc/systemd/system/vortanix-daemon.service > /dev/null';

        return array_merge([
            'sudo apt install -y python3 python3-pip python3-venv',
            'sudo mkdir -p /opt/vortanix-daemon',
            'sudo chown root:root /opt/vortanix-daemon',
            'sudo python3 -m venv /opt/vortanix-daemon/venv',
            'sudo /opt/vortanix-daemon/venv/bin/pip install aiohttp',
        ], $daemonContentCommands, [
            ...array_map(function (array $image) {
                return 'sudo docker build -t ' . $image['tag'] . ' /opt/vortanix-daemon/docker/' . $image['key'];
            }, $runtimeImages),
            $serviceCommand,
            'sudo systemctl daemon-reload',
            'sudo systemctl enable vortanix-daemon',
            'sudo systemctl restart vortanix-daemon',
        ]);
    }

    private function runSSHCommands(Location $location, array $commands, string $cacheKey): void
    {
        $host = $location->ssh_host;
        $user = (string) $location->ssh_user;
        $port = (int) $location->ssh_port;
        $password = $location->ssh_password;
        $ssh = new SSH2($host, $port);

        if (!$ssh->login($user, $password)) {
            $lastError = $ssh->getLastError();
            Log::error('SSH login failed', [
                'host' => $host,
                'user' => $user,
                'port' => $port,
                'error' => $lastError
            ]);
            throw new \Exception('SSH login failed: ' . $lastError);
        }

        $ssh->setTimeout(0);

        foreach ($commands as $command) {
            Log::info("Executing command", ['command' => $command]);

            $this->appendSetupLog($cacheKey, "\n> $command");
            $this->updateSetupCache($cacheKey, ['completed' => false]);

            $marker = '__VTX_EXIT_CODE__:';
            $wrapped = 'bash -lc ' . escapeshellarg($command . '; echo "' . $marker . '$?"');
            $result = $ssh->exec($wrapped);
            if ($result === false) {
                $error = $ssh->getLastError();
                $this->appendSetupLog($cacheKey, "\n❌ Ошибка выполнения: $command ($error)");
                $this->updateSetupCache($cacheKey, ['completed' => false]);
                $ssh->disconnect();
                throw new \Exception("SSH command failed: $command ($error)");
            }

            $exitCode = null;
            $pos = strrpos($result, $marker);
            if ($pos !== false) {
                $codeRaw = trim(substr($result, $pos + strlen($marker)));
                $parts = preg_split('/\s+/', $codeRaw);
                $codeRaw = (string) $parts[0];
                if ($codeRaw !== '' && ctype_digit($codeRaw)) {
                    $exitCode = (int) $codeRaw;
                }
            }

            if ($exitCode !== null && $exitCode !== 0) {
                $this->appendSetupLog($cacheKey, "\n❌ Ошибка выполнения: $command (exit=$exitCode)");
                $this->updateSetupCache($cacheKey, ['completed' => false]);
                $ssh->disconnect();
                throw new \Exception("SSH command failed: $command (exit=$exitCode)");
            }

            $this->appendSetupLog($cacheKey, "\n$result");
            $this->updateSetupCache($cacheKey, ['completed' => false]);
            Log::info("Command result", ['command' => $command, 'result_length' => strlen($result)]);
        }

        $ssh->disconnect();
    }

    private function refreshDaemonInfo(Location $location): void
    {
        try {
            $port = (int) config('services.location_daemon.port', 9201);
            $token = (string) config('services.location_daemon.token', '');
            $url = sprintf('http://%s:%d/info', $location->ssh_host, $port);

            $request = Http::timeout(5);

            if ($token !== '') {
                $request = $request->withHeaders([
                    'X-Location-Daemon-Token' => $token,
                ]);
            }

            $response = $request->get($url);

            if ($response->successful()) {
                $data = $response->json();

                LocationDaemon::updateOrCreate(
                    ['location_id' => $location->id],
                    [
                        'status' => LocationDaemon::STATUS_ONLINE,
                        'version' => '1.0',
                        'pid' => $data['pid'],
                        'uptime_sec' => $data['uptime_sec'],
                        'platform' => $data['platform'],
                        'last_seen' => now(),
                    ]
                );
            } else {
                LocationDaemon::updateOrCreate(
                    ['location_id' => $location->id],
                    [
                        'status' => LocationDaemon::STATUS_OFFLINE,
                        'last_seen' => now(),
                    ]
                );
            }
        } catch (\Exception $e) {
            LocationDaemon::updateOrCreate(
                ['location_id' => $location->id],
                [
                    'status' => LocationDaemon::STATUS_UNKNOWN,
                    'last_seen' => now(),
                ]
            );
        }
    }
}
