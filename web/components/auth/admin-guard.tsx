"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useMe } from "@/hooks/use-queries";
import { isStaffRole } from "@/lib/rbac";

// Гвард отвечает ровно на один вопрос: штатная ли роль. Сессией занимается
// DashboardLayout ниже — он показывает свой лоадер, обновляет протухший токен и
// уводит на /login, если сессии нет.
//
// Раньше гвард сам рисовал лоадер при `isLoading || !me` — и тем самым не давал
// DashboardLayout смонтироваться. А `me` остаётся пустым не только пока идёт
// загрузка: запрос отключён, когда токенов нет вовсе (useMe: enabled), и
// обнуляется при ошибке. В обоих случаях isLoading равен false (в React Query
// это isPending && isFetching), ветки ошибки у гварда не было, редирект на
// /login выполнить было некому — страница крутилась вечно.
//
// Поэтому пока роль неизвестна, гвард пропускает управление вниз и ничего не
// решает.
export function AdminGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { data: me } = useMe();

  useEffect(() => {
    if (me && !isStaffRole(me.role)) {
      router.replace("/dashboard");
    }
  }, [me, router]);

  if (me && !isStaffRole(me.role)) {
    return null;
  }

  return <>{children}</>;
}
