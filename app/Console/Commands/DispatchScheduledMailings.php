<?php

namespace App\Console\Commands;

use App\Jobs\PrepareMailingRecipients;
use App\Models\Mailing;
use Illuminate\Console\Command;

class DispatchScheduledMailings extends Command
{
    protected $signature = 'mailings:dispatch-scheduled';

    protected $description = 'Отправка рассылки в очередь.';

    public function handle(): int
    {
        $items = Mailing::query()
            ->where('status', 'scheduled')
            ->whereNotNull('scheduled_at')
            ->where('scheduled_at', '<=', now())
            ->orderBy('scheduled_at')
            ->limit(20)
            ->get();

        foreach ($items as $m) {
            PrepareMailingRecipients::dispatch((int) $m->id);
        }

        return self::SUCCESS;
    }
}
