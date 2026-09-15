# WHMCS integration

[English](whmcs.md) · [Русский](whmcs.ru.md)

The panel connects to WHMCS as a provisioning module. WHMCS takes orders and
payments, the panel creates and runs the game servers.

## Features

| Event in WHMCS | Action in the panel |
|---|---|
| Service created | A server is created from a panel plan. The client gets a new panel account or the existing one with the same email |
| Suspended | The server is stopped and blocked, the client is notified with a link to the service in WHMCS |
| Unsuspended | The block set by WHMCS is lifted. A block set by a panel administrator stays |
| Terminated | The server and its files are deleted from the node |
| Product or options changed | The server plan and resources change. New limits apply on the next start |
| Password changed | The client's panel account password changes |
| Renewed | The next due date is updated |
| Client details changed | Email and name are updated in the panel |
| Log in to panel | The client opens the server page without entering a password |
| Usage update | Disk usage is written to WHMCS |

In the WHMCS client area the client sees the status, address, game, location
and resources of the server and can start, restart and stop it. A WHMCS
administrator sees the same data on the service page, can reinstall the server
and sync its address and due date.

A server created through WHMCS is not renewed from the panel balance: WHMCS
manages the rental period. The plan tab of such a server shows the service
number, the next due date and a link to the service in WHMCS. The client
renews, changes the plan and cancels the server in WHMCS.

## Setup

1. In the panel open Integrations, the WHMCS section, and click Download
   module. Unpack the archive into the WHMCS root: the files go to
   `modules/servers/vortanix`.
2. In the same section click Issue a WHMCS key and copy the key. The key only
   gets the `admin.whmcs.write` permission and opens nothing else in the
   admin area.
3. In WHMCS open System Settings → Servers → Add New Server:
   - Module — Vortanix;
   - Hostname — the panel domain, for example `panel.example.com`;
   - Access Hash — the key from step 2;
   - Secure — on.

   Click Test Connection.
4. Create a server group and add the server to it.
5. Create a product. On the Module Settings tab pick the Vortanix module, the
   server group from step 4 and a panel plan. Choose when the server is
   created, for example after the first invoice is paid.
6. Enter the WHMCS address in the panel: links to the client's service are
   built from it.

The sign-in link for clients coming from WHMCS points to the panel address
from the `FRONTEND_URL` environment variable.

## Product settings

| Field | Value |
|---|---|
| Panel plan | Required. Sets the resources and the game |
| Game | Empty takes the game of the plan |
| Location | Automatic picks the least loaded location with room for the game |
| Game version | Version ID or name. Empty installs the default version |
| Slots, CPU cores, RAM, disk | 0 takes the value from the plan |
| Server name | Template with `{service_id}`, `{client_id}`, `{domain}`. Empty uses the game name and service number |

## Client choices at order time

To let clients choose parameters, create Configurable Options with the names
from the table. The display name goes after a pipe: `cs16|Counter-Strike 1.6`.

| Option name | Value |
|---|---|
| `game` | Game code from the panel catalog, for example `cs16` |
| `location` | Location ID, code or name |
| `version` | Game version ID or name |
| `slots`, `cpu_cores`, `ram_gb`, `disk_gb` | Number |
| `tariff` | Panel plan ID |

The client sets the server name in a Custom Field named `server_name`. Option
and field values take priority over the product settings.

## Orders only in WHMCS

The Take server orders only in WHMCS checkbox sends clients from the panel rent
page to WHMCS and turns off renting servers from the panel balance. The trial
server is not offered on the rent page in this mode. Web hosting and balance
top-ups keep working.

## Accounts

- A WHMCS client is linked to a panel account by client number. On the first
  order the account is looked up by email; if there is none, it is created
  with the service password from WHMCS.
- A panel staff email cannot be used for a WHMCS client: service creation
  fails, and signing in as staff from WHMCS is refused.
- If the account already existed, its password is not changed. The client
  signs in with the button in WHMCS or with the existing password.

## API

The module calls `https://<panel>/v1/admin/whmcs` with the `X-API-Key` header.
You can use the same methods from your own scripts.

| Method | Path | Purpose |
|---|---|---|
| GET | `/info` | Connection check |
| GET | `/catalog` | Games, plans, locations and game versions |
| GET | `/services` | List of services |
| GET | `/usage` | Disk usage per service |
| POST | `/services/{id}` | Create a server; a repeated call returns the existing one |
| GET | `/services/{id}` | Service and server details |
| DELETE | `/services/{id}` | Delete the server |
| POST | `/services/{id}/suspend` | Suspend |
| POST | `/services/{id}/unsuspend` | Unsuspend |
| POST | `/services/{id}/package` | Change plan and resources |
| POST | `/services/{id}/renew` | Update the next due date |
| POST | `/services/{id}/password` | Change the account password |
| POST | `/services/{id}/power` | `start`, `stop`, `restart` or `kill` |
| POST | `/services/{id}/reinstall` | Reinstall the server |
| POST | `/services/{id}/sso` | One-time sign-in link, valid for 2 minutes |
| PUT | `/clients/{id}` | Update the client's email and name |

`{id}` is the WHMCS service or client number.

## Troubleshooting

- Module requests and panel responses are written to the WHMCS System Logs →
  Module Log when the log is enabled.
- `invalid api key` — the key was revoked or copied incompletely.
- `ключ не имеет права admin.whmcs.write` (the key lacks the permission) —
  issue a key with the button in the WHMCS section.
- `не задан публичный адрес панели (FRONTEND_URL)` (the panel public address
  is not set) — set the environment variable, otherwise sign-in from WHMCS
  cannot build a link.
