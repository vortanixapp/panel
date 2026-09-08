import { redirect } from "next/navigation";

// Настройки — одна страница с вкладками в ?tab=. Без вкладки в адресе редирект
// открывал «Аккаунт», и пункт меню «Внешний вид» вёл не туда, куда обещал.
export default function Page() {
  redirect("/settings?tab=appearance");
}
