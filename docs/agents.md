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
  `--output FILE` (with a space) is rejected with a hint.
- The format follows the file name: `.png`, `.jpg`/`.jpeg` or `.svg`. Any other ending is rejected before capturing.
- `--output` and `--export` replace an existing file (written atomically: no half-written file is ever left behind).
- Exported files carry no metadata: no paths, no program or host names.

## Output

With `--json`, a successful call prints exactly one line of JSON on standard output. The keys are English and do not
depend on the language of the system.

```json
{"path":"/home/me/shots/shot.png","width":1920,"height":1080}
```

for a capture (with or without `--output`) and for `--export`. `path` is absolute; `width` and `height` are the size of
the image in pixels.

```json
{"screens":[{"name":"DP-1","width":2560,"height":1440},{"name":"HDMI-A-1","width":1920,"height":1080}]}
```

for `--list-screens --json`. `name` is exactly the value for `--screen=NAME`; the size is the current mode in physical
pixels.

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
- Annotating and redacting from the command line, and an MCP server (`schnapp mcp`, stdio only, with separate consent
  for capturing and for reading the library), are planned; see ADR-0030.
