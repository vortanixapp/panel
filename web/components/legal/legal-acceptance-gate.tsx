"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { acceptAccountLegal, clearAuth, fetchAccountLegal } from "@/lib/api";
import { useT } from "@/hooks/use-translations";

const KEY = ["account-legal"];

export function LegalAcceptanceGate() {
  const t = useT();
  const qc = useQueryClient();
  const router = useRouter();
  const [checked, setChecked] = useState<Record<string, boolean>>({});
  const { data } = useQuery({
    queryKey: KEY,
    queryFn: fetchAccountLegal,
    staleTime: 60_000,
    retry: false,
  });
  const pending = data?.pending ?? [];

  const acceptMut = useMutation({
    mutationFn: () => acceptAccountLegal(pending.map((doc) => doc.kind)),
    onSuccess: (res) => {
      qc.setQueryData(KEY, res);
      setChecked({});
    },
    onError: (e: Error) => toast.error(e.message || t("legal.gate.failed")),
  });

  if (pending.length === 0) return null;

  const ready = pending.every((doc) => checked[doc.kind]);

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-background/80 p-4 backdrop-blur-sm">
      <div
        role="dialog"
        aria-modal="true"
        className="w-full max-w-lg rounded-2xl border border-border bg-card p-6 shadow-2xl"
      >
        <h2 className="text-lg font-semibold">{t("legal.gate.title")}</h2>
        <p className="mt-2 text-sm text-muted-foreground">{t("legal.gate.text")}</p>
        <div className="mt-5 space-y-3">
          {pending.map((doc) => (
            <label key={doc.kind} className="flex cursor-pointer items-start gap-3 text-sm">
              <Checkbox
                className="mt-0.5"
                checked={Boolean(checked[doc.kind])}
                onCheckedChange={(value) =>
                  setChecked((prev) => ({ ...prev, [doc.kind]: value === true }))
                }
              />
              <span>
                {t(`legal.gate.accept_${doc.kind}`)}{" "}
                <a
                  href={`/legal/${doc.kind}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary hover:underline"
                >
                  {doc.title}
                </a>{" "}
                <span className="text-muted-foreground">
                  ({t("legal.gate.version", { version: doc.version })})
                </span>
              </span>
            </label>
          ))}
        </div>
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <Button
            variant="outline"
            onClick={() => {
              clearAuth();
              router.replace("/login");
            }}
          >
            {t("legal.gate.logout")}
          </Button>
          <Button disabled={!ready || acceptMut.isPending} onClick={() => acceptMut.mutate()}>
            {t("legal.gate.accept")}
          </Button>
        </div>
      </div>
    </div>
  );
}
