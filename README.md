# Schnapp

Screenshot and annotation tool for Linux/Wayland — KDE Plasma first. Inspired by the workflow of Snagit: capture
fast, annotate (straight and curved arrows, boxes, step numbers, stamps), redact for real, cut out the middle of long
pages, share. Also meant as a simple image editor.

**Status:** early work. Nothing usable yet; a prototype decides whether the Gio UI toolkit is up to the job.

## AI disclosure

Most of the code and documentation in this repository is written by an AI model (Anthropic Claude) under the
direction and review of the maintainer. Decisions are recorded in `docs/adr/`.

## Layout

- `zeichnen/` — Go module `github.com/Andiministrator/schnapp/zeichnen`: SVG parsing, an anti-aliased rasterizer and
  numeric helpers (standard library only).
- `docs/` — requirements, architecture decisions (ADR), QA checklists, findings. Mostly in German.
- `knowledge/` — research notes with sources (German).

## Licence

GPL-3.0-or-later, see `LICENSE`.

---

*Deutsch:* Schnapp ist ein Screenshot- und Beschriftungswerkzeug für Linux/Wayland (KDE zuerst). Der größte Teil des
Codes und der Doku wird von einem KI-Modell (Anthropic Claude) unter Anleitung und Prüfung des Maintainers geschrieben.
Projektdokumentation auf Deutsch unter `docs/` und `knowledge/`.
