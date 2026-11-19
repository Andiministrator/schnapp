# Plattformen und ihre Regeln zu KI-erzeugtem Code

Stand: 2026-11-19

Schnapp wird überwiegend von Claude geschrieben; das schließt manche Plattformen aus oder bindet sie
an Auflagen. Quellen: `sources-visited.md` (Einträge vom 19.11.2026).

## Codeberg — ausgeschlossen

Nutzungsbedingungen § 2 Abs. 1 Nr. 7 (seit Juli 2026, Mitgliederabstimmung 358:144): „You must not share projects
that mostly consist of code written by "generative AI"-tools" — ausdrücklich mit Claude und OpenAI Codex als Beispiel.
Begründung: unklarer Urheberrechtsstatus, kaum Schutz gegen schädlichen Code. Erlaubt bleibt nur geringe KI-Hilfe.
**Folge:** Schnapp kann nicht auf Codeberg liegen (Andi, 19.11.2026; ADR-0010).

## Flathub — erlaubt mit Offenlegung, aber mit harten Grenzen

Verlauf: Mai 2026 Verbot fast aller KI-erzeugten Apps, Oktober 2026 zurückgenommen und durch eine Offenlegungspflicht
ersetzt (laut GamingOnLinux und Linuxiac; Wortlaut vor einer Einreichung in Flathubs Doku selbst prüfen).
- KI-erzeugten Code, Doku, Paketierung **offenlegen** (welche Teile, ungefährer Umfang).
- Das **Flathub-Manifest darf nicht KI-erzeugt oder KI-unterstützt** sein — Andi schreibt es selbst.
- **KI-Werkzeuge dürfen keine Einreichungs-PRs öffnen** oder deren Commit-Nachrichten, Beschreibungen, Antworten
  verfassen. Das macht Andi.
- Prüfer dürfen nach Umfang der KI-Anteile ablehnen; Offenlegung ist keine Zusage.
**Folge:** Flathub bleibt möglich, aber nur mit Andis Handarbeit an Manifest und Einreichung und ohne Garantie.
Alternative: eigenes Flatpak-Repo oder Pakete (AUR, eigene Releases).

## Für alle öffentlichen Orte

KI-Anteil in der README offen nennen (passt zu Flathub-Pflicht und schafft Vertrauen). Vor jeder neuen Plattform deren
aktuelle Regeln zu KI-Code prüfen und hier nachtragen.
