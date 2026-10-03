"use client";

import { CountrySelect } from "@/components/accounting/country-select";
import { Input } from "@/components/ui/input";
import { useT } from "@/hooks/use-translations";

export type ProfileAddress = {
  middle_name: string;
  country: string;
  address_line: string;
  city: string;
  region: string;
  postal_code: string;
};

export const EMPTY_ADDRESS: ProfileAddress = {
  middle_name: "",
  country: "",
  address_line: "",
  city: "",
  region: "",
  postal_code: "",
};

export function addressOf(user: {
  middle_name?: string | null;
  country?: string | null;
  address_line?: string | null;
  city?: string | null;
  region?: string | null;
  postal_code?: string | null;
}): ProfileAddress {
  return {
    middle_name: user.middle_name ?? "",
    country: user.country ?? "",
    address_line: user.address_line ?? "",
    city: user.city ?? "",
    region: user.region ?? "",
    postal_code: user.postal_code ?? "",
  };
}

function FieldBox({
  label,
  htmlFor,
  children,
}: {
  label: string;
  htmlFor?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="text-[13px] font-medium">
        {label}
      </label>
      {children}
    </div>
  );
}

export function AddressFields({
  value,
  onChange,
  disabled,
  withMiddleName = true,
  countryInvalid,
}: {
  value: ProfileAddress;
  onChange: (next: ProfileAddress) => void;
  disabled?: boolean;
  withMiddleName?: boolean;
  countryInvalid?: boolean;
}) {
  const t = useT();
  const set = (key: keyof ProfileAddress) => (e: React.ChangeEvent<HTMLInputElement>) =>
    onChange({ ...value, [key]: e.target.value });

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {withMiddleName && (
        <FieldBox label={t("profile.middle_name")} htmlFor="profile-middle-name">
          <Input
            id="profile-middle-name"
            value={value.middle_name}
            onChange={set("middle_name")}
            disabled={disabled}
            maxLength={200}
            autoComplete="additional-name"
          />
        </FieldBox>
      )}
      <FieldBox label={t("profile.country")} htmlFor="profile-country">
        <CountrySelect
          id="profile-country"
          value={value.country}
          onChange={(country) => onChange({ ...value, country })}
          disabled={disabled}
          invalid={countryInvalid}
        />
      </FieldBox>
      <FieldBox label={t("profile.address_line")} htmlFor="profile-address">
        <Input
          id="profile-address"
          value={value.address_line}
          onChange={set("address_line")}
          disabled={disabled}
          maxLength={200}
          autoComplete="street-address"
        />
      </FieldBox>
      <FieldBox label={t("profile.city")} htmlFor="profile-city">
        <Input
          id="profile-city"
          value={value.city}
          onChange={set("city")}
          disabled={disabled}
          maxLength={200}
          autoComplete="address-level2"
        />
      </FieldBox>
      <FieldBox label={t("profile.region")} htmlFor="profile-region">
        <Input
          id="profile-region"
          value={value.region}
          onChange={set("region")}
          disabled={disabled}
          maxLength={200}
          autoComplete="address-level1"
        />
      </FieldBox>
      <FieldBox label={t("profile.postal_code")} htmlFor="profile-postal">
        <Input
          id="profile-postal"
          value={value.postal_code}
          onChange={set("postal_code")}
          disabled={disabled}
          maxLength={20}
          autoComplete="postal-code"
        />
      </FieldBox>
    </div>
  );
}
