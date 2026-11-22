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
