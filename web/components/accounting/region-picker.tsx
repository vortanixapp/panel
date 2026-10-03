"use client";

import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";

export function RegionPicker({
  value,
  options,
  label,
  note,
  allLabel,
  disabled,
  onChange,
}: {
  value: string[];
  options: string[];
  label: (option: string) => string;
  note?: (option: string) => string;
  allLabel: string;
  disabled?: boolean;
  onChange: (next: string[]) => void;
}) {
  const allChecked = options.every((option) => value.includes(option));

  const toggle = (option: string, checked: boolean) => {
    const next = checked ? [...value, option] : value.filter((item) => item !== option);
    onChange(options.filter((item) => next.includes(item)));
  };

  const row = (checked: boolean, title: string, description: string | undefined, onToggle: (next: boolean) => void) => (
    <label
      key={title}
      className={cn(
        "flex cursor-pointer items-start gap-3 rounded-lg border px-3.5 py-2.5 transition-colors",
        checked ? "border-primary/50 bg-primary/5" : "hover:bg-accent/40",
        disabled && "cursor-not-allowed opacity-60"
      )}
    >
      <Checkbox
        className="mt-0.5"
        checked={checked}
        disabled={disabled}
        onCheckedChange={(next) => onToggle(next === true)}
      />
      <span className="min-w-0">
        <span className="block text-sm font-medium">{title}</span>
        {description ? (
          <span className="block text-xs text-muted-foreground">{description}</span>
        ) : null}
      </span>
    </label>
  );

  return (
    <div className="grid gap-2">
      {row(allChecked, allLabel, undefined, (next) => onChange(next ? [...options] : []))}
      {options.map((option) =>
        row(value.includes(option), label(option), note?.(option), (next) => toggle(option, next))
      )}
    </div>
  );
}
