<?php

namespace App\Jobs;

use App\Mail\MailingMessage;
use App\Models\Mailing;
use App\Models\MailingDelivery;
use App\Models\UserNotification;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Mail;

class SendMailingDelivery implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public int $deliveryId;

    public function __construct(int $deliveryId)
    {
        $this->deliveryId = $deliveryId;
    }

    public function handle(): void
    {
        $delivery = MailingDelivery::query()->find($this->deliveryId);
        if (! $delivery) {
            return;
        }

        if ((string) $delivery->status !== 'queued') {
            return;
        }

        $mailing = Mailing::query()->find((int) $delivery->mailing_id);
        if (! $mailing || (string) $mailing->status !== 'sending') {
            return;
        }

        $channel = (string) $delivery->channel;

        try {
            if ($channel === 'email') {
                $addr = (string) $delivery->address;
                if ($addr === '') {
                    $this->markSkipped($delivery, $mailing, 'no address');
                    return;
                }

                $subject = (string) $mailing->subject;
                $body = (string) $mailing->body;
                $isHtml = (bool) $mailing->is_html;

                Mail::to($addr)->send(new MailingMessage($subject, $body, $isHtml));
                $this->markSent($delivery, $mailing);
                return;
            }

            if ($channel === 'internal') {
                if (! $delivery->user_id) {
                    $this->markSkipped($delivery, $mailing, 'no user');
                    return;
                }

                UserNotification::create([
                    'user_id' => (int) $delivery->user_id,
                    'title' => (string) $mailing->subject,
                    'body' => (string) $mailing->body,
                    'read_at' => null,
                ]);

                $this->markSent($delivery, $mailing);
                return;
            }

            $this->markSkipped($delivery, $mailing, 'channel not configured');
        } catch (\Throwable $e) {
            $this->markFailed($delivery, $mailing, $e->getMessage());
        }
    }

    private function markSent(MailingDelivery $delivery, Mailing $mailing): void
    {
        DB::transaction(function () use ($delivery, $mailing) {
            $d = MailingDelivery::whereKey((int) $delivery->id)->lockForUpdate()->first();
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $d || ! $m) {
                return;
            }
            if ((string) $d->status !== 'queued') {
                return;
            }

            $d->update([
                'status' => 'sent',
                'sent_at' => now(),
                'attempts' => (int) $d->attempts + 1,
                'error' => null,
            ]);
            $m->increment('sent_count');
        });
    }

    private function markSkipped(MailingDelivery $delivery, Mailing $mailing, string $reason): void
    {
        DB::transaction(function () use ($delivery, $mailing, $reason) {
            $d = MailingDelivery::whereKey((int) $delivery->id)->lockForUpdate()->first();
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $d || ! $m) {
                return;
            }
            if ((string) $d->status !== 'queued') {
                return;
            }

            $d->update([
                'status' => 'skipped',
                'sent_at' => now(),
                'attempts' => (int) $d->attempts + 1,
                'error' => $reason,
            ]);
            $m->increment('skipped_count');
        });
    }

    private function markFailed(MailingDelivery $delivery, Mailing $mailing, string $error): void
    {
        DB::transaction(function () use ($delivery, $mailing, $error) {
            $d = MailingDelivery::whereKey((int) $delivery->id)->lockForUpdate()->first();
            $m = Mailing::whereKey((int) $mailing->id)->lockForUpdate()->first();
            if (! $d || ! $m) {
                return;
            }
            if ((string) $d->status !== 'queued') {
                return;
            }

            $d->update([
                'status' => 'failed',
                'sent_at' => now(),
                'attempts' => (int) $d->attempts + 1,
                'error' => $error,
            ]);
            $m->increment('failed_count');
            $m->update([
                'last_error' => $error,
            ]);
        });
    }
}
