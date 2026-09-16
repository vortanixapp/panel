import {
  Briefcase,
  LifeBuoy,
  Megaphone,
  Server,
  ShieldCheck,
  Wallet,
  type LucideIcon,
} from "lucide-react";

const GROUP_ICONS: Record<string, LucideIcon> = {
  servers: Server,
  billing: Wallet,
  support: LifeBuoy,
  security: ShieldCheck,
  system: Megaphone,
  staff: Briefcase,
};

export function groupIcon(id: string): LucideIcon {
  return GROUP_ICONS[id] ?? Megaphone;
}
