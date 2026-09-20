"use client";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { accountInitials } from "@/lib/accounts";
import { cn } from "@/lib/utils";

type UserAvatarProps = {
  email: string;
  name?: string | null;
  src?: string | null;
  className?: string;
  fallbackClassName?: string;
};

export function UserAvatar({
  email,
  name,
  src,
  className,
  fallbackClassName,
}: UserAvatarProps) {
  const initials = accountInitials(email, name);
  const alt = (name ?? "").trim() || email;

  return (
    <Avatar className={className}>
      {src ? <AvatarImage src={src} alt={alt} /> : null}
      <AvatarFallback className={cn("uppercase", fallbackClassName)}>
        {initials}
      </AvatarFallback>
    </Avatar>
  );
}
