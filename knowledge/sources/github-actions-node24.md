# GitHub Actions: checkout und setup-go auf Node 24 (Karte #29, F-43/F-309)
- URL: https://github.com/actions/checkout/releases · https://github.com/actions/setup-go/releases (Abruf über die öffentliche REST-API und raw.githubusercontent.com)
- Abgerufen: 2026-11-21
- Herausgeber / Stand: GitHub-Aktionen von actions/. checkout v7.0.1 (2026-07-20), setup-go v7.1.0 (2026-11-17)
- Verlässlichkeit: offiziell

## Kernaussagen
- `actions/checkout` v7.0.1: `action.yml` Zeile 116, `runs.using: node24`, `main: dist/index.js`. Tag `v7.0.1` zeigt direkt auf Commit `3d3c42e5aac5ba805825da76410c181273ba90b1` (leichtes Tag).
- `actions/setup-go` v7.1.0: `action.yml` Zeile 48, `runs.using: 'node24'`, `main: dist/setup/index.js`. Tag `v7.1.0` zeigt direkt auf Commit `d8cbb8045d7f43706b9177cceb36c1f91a6644d5` (leichtes Tag).
- Eingabe `cache` von setup-go bleibt standardmäßig `true` (action.yml, Zeile 15 bis 17); `go-version-file` unverändert.
- Release-Notes checkout v7.0.0: Fork-PRs werden für `pull_request_target` und `workflow_run` blockiert. Die Workflow-Datei nutzt nur `push` und `pull_request`, ist also nicht betroffen.
- Release-Notes checkout v6.0.0: Zugangsdaten werden in eine eigene Datei geschrieben. Der Workflow führt keinen `git push` aus, also ohne Wirkung.
- Release-Notes setup-go v7.0.0: Umstellung auf ESM und Aktualisierung der Abhängigkeiten, ohne Eingabeänderung. v7.1.0: neue Ausgaben (go env), Prüfung von rc-/beta-Versionen in go.mod, Hinweis auf die Nutzung von checkout v7 in der eigenen Dokumentation.

## Offene Punkte
- Die Eingabe-Standardwerte wurden nur in action.yml geprüft, nicht im Lauf einer CI-Ausführung. Der Prüfbefehl ist ein CI-Lauf ohne die Annotation „Node.js 20 is deprecated“.
- Dependabot für `github-actions` würde künftige SHA-Hebungen automatisieren (Vorschlag, nicht umgesetzt; entschieden 21.11.2026: kein Dependabot, Prüfpunkt beim Sprint-Abschluss (F-316)).
