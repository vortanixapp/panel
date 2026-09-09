import { UnauthorizedError } from "@/features/errors/unauthorized-error";

export default function Error401Page() {
  return <UnauthorizedError />;
}
