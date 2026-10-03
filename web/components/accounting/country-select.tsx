"use client";

import { useMemo, useState } from "react";
import { Check, ChevronsUpDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useT } from "@/hooks/use-translations";
import { localeTag } from "@/lib/i18n";
import { countryName, sortedCountries } from "@/lib/countries";
import { cn } from "@/lib/utils";

export function CountrySelect({
  value,
  onChange,
  disabled,
  id,
  invalid,
}: {
  value: string;
  onChange: (code: string) => void;
  disabled?: boolean;
  id?: string;
  invalid?: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const tag = localeTag();
  const countries = useMemo(() => sortedCountries(tag), [tag]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          type="button"
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-invalid={invalid || undefined}
          disabled={disabled}
          className={cn(
            "h-9 w-full justify-between px-3 font-normal",
            !value && "text-muted-foreground",
            invalid && "border-destructive"
          )}
        >
          <span className="truncate">
            {value ? countryName(value, tag) : t("profile.country_placeholder")}
          </span>
          <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[min(calc(100vw-2rem),340px)] p-0" align="start">
        <Command>
          <CommandInput placeholder={t("profile.country_search")} />
          <CommandList className="max-h-72">
            <CommandEmpty>{t("profile.country_empty")}</CommandEmpty>
            <CommandGroup>
              {countries.map((country) => (
                <CommandItem
                  key={country.code}
                  value={`${country.name} ${country.code}`}
                  onSelect={() => {
                    onChange(country.code);
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn("size-4", value === country.code ? "opacity-100" : "opacity-0")}
                  />
                  <span className="truncate">{country.name}</span>
                  <span className="ms-auto text-xs text-muted-foreground">{country.code}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
