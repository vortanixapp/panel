import { NotFoundError } from "@/features/errors/not-found-error";

export const dynamic = "force-dynamic";

export default function Error404Page() {
  return <NotFoundError />;
}
