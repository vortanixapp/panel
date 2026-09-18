import { API_URL, apiFetch, authHeaders, ensureValidSession } from "@/lib/api";
import type { SiteDocument, SiteMenu } from "@/lib/site/types";

export type TemplateStatus = {
  revision: number;
  changed: boolean;
  pending_texts: number;
  draft_updated_at: string | null;
  draft_updated_by: string;
  published_at: string | null;
  published_by: string;
};

export type TemplateState = TemplateStatus & {
  draft: SiteDocument;
  published: SiteDocument;
};

export type TemplateVersion = {
  id: number;
  created_at: string;
  author: string;
  note: string;
  texts: number;
};

export function fetchAdminTemplate() {
  return apiFetch<TemplateState>("/v1/admin/template");
}

export function saveTemplateDraft(document: SiteDocument, revision: number, force = false) {
  return apiFetch<TemplateStatus>("/v1/admin/template/draft", {
    method: "PUT",
    body: JSON.stringify({ document, revision, force }),
  });
}

export function publishTemplate(note: string, revision: number) {
  return apiFetch<TemplateState>("/v1/admin/template/publish", {
    method: "POST",
    body: JSON.stringify({ note, revision }),
  });
}

export function discardTemplate() {
  return apiFetch<TemplateState>("/v1/admin/template/discard", { method: "POST" });
}

export function fetchTemplateVersions() {
  return apiFetch<{ items: TemplateVersion[] }>("/v1/admin/template/versions");
}

export function restoreTemplateVersion(id: number) {
  return apiFetch<TemplateState>(`/v1/admin/template/versions/${id}/restore`, { method: "POST" });
}

export function fetchAdminSiteMenu() {
  return apiFetch<{ menu: SiteMenu | null }>("/v1/admin/site-menu");
}

export async function uploadTemplateAsset(file: File): Promise<{ path: string; url: string }> {
  await ensureValidSession();
  const form = new FormData();
  form.append("file", file);
  const res = await fetch(`${API_URL}/v1/admin/template/assets`, {
    method: "POST",
    headers: authHeaders(),
    credentials: "include",
    body: form,
  });
  const data = (await res.json().catch(() => ({}))) as {
    path?: string;
    url?: string;
    error?: string;
    message?: string;
  };
  if (!res.ok || !data.path) {
    throw new Error(data.error ?? data.message ?? "Request failed");
  }
  return { path: data.path, url: data.url ?? "" };
}
