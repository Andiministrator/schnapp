# Schnapp for scripts and agents

Schnapp can be driven from scripts, AI agents and local models through its command line. This page lists what a
non-interactive caller needs: the calls, the JSON they print, the exit codes and the limits. The design decision behind
it is ADR-0030 (`docs/adr/0030-agenten-schnittstelle.md`, German).

Schnapp runs on Linux with Wayland (KDE Plasma/KWin). Capturing talks to KWin directly; nothing goes over the network.

## Calls

| Call | What it does |
|---|---|
| `schnapp --screen=NAME --output=shot.png` | Captures monitor NAME into `shot.png`, without a window and without the library |
| `schnapp --screen --output=shot.png` | Captures the monitor chosen by the setting `[capture] screen` (`pointer`, `all`; `pick` needs a click) |
| `schnapp --repeat --output=shot.png` | Captures the last area selected with `--region` again |
| `schnapp --window --output=shot.png` | Lets the user click a window, then captures it |
| `schnapp --region --output=shot.png` | Lets the user drag an area (interactive) |
| `schnapp --list-screens --json` | Lists the monitors |
| `schnapp --export=out.png FILE` | Renders FILE (a Schnapp working file, PNG or JPEG) with its crop, annotations and redactions into `out.png`, without a window; FILE stays unchanged |
| `schnapp --version --json` | Prints the version |

Without `--output`, a capture is stored in the library (`~/Pictures/schnapp/YYYY-MM/…`, see the README) and the
editor opens unless `--no-editor` is given or the setting `[capture] open-editor` is off.

Rules:

- Options with a value take it after an equals sign only: `--output=FILE`, `--export=TARGET`, `--screen=NAME`.
- `--export` to PNG keeps transparency: where a working file has a squeeze seam, the gap is transparent (alpha 0, with a soft shadow); JPEG fills it with white.
  `--output FILE` (with a space) is rejected with a hint.
- The format follows the file name: `.png`, `.jpg`/`.jpeg` or `.svg`. Any other ending is rejected before capturing.
- `--output` and `--export` replace an existing file. The image is first rendered completely in memory, then written to
  a temporary file `.schnapp-export-…` next to the target and renamed over it: the target never holds a half-written
  file, and on an error an existing target stays as it was. Two exceptions:
  - While the short write itself runs, Schnapp catches `SIGTERM`, `SIGINT` and `SIGHUP` (not one the process already
    ignores, such as `SIGHUP` under `nohup`). A signal before the first byte is written removes the temporary file,
    leaves the target untouched and ends with exit code 1. A signal after the write has begun cannot undo it: the
    target is then replaced completely. The call still ends with exit code 1, but says "Interrupted by a signal after
    … was written completely" and prints the result line (path, or the JSON line with `--json`) on standard output, so
    a script can tell this case from a failure (an ordinary failure prints no result line). If the write hangs, a second
    signal removes the temporary file and ends the call with exit code 1, and says what state it left: the file was
    already complete, or the temporary file could not be removed. In the direct write below there is no temporary
    file: a second signal then ends the call with exit code 1 and the message "Interrupted again while writing
    directly to the target; it may be incomplete." (Schnapp does not touch the hanging write, so the target may be
    truncated; check it before using it). Being signalled while rendering leaves
    nothing either. Only `SIGKILL` (or a crash of the machine) during the write can leave a temporary file behind.
  - If the folder does not allow a new file but the existing target is writable, Schnapp writes the target directly
    and restores the old content if that fails. Only if restoring fails too, the call ends with exit code 1 and the
    message that the target file is incomplete.
- `--export` refuses a target that is FILE itself, also through a symbolic or hard link (exit code 1; nothing is
  written): it would replace the working file, with its original and annotations, by the flat image.
- A new file written by `--output` gets the permissions `0600` (it shows the screen, like the library). `--export`
  creates files with the usual permissions (umask), like the export from the editor. An existing target keeps its
  permissions in both cases.
- With `--json`, the file names for `--output` and `--export` must be valid UTF-8 (otherwise exit code 2): JSON could not
  report such a path faithfully. If a path that Schnapp itself chooses (the library folder, the current folder for a
  relative name) is not valid UTF-8, no JSON line is printed; the file exists and the exit code is 0, a message on
  standard error says why.
- Exported files carry no metadata: no paths, no program or host names.

## Output

With `--json`, a successful call prints exactly one line of JSON on standard output. The keys are English and do not
depend on the language of the system.

```json
{"path":"/data/shots/shot.png","width":1920,"height":1080}
```

for a capture (with or without `--output`) and for `--export`. `path` is absolute; `width` and `height` are the size of
the image in pixels.

```json
{"screens":[{"name":"DP-1","width":2560,"height":1440},{"name":"HDMI-A-1","width":1920,"height":1080}]}
```

for `--list-screens --json`. `name` is exactly the value for `--screen=NAME`; the size is the current mode in physical
pixels, as the monitor shows it: for a monitor rotated by 90 or 270 degrees, width and height are swapped, so the size
matches the image of `--screen=NAME`.

```json
{"version":"0.5.0"}
```

for `--version --json`.

Without `--json`, a capture or export prints only the absolute path. Messages and errors go to standard error, in the
language of the system; do not parse them.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Error (capture failed, file not readable or not writable, …); details on standard error |
| 2 | Wrong call (unknown option, options that do not combine); nothing was captured or written |
| 3 | Cancelled by the user (Esc in the area or window selection) |

## Limits and privacy

- Every capture shows what is on the screen, including mail, chats and passwords that happen to be visible. Capture
  only what the task needs, and prefer `--screen=NAME` over all monitors.
- `--output` bypasses the library on purpose: such captures are not indexed, not shown in the library and not cleaned
  up. Delete them yourself when they are no longer needed.
- Cropping hides but does not remove: a Schnapp working file keeps the uncropped original. Exports (`--output`,
  `--export`) contain only the visible part. Redaction (black out, pixelate) is final in every saved file.
- `--export` refuses files that would be too costly to render (crafted files); `--ignore-limit` skips this check — use
  it only for files you trust.
- Annotating and redacting from the command line (and as MCP tools) are planned; see ADR-0030.

## MCP server

`schnapp mcp` runs Schnapp as a Model Context Protocol server over standard input and output (newline-delimited
JSON-RPC 2.0). It opens no network socket; it talks to the desktop only over the session D-Bus (capture,
notification) and Wayland (confirmation dialog). Register it in your agent like any stdio server, with the command
`schnapp` and the single argument `mcp`. It speaks protocol revision 2026-07-28 (`server/discover`, `_meta` on every
request) and the older handshake form (`initialize`, revisions 2024-11-05 to 2025-11-25); `ping` is answered in both.
Logs go to standard error only. Requests run one at a time; while a tool runs, at most 64 further messages (8 MiB) are
held, and a request beyond that fails with `-32600`. After SIGTERM, SIGINT or SIGHUP no further request starts; a
second signal stops the server at once.

Tools (names and JSON keys are stable):

| Tool | Arguments | Needs |
|---|---|---|
| `list_screens` | — | nothing |
| `capture_screen` | `screen` (a name from `list_screens`), `all` (bool), `output` (absolute path) — without `screen` and `all`: the active monitor, or all if `[capture] screen = all` | `capture` |
| `capture_repeat` | `output` | `capture` |
| `library_search` | `tag`, `from`, `to` (RFC 3339 or `YYYY-MM-DD`), `limit` (1–50, default 20), `agents` (default true) | `library` |
| `library_read` | `path` (absolute or relative to the library) | `library` |
| `export` | `file`, `target` (absolute paths) | `library` if `file` is in the library or its preview cache |

Captures and `library_read` return `structuredContent` `{"path","width","height"}`, the same JSON as text and — with
`image = yes` — the visible image as a PNG (at most 8 MiB). There is no overlay over MCP: picking an area needs a person.

`output` and `target` must be new files: the server never replaces an existing file (the call fails with "already
exists") and refuses targets inside the Schnapp library or its preview cache, also through a symbolic link to the
folder. A new `output` file gets mode 0600, a new `target` follows the umask. If the location of the library cannot be
determined, `output` and `export` are refused. This differs from `--output` and `--export` on the command line, which
replace an existing file.

Consent lives only in `settings.ini` under the user's real home directory (`~/.config/schnapp/settings.ini`, home taken
from the user database). The server ignores `HOME` and `XDG_*` from its environment — the agent's client sets that
environment — and takes the settings, the library and the last area from the real home directory instead; if the
environment points elsewhere, it says so on standard error. It reads the file on every call and never writes it:

```ini
[agents]
capture = no   # yes: agents may take screenshots
library = no   # yes: agents may search the library and read images from it
confirm = yes  # ask in a dialog before every capture (Allow / Deny, 60 s, then denied)
image = yes    # send the image itself to the agent; no: only path and size
```

A call without consent fails with `isError: true` and a sentence that names the missing switch. Every capture shows a
desktop notification (urgency critical); if the notification service does not accept it, the screenshot is discarded.
The notification is only a hint: the server knows that the service took it, not that a person saw it, and a
do-not-disturb mode may hide it. If you want to see every capture, keep `confirm = yes`. Captures without `output` go
to the library under `<Pictures>/schnapp/agents/` and are cleaned up like any capture. A capture that is cancelled
(`notifications/cancelled`, end of input, first signal) is discarded even if the image was already taken or written: the
server removes the file it has just created (in the library together with its preview and index entry). Only if that
removal fails does the answer name the file. An `export` that a signal interrupts after the target was written
completely keeps the file and says so. Error texts to the agent are always English and never contain the user's home directory (only paths the
agent gave or got); `export` gives one sentence for every source it cannot open (missing, unreadable, not an image).

The dialog answers only to a pointer (mouse, touch or pen): "Allow" counts only when it is pressed and released after a
short lock (0.6 s) once the dialog appears and again each time it gets the focus; no key ever allows (Return, Space
and Tab cannot reach the button). Esc, "Deny" and closing the window deny. The dialog closes itself shortly after the
60 s limit, even if the server is gone. The name the agent reports about itself is shown in the dialog as unverified;
the dialog speaks the language of the server (from the real home directory).

The dialog appears only on the compositor that also takes the screenshot: before showing it, the server connects to
the Wayland socket (`WAYLAND_DISPLAY`, relative to `XDG_RUNTIME_DIR`) and checks that the process serving it owns
`org.kde.KWin` on the session bus (`DBUS_SESSION_BUS_ADDRESS`) or is its parent (`kwin_wayland_wrapper` in Plasma).
The dialog gets exactly this checked connection. If they differ, or `WAYLAND_SOCKET` is already set, nothing is
captured. An environment that points both elsewhere can only capture that other desktop, never the real one.
