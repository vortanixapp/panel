import { GeneralError } from "@/features/errors/general-error";

export const dynamic = "force-dynamic";

export default function Error500Page() {
  return <GeneralError />;
}
