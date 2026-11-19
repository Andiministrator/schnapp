# ADR-0010: Repos auf GitHub und einem privaten Zweit-Repo

## Status
Akzeptiert (Andi, 18. und 19.11.2026)

## Kontext
Schnapp ist öffentlich und braucht eine CI für Go-Tests. Ursprünglich war Codeberg als Haupt-Repo vorgesehen.
Codeberg verbietet seit Juli 2026 Projekte, die überwiegend aus KI-erzeugtem Code bestehen (Nutzungsbedingungen
§ 2 Abs. 1 Nr. 7, Claude ausdrücklich genannt); Schnapp ist so ein Projekt
(`knowledge/topics/plattformen-und-ki-regeln.md`).

## Entscheidung
- **Haupt-Repo öffentlich auf GitHub:** `github.com/Andiministrator/schnapp`, Remote `origin`. CI über GitHub Actions.
- **Zweites Push-Ziel:** ein privates Repo im Heimnetz des Maintainers, als zusätzliche Push-URL von `origin`; beide
  werden bei jedem Push bedient. Einzelheiten stehen in der nicht versionierten `CLAUDE.local.md`.
- Der KI-Anteil wird in der README offen genannt.
- Commits tragen die GitHub-noreply-Adresse des Maintainers, keine private Mailadresse.

## Konsequenzen
- Vor jedem Push gilt: nichts mit Kunden- oder Firmenbezug, keine lokalen Pfade, keine Hausnetz-Angaben (F-1).
- Force-Push ist auf dem Zweit-Repo serverseitig gesperrt; die gemeinsame Geschichte darf nicht umgeschrieben werden.
