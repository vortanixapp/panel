<?php

namespace App\Dto\Daemon;

final class MetricsResponse
{
    /** @var array<MetricItem> */
    public array $items;

    private function __construct(array $items)
    {
        $this->items = $items;
    }

    public static function fromArray(array $data): self
    {
        $raw = array_key_exists('metrics', $data) && is_array($data['metrics']) ? (array) $data['metrics'] : [];
        $items = [];
        foreach ($raw as $metric) {
            if (! is_array($metric)) {
                continue;
            }
            $item = MetricItem::fromArray($metric);
            if (! $item) {
                continue;
            }
            $items[] = $item;
        }
        return new self($items);
    }
}
