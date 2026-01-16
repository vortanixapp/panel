<?php

namespace App\Services;

use App\Models\User;
use Illuminate\Database\Eloquent\Builder;

class MailingAudienceService
{
    public function buildUserQuery(array $audience): Builder
    {
        $q = User::query();

        if (array_key_exists('only_non_admin', $audience) && (bool) $audience['only_non_admin']) {
            $q->where('is_admin', false);
        }

        if (array_key_exists('only_admin', $audience) && (bool) $audience['only_admin']) {
            $q->where('is_admin', true);
        }

        if (array_key_exists('balance_min', $audience) && $audience['balance_min'] !== null && $audience['balance_min'] !== '') {
            $q->where('balance', '>=', (float) $audience['balance_min']);
        }

        if (array_key_exists('has_servers', $audience) && $audience['has_servers'] !== null) {
            if ((bool) $audience['has_servers']) {
                $q->whereHas('servers');
            } else {
                $q->whereDoesntHave('servers');
            }
        }

        if (! empty($audience['user_ids']) && is_array($audience['user_ids'])) {
            $ids = array_values(array_unique(array_map('intval', $audience['user_ids'])));
            $ids = array_values(array_filter($ids, fn ($v) => $v > 0));
            if (count($ids) > 0) {
                $q->whereIn('id', $ids);
            }
        }

        if (! empty($audience['emails']) && is_array($audience['emails'])) {
            $emails = array_values(array_unique(array_map('strval', $audience['emails'])));
            $emails = array_values(array_filter($emails, fn ($v) => trim($v) !== ''));
            if (count($emails) > 0) {
                $q->whereIn('email', $emails);
            }
        }

        return $q;
    }
}
