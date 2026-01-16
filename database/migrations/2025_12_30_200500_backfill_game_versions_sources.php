<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('game_versions')) {
            return;
        }

        if (! Schema::hasColumn('game_versions', 'url')) {
            return;
        }

        if (Schema::hasColumn('game_versions', 'source_type') && Schema::hasColumn('game_versions', 'steam_app_id')) {
            $rows = DB::table('game_versions')
                ->select(['id', 'url'])
                ->whereNotNull('url')
                ->where(function ($q) {
                    $q->whereNull('source_type')->orWhere('source_type', '');
                })
                ->get();

            foreach ($rows as $row) {
                $url = trim((string) ($row->url ?? ''));
                if ($url === '') {
                    continue;
                }

                if (preg_match('/^steam\s*:\s*(\d+)\s*$/i', $url, $m)) {
                    $appId = (int) $m[1];
                    if ($appId > 0) {
                        DB::table('game_versions')->where('id', $row->id)->update([
                            'source_type' => 'steam',
                            'steam_app_id' => $appId,
                        ]);
                    }
                }
            }
        }

        if (Schema::hasColumn('game_versions', 'archive_url')) {
            DB::table('game_versions')
                ->whereNull('archive_url')
                ->whereNotNull('url')
                ->where(function ($q) {
                    $q->whereNull('source_type')->orWhere('source_type', '')->orWhere('source_type', 'archive');
                })
                ->update([
                    'archive_url' => DB::raw('`url`'),
                ]);
        }

        if (Schema::hasColumn('game_versions', 'source_type') && Schema::hasColumn('game_versions', 'archive_url')) {
            DB::table('game_versions')
                ->where(function ($q) {
                    $q->whereNull('source_type')->orWhere('source_type', '');
                })
                ->whereNotNull('archive_url')
                ->update([
                    'source_type' => 'archive',
                ]);
        }
    }

    public function down(): void
    {
        // 
    }
};
