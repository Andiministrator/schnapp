# Schritt-QA

Nach jedem abgeschlossenen Schritt durchgehen. Was nicht zutrifft, überspringen. Nur P0 blockiert den nächsten
Schritt.

## 1. Inhalt & Korrektheit
- [ ] Aufgabe der Karte erfüllt, nichts darüber hinaus?
- [ ] ADR-konform? Besonders: nur Wayland (ADR-0007), Dateiformat (ADR-0008), Fremdpakete begründet (ADR-0011)?
- [ ] Architektur-konform: Aufnahme nur über die Aufnahmeschicht, Zeichnen über die Zeichenpakete?

## 2. Datenschutz im Produkt ([S]-Pfad)
- [ ] Berührt der Schritt Schwärzen/Verpixeln, das eingebettete Original, einen Export, die Zwischenablage oder die
      Bibliothek? Dann: Test, dass geschwärzte Pixel in **keinem** Teil der Ausgabe überleben.
- [ ] Keine Metadaten mit Pfaden, Benutzer-, Rechner- oder Programmnamen in erzeugten Dateien?

## 3. Öffentliches Repo
- [ ] Keine Kunden-/Firmenbezüge, keine lokalen Pfade (`/home/…`), keine Mailadressen oder Hausnetz-Angaben, keine echten Screenshots in Code,
      Kommentaren, Testdaten, Golden-Dateien, Commit-Nachricht?
- [ ] Neue Fremdpakete oder Stempel: Lizenz GPL-3.0-verträglich und im Commit genannt (ADR-0009, ADR-0011)?

## 4. Abhängigkeiten
- [ ] Andere Karten betroffen? Ändert sich eine öffentliche Schnittstelle der Zeichenpakete, die ein
      Schwesterprojekt nutzt (ADR-0012)? Dann vorher abstimmen (`CLAUDE.local.md`).

## 5. Code-Qualität
- [ ] `go vet ./...` und `go test ./...` grün; `gofmt` angewendet. Bei Änderungen an `cmd/schnapp-fernbedienung` oder
      Probe-Code zusätzlich mit Tag `probe` (`GOFLAGS=-tags=novulkan,nox11,probe go vet … && go test …`).
- [ ] Sichtbares Verhalten selbst geprüft, soweit der virtuelle KWin es kann (`scripts/probe.sh --drehbuch`)?
- [ ] Keine Hacks, kein toter Code, keine `// TODO` ohne Finding.
- [ ] Kein `cgo` ohne ADR.

## 6. Gegenprobe
Jeder Fix und jede neue Prüfung muss zeigen, dass der Test das Produkt prüft:
- [ ] **Fix entfernt → mindestens ein Test rot?** Sonst prüft der Test etwas anderes als das Produkt.
- [ ] **Minimal zurückbauen** — genau die Zeile, die der Fix hinzugefügt hat, nicht ihren Kontext. Ein rotes Ergebnis
      nach zu großem Rückbau misst den Kollateralschaden.
- [ ] **Betritt der Testfall den Zweig überhaupt?** Ein Fix hinter einer im Test nie wahren Bedingung ist mit und ohne
      Fix grün.
- [ ] **Nie per `git checkout <datei>` zurückbauen** — gezielt zurückeditieren und `git diff` lesen.
- [ ] Nicht sinnvoll testbar (z. B. Wayland-Overlay, KWin-Aufruf)? Dann gehört **das** in den Testkommentar, mit
      Begründung. Eine offen benannte Lücke ist besser als eine behauptete Abdeckung.

## 7. Dokumentation
- [ ] Neue Entscheidung → ADR? Recherche → Spurensicherung (`knowledge/sources-visited.md`)? Wissen →
      `knowledge/topics/`?

## 8. ADR-Gegenrichtung
- [ ] **Ändert der Schritt Verhalten, das ein Nutzer bemerkt? Dann vorher in `docs/adr/` nachsehen — nach dem
      VERHALTEN, nicht nach dem geänderten Bezeichner.** Eine bestehende ADR kann genau das alte Verhalten
      festgeschrieben haben; wird sie nicht nachgezogen, behauptet das Repo das Gegenteil des Codes.
- [ ] **Nennt der Auftrag (Karte, Kontextpaket) eine ADR, ist sie zu PRÜFEN**, nicht zur Kenntnis zu nehmen.
- [ ] Bezeichner-Suche reicht nicht. Beispiel für einen Speicher- oder Schwärzen-Schritt:
      `grep -rniE "schwärz|verpixel|original|metadat|block|export" docs/adr/`

*Anlass (aBoard, Sprint 63): Eine Karte stellte das Karten-Modal von „beim Schliessen verwerfen" auf „beim Schliessen
speichern" um; die zuständige ADR sagte das Gegenteil und stand sogar im Kontextpaket — bemerkt erst nach der
Auslieferung.*

## 9. Findings
- [ ] Neue Befunde in `docs/findings.md` (mit Prüfbefehl, ggf. Widerlegungsbedingung) **und** Board-Karte? P0 behoben?
