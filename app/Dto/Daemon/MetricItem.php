<?php

namespace App\Dto\Daemon;

final class MetricItem
{
    public string $type;

    public mixed $value;

    public mixed $measuredAt;

    private function __construct(string $type, mixed $value, mixed $measuredAt)
    {
        $this->type = $type;
        $this->value = $value;
        $this->measuredAt = $measuredAt;
    }

    public static function fromArray(array $data): ?self
    {
        if (! array_key_exists('metric_type', $data) || ! array_key_exists('value', $data)) {
            return null;
        }

        $type = (string) $data['metric_type'];
        if ($type === '') {
            return null;
        }

        $value = $data['value'];
        $measuredAt = array_key_exists('measured_at', $data) ? $data['measured_at'] : null;

        return new self($type, $value, $measuredAt);
    }
}
