import { redirect } from "next/navigation";

// Отдельной вкладки «Экран» нет: её содержимое давно живёт во «Внешнем виде».
export default function Page() {
  redirect("/settings?tab=appearance");
}
