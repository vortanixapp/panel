# Telegram bot and wipes

The bot sends notifications and lets users control servers with buttons. The wipe scheduler warns players, makes a backup, stops the server, deletes the game data and starts it again.

## Connecting

1. Create a bot with `@BotFather` and set its token in `TELEGRAM_BOT_TOKEN` or in "Settings → Notifications".
2. A user opens "Settings → Notifications → Telegram", presses "Connect" and follows the link to the bot.
3. After `/start` the bot binds the private chat to the account and shows the menu. Server control can be turned off with the switch in the settings.

Control works only in a private chat. Groups and channels receive notifications but their commands are ignored. A chat ID typed by hand is not verified: control needs the link flow.

The bot reads messages with long polling, no inbound port is needed. If the token has a webhook, the panel removes it on start. One API instance runs the polling, guarded by a PostgreSQL advisory lock.

## What the bot does

- Server list and a card with status, address, players, load and rental expiry.
- Start, stop, restart and force kill. Stop and kill ask for confirmation.
- Console: ready-made game commands and free-form commands, with the output in the chat.
- Logs, player list, one-hour load chart.
- Backups: create, list, restore with confirmation.
- Wipes: schedules, skip the next one, run manually, cancel before it starts.
- Balance and disconnecting Telegram.
- Buttons in notifications: a crashed server comes with "Restart" and "Logs", a failed backup with "Make a backup".

The buttons depend on the user's permissions on the server. Every action is written to the audit log with a `telegram` mark. At most 40 actions per minute per user. In read-only mode (panel transfer) actions are disabled.

## Wipes

The "Wipes" tab on the server page is available for Rust, ARK (SE and SA) and DayZ.

| Game | Types | What is deleted |
|---|---|---|
| Rust | map; map and blueprints; full | `server/<identity>`: `*.map`, `*.sav`, `sv.files.*.db`, `player.deaths.*.db`, `player.states.*.db`; blueprints add `player.blueprints.*.db`; full removes all `*.db`. The `cfg` folder stays |
| ARK | world; world and players | `*.ark` in `ShooterGame/Saved/SavedArks`; the second type also removes profiles and tribes |
| DayZ | economy; full | `storage_1/data` or the whole `storage_1` in each `mpmissions` folder |

For Rust a new map seed is written to `rust.env` (`RUST_SEED`) when the map is procedural.

Schedules: weekly, monthly (for example the first Thursday), cron or one time. Times use the owner's time zone or the one set in the schedule.

Order: chat announcements (60, 30, 10, 5 and 1 minutes by default), `save` and a backup, stop, delete, new seed, start. If the backup fails, the wipe does not start and the data stays untouched. If the server was stopped, the backup is skipped and the server stays stopped afterwards. Announcements work for Rust (`say`) and ARK (`Broadcast`); DayZ has no console channel, so players need another way of being warned.

A wipe missed by more than 10 minutes (the panel was down) is not run, the schedule moves to the next one. The owner gets a notification with the result; on failure the server stays stopped and Telegram offers a "Start" button.

Wipes do not clear plugin data (Oxide/Carbon and others): add that cleanup to the server's cron.
