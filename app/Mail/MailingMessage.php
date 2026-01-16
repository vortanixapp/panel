<?php

namespace App\Mail;

use Illuminate\Bus\Queueable;
use Illuminate\Mail\Mailable;
use Illuminate\Queue\SerializesModels;

class MailingMessage extends Mailable
{
    use Queueable, SerializesModels;

    public string $body;

    public bool $isHtml;

    public function __construct(string $subject, string $body, bool $isHtml)
    {
        $this->subject($subject);
        $this->body = $body;
        $this->isHtml = $isHtml;
    }

    public function build(): self
    {
        if ($this->isHtml) {
            return $this->html($this->body);
        }

        return $this->text('mailing.plain')->with([
            'body' => $this->body,
        ]);
    }
}
