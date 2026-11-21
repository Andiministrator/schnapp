# Plattformen und ihre Regeln zu KI-erzeugtem Code

Stand: 2026-11-21 (Gio-Einreichung ergänzt), 2026-11-19

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

## Gio — Patch einreichen (Stand 21.11.2026, Karte #17)

Quelle: https://gioui.org/doc/contribute, am 21.11.2026 geöffnet und gelesen (Zitate gekürzt):

- **Hauptweg Mailingliste:** „Patches should be sent to ~eliasnaur/gio-patches@lists.sr.ht mailing list with the
  `git send-email` command.“ Kein SourceHut-Konto nötig („No Sourcehut account is required and you can post without
  being subscribed“). Betreff-Präfix per `git config format.subjectPrefix "PATCH gio"`, sonst greifen die
  automatischen Tests nicht; Neufassungen mit `-v2` usw.
- **GitHub nur als Ersatzweg:** „If using sourcehut is problematic the project is mirrored on Github and does accept
  pull-requests.“
- **Sign-off ist Pflicht**, auch für Pull Requests („signing the commits with --signoff is still required“, DCO,
  Kompatibilität mit MIT und UNLICENSE). Der Sign-off läuft unter Andis Namen — Andi reicht selbst ein.
- **Zu KI-erzeugtem Code sagt die Seite nichts.** Offenlegung im Anschreiben ist trotzdem der ehrliche Weg
  (eigene Abwägung, keine Gio-Regel).
- ⚠️ unbelegt (nur Suchtreffer, Newsletter nicht geöffnet): Gio v0.10.3 ist die aktuelle Fassung (September 2026).
- **Entscheidung (Claude im Auftrag Andis, 21.11.2026):** Pull Request am GitHub-Spiegel `gioui/gio` — Andis
  GitHub-Konto ist eingerichtet, `git send-email` wäre neu einzurichten. Gio nimmt PRs als Ersatzweg ausdrücklich an;
  im PR-Text kurz sagen, warum nicht per Mail. Die Mailingliste bleibt der Weg, wenn die Gio-Leute darum bitten.
- Vorbereitung (Karte #17): Patch in zwei Teile (FullscreenOn/WaylandOutputs; fractional-scale + viewporter samt
  Zeiger), Protokoll-Code per `go:generate` statt erzeugter Dateien, `log.Printf` in `outputByName` als offene Frage.

## Für alle öffentlichen Orte

KI-Anteil in der README offen nennen (passt zu Flathub-Pflicht und schafft Vertrauen). Vor jeder neuen Plattform deren
aktuelle Regeln zu KI-Code prüfen und hier nachtragen.
