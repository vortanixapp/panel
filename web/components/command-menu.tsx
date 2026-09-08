"use client";

import React from "react";
import { usePathname, useRouter } from "next/navigation";
import {
  ArrowRight,
  ChevronRight,
  Laptop,
  Moon,
  Server,
  Sun,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { useSearch } from "@/context/search-provider";
import { adminSearch } from "@/lib/api";

import { useTheme } from "@/context/theme-provider";
import { useServers } from "@/hooks/use-queries";
import { getNavGroupsForVariant } from "@/lib/nav";
import {
  isAdminSection,
  serverPath,
  variantToBasePath,
} from "@/lib/panel-paths";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useT } from "@/hooks/use-translations";

export function CommandMenu() {
  const t = useT();
  const router = useRouter();
  const pathname = usePathname();
  const { setTheme } = useTheme();
  const { open, setOpen } = useSearch();
  const { data: servers = [] } = useServers();
  const [query, setQuery] = React.useState("");

  const variant = isAdminSection(pathname) ? "admin" : "user";
  const basePath = variantToBasePath(variant);

  const navGroups = React.useMemo(
    () => getNavGroupsForVariant(variant),
    [variant]
  );

  const { data: hits } = useQuery({
    queryKey: ["admin-search", query],
    queryFn: () => adminSearch(query),
    enabled: variant === "admin" && query.trim().length >= 2,
    staleTime: 10_000,
  });

  const runCommand = React.useCallback(
    (command: () => void) => {
      setOpen(false);
      command();
    },
    [setOpen]
  );

  return (
    <CommandDialog modal open={open} onOpenChange={setOpen}>
      <CommandInput
        placeholder={
          variant === "admin"
            ? t("layout.search.placeholder_admin")
            : t("layout.search.placeholder_user")
        }
        value={query}
        onValueChange={setQuery}
      />
      <CommandList>
        <ScrollArea type="hover" className="h-72 pe-1">
          <CommandEmpty>{t("common.not_found")}</CommandEmpty>
          {(hits?.results.length ?? 0) > 0 && (
            <CommandGroup heading={t("layout.search.found_in_panel")}>
              {(hits?.results ?? []).map((hit) => (
                <CommandItem
                  key={`${hit.kind}-${hit.id}`}
                  value={`${hit.title} ${hit.subtitle}`}
                  onSelect={() => runCommand(() => router.push(hit.url))}
                >
                  <div className="flex min-w-0 flex-col">
                    <span className="truncate">{hit.title}</span>
                    <span className="truncate text-xs text-muted-foreground">
                      {t(`layout.search.kind.${hit.kind}`)} · {hit.subtitle}
                    </span>
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          )}
          {navGroups.map((group) => (
            <CommandGroup key={group.title} heading={group.title}>
              {group.items.map((navItem, i) => {
                if ("url" in navItem && navItem.url)
                  return (
                    <CommandItem
                      key={`${navItem.url}-${i}`}
                      value={navItem.title}
                      onSelect={() => {
                        runCommand(() => router.push(navItem.url));
                      }}
                    >
                      <div className="flex size-4 items-center justify-center">
                        <ArrowRight className="size-2 text-muted-foreground/80" />
                      </div>
                      {navItem.title}
                    </CommandItem>
                  );

                return navItem.items?.map((subItem, j) => (
                  <CommandItem
                    key={`${navItem.title}-${subItem.url}-${j}`}
                    value={`${navItem.title} ${subItem.title}`}
                    onSelect={() => {
                      runCommand(() => router.push(subItem.url));
                    }}
                  >
                    <div className="flex size-4 items-center justify-center">
                      <ArrowRight className="size-2 text-muted-foreground/80" />
                    </div>
                    {navItem.title} <ChevronRight /> {subItem.title}
                  </CommandItem>
                ));
              })}
            </CommandGroup>
          ))}
          {servers.length > 0 && (
            <>
              <CommandSeparator />
              <CommandGroup heading={t("common.servers")}>
                {servers.slice(0, 12).map((server) => (
                  <CommandItem
                    key={server.id}
                    value={`${t("layout.search.server_token")} ${server.name}`}
                    onSelect={() => {
                      runCommand(() =>
                        router.push(serverPath(basePath, server.id))
                      );
                    }}
                  >
                    <Server className="size-4 text-muted-foreground" />
                    {server.name}
                  </CommandItem>
                ))}
              </CommandGroup>
            </>
          )}
          <CommandSeparator />
          <CommandGroup heading={t("layout.appearance.theme")}>
            <CommandItem onSelect={() => runCommand(() => setTheme("light"))}>
              <Sun /> <span>{t("layout.theme.light")}</span>
            </CommandItem>
            <CommandItem onSelect={() => runCommand(() => setTheme("dark"))}>
              <Moon className="scale-90" />
              <span>{t("layout.theme.dark")}</span>
            </CommandItem>
            <CommandItem onSelect={() => runCommand(() => setTheme("system"))}>
              <Laptop />
              <span>{t("layout.theme.system")}</span>
            </CommandItem>
          </CommandGroup>
        </ScrollArea>
      </CommandList>
    </CommandDialog>
  );
}
