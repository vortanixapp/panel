<?php

namespace App\Jobs;

use App\Models\Mailing;
use App\Models\MailingDelivery;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;

class SendMailingBatch implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $mailingId;

    public function __construct(int $mailingId)
    {
        $this->mailingId = $mailingId;
    }

    public function handle(): void
    {
        $mailing = Mailing::query()->find($this->mailingId);
        if (! $mailing) {
            return;
        }

        if ((string) $mailing->status !== 'sending') {
            return;
        }

        $ids = MailingDelivery::query()
            ->where('mailing_id', (int) $mailing->id)
            ->where('status', 'queued')
            ->orderBy('id')
            ->limit(500)
            ->pluck('id')
            ->all();

        if (count($ids) === 0) {
            $mailing->update([
                'status' => 'sent',
                'finished_at' => now(),
            ]);
            return;
        }

        foreach ($ids as $id) {
            SendMailingDelivery::dispatch((int) $id);
        }

        self::dispatch((int) $mailing->id)->delay(now()->addSeconds(1));
    }
}
