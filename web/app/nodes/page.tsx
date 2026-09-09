import { redirect } from "next/navigation";

export default function NodesRedirectPage() {
  redirect("/admin/locations");
}
