import { adminContent } from "./admin-content";
import { adminInfra } from "./admin-infra";
import { auth } from "./auth";
import { billing } from "./billing";
import { common } from "./common";
import { dashboard } from "./dashboard";
import { errors } from "./errors";
import { landing } from "./landing";
import { layout } from "./layout";
import { nav } from "./nav";
import { news } from "./news";
import { servers } from "./servers";
import { settings } from "./settings";
import { support } from "./support";

export const RU = {
  ...common,
  ...nav,
  ...layout,
  ...auth,
  ...errors,
  ...landing,
  ...servers,
  ...billing,
  ...support,
  ...news,
  ...settings,
  ...dashboard,
  ...adminInfra,
  ...adminContent,
};
