import { redirect } from "next/navigation";

export default function AdminAppearanceRedirectPage() {
  redirect("/admin/settings/appearance");
}
