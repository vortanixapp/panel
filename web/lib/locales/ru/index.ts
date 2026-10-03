import { adminAgents } from "./admin-agents";
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
import { panelTransfer } from "./panel-transfer";
import { projects } from "./projects";
import { registry } from "./registry";
import { registryBilling } from "./registry-billing";
import { registryNotifications } from "./registry-notifications";
import { registryServers } from "./registry-servers";
import { news } from "./news";
import { notifications } from "./notifications";
import { servers } from "./servers";
import { settings } from "./settings";
import { support } from "./support";
import { template } from "./template";

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
  ...notifications,
  ...settings,
  ...dashboard,
  ...adminInfra,
  ...adminAgents,
  ...adminContent,
  ...panelTransfer,
  ...projects,
  ...registry,
  ...registryBilling,
  ...registryServers,
  ...registryNotifications,
  ...template,
};
