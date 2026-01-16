<?php

namespace Database\Seeders;

use App\Models\Location;
use Illuminate\Database\Seeder;

class LocationSeeder extends Seeder
{
    public function run(): void
    {
        $locations = [
            [
                'code' => 'eu-frankfurt',
                'name' => 'Европа — Франкфурт',
                'country' => 'Германия',
                'city' => 'Франкфурт',
                'region' => 'Европа',
                'description' => 'Низкие задержки для игроков из ЕС и РФ, дата‑центры уровня Tier III.',
                'sort_order' => 10,
                'ssh_host' => '192.0.2.10',
                'ssh_user' => 'gameadmin',
                'ssh_password' => 'changeMe!',
                'ssh_port' => 22,
            ],
            [
                'code' => 'ru-moscow',
                'name' => 'Россия — Москва',
                'country' => 'Россия',
                'city' => 'Москва',
                'region' => 'Россия',
                'description' => 'Оптимальный пинг для игроков из СНГ, защищённый периметр и DDoS‑фильтрация.',
                'sort_order' => 20,
                'ssh_host' => '192.0.2.20',
                'ssh_user' => 'gameadmin',
                'ssh_password' => 'changeMe!',
                'ssh_port' => 22,
            ],
            [
                'code' => 'asia-singapore',
                'name' => 'Азия — Сингапур',
                'country' => 'Сингапур',
                'city' => 'Сингапур',
                'region' => 'Азия',
                'description' => 'Хороший выбор для международных проектов с аудиторией в Азии и Океании.',
                'sort_order' => 30,
                'ssh_host' => '192.0.2.30',
                'ssh_user' => 'gameadmin',
                'ssh_password' => 'changeMe!',
                'ssh_port' => 22,
            ],
        ];

        foreach ($locations as $data) {
            Location::updateOrCreate(
                ['code' => $data['code']],
                $data
            );
        }
    }
}
