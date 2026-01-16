<?php

namespace App\Jobs;

use App\Models\Mailing;
use App\Models\MailingDelivery;
use App\Services\MailingAudienceService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\DB;

class PrepareMailingRecipients implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $mailingId;

    public function __construct(int $mailingId)
    {
        $this->mailingId = $mailingId;
    }

    public function handle(MailingAudienceService $audienceService): void
    {
        $mailing = Mailing::query()->find($this->mailingId);
        if (! $mailing) {
            return;
        }

        if (! in_array((string) $mailing->status, ['draft', 'scheduled', 'queued'], true)) {
            return;
        }

        $channels = is_array($mailing->channels) ? $mailing->channels : [];
        $channels = array_values(array_unique(array_map('strval', $channels)));
        $channels = array_values(array_filter($channels, fn ($c) => in_array($c, ['email', 'internal', 'telegram', 'sms'], true)));
        if (count($channels) === 0) {
            $channels = ['email'];
        }

        DB::transaction(function () use ($mailing) {
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $m) {
                return;
            }
            if (! in_array((string) $m->status, ['draft', 'scheduled', 'queued'], true)) {
                return;
            }

            $m->update([
                'status' => 'sending',
                'started_at' => $m->started_at ?: now(),
                'finished_at' => null,
                'sent_count' => 0,
                'failed_count' => 0,
                'skipped_count' => 0,
                'last_error' => null,
            ]);

            MailingDelivery::query()->where('mailing_id', (int) $m->id)->delete();
        });

        $audience = is_array($mailing->audience) ? $mailing->audience : [];
        $query = $audienceService->buildUserQuery($audience)->orderBy('id');

        $total = 0;
        $query->chunkById(500, function ($users) use ($mailing, $channels, &$total) {
            $rows = [];
            $now = now();

            foreach ($users as $u) {
                foreach ($channels as $ch) {
                    $addr = null;
                    if ($ch === 'email') {
                        $addr = (string) $u->email;
                        if ($addr === '') {
                            continue;
                        }
                    }

                    $rows[] = [
                        'mailing_id' => (int) $mailing->id,
                        'user_id' => (int) $u->id,
                        'channel' => $ch,
                        'address' => $addr,
                        'status' => 'queued',
                        'attempts' => 0,
                        'sent_at' => null,
                        'error' => null,
                        'created_at' => $now,
                        'updated_at' => $now,
                    ];
                }
            }

            if (count($rows) > 0) {
                MailingDelivery::query()->insert($rows);
                $total += count($rows);
            }
        });

        Mailing::whereKey((int) $mailing->id)->update([
            'total_recipients' => $total,
        ]);

        SendMailingBatch::dispatch((int) $mailing->id);
    }
}
