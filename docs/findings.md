# Findings

Typen: `[S]` Sicherheit · `[K]` Stabilität · `[B]` Blockierend · `[V]` Verbesserung · `[D]` Konsistenz.
Priorität: P0 (sofort) · P1 (diesen Sprint) · P2 (nächsten Sprint) · P3 (irgendwann).
Schulden-Grenze: > 10 offene `[K]`-Findings → Hygiene-Sprint einplanen.

**Jedes offene Finding hat eine offene Board-Karte** mit seiner F-Nummer in Titel oder Beschreibung.

**Pflichtangaben für offen eingetragene Befunde:**
- **Prüfbefehl** — ein Aufruf mit erwartetem Wert, an dem sich ablesen lässt, ob die Lücke noch besteht. Am besten
  eine Verhaltensprobe (Test, Aufruf, Messung); ein Wort-`grep` nur für reine Doku-Befunde.
- **Widerlegungsbedingung** — nur bei Befunden, deren Fix-Richtung Code ändert: woran sich zeigt, dass der notierte Weg
  nicht trägt. Für `[D]` freiwillig.

Im selben Zug gefundene und behobene Befunde brauchen beides nicht; dort steht der Test im Fix-Commit.

## Offen

| ID | Typ | P | Sprint | Beschreibung | Status |
|---|---|---|---|---|---|
| F-970 | [K] | P2 | Sprint 14 | **Feld „Schatten“ kehrt nach Texteingabe oder Einzelauswahl ohne Klick zurück** (Sitzungs-Kritik 28.11. K-1, Regression durch F-952 in d0e6139): `feldSchattenID` = Hauptauswahl ist zugleich `schattenZiel()` der allein gewählten; die nächste Farbwahl färbt den Schatten eines Kastens. **Prüfbefehl:** `TestFeldSchattenNachTexteingabe`, `TestFeldSchattenNachVerkleinern` in `internal/canvas/feldschatten_test.go`. | offen (Karte #391) |
| F-971 | [K] | P2 | Sprint 14 | **`bereinigeAuswahl` zählt `mf.gen` auch ohne Änderung: Anker der Ebenenliste und Strg+H-Gedächtnis verfallen nach Deckkraft/Umbenennen** (Sitzungs-Kritik K-2). **Prüfbefehl:** `TestAnkerNachUmbenennen`, `TestSchalterNachDeckkraft`. | offen (Karte #391) |
| F-972 | [K] | P2 | Sprint 14 | **`schalteGruppeEbene` rechnet die Schaltmenge vor dem Bestätigen der Texteingabe**: Hauptauswahl kann auf ein gelöschtes Objekt zeigen; Auswahl von vorher nicht hergestellt (Sitzungs-Kritik K-3). **Prüfbefehl:** `TestSchalterInTexteingabe`, `TestSchalterInTexteingabeLeer`. | offen (Karte #391) |
| F-973 | [V] | P3 | Sprint 14 | **Strg+H, dann Strg+L verliert das Gedächtnis von Strg+H; Strg+H nach Rückgängig schaltet nur ein Mitglied** (Sitzungs-Kritik K-4). **Prüfbefehl:** `TestSchalterWechsel`. | offen (Karte #391) |
| F-974 | [B] | P3 | Sprint 14 | **Testlücken Mehrfachstil/Gruppen** (Sitzungs-Kritik K-5): Merker im Zug nur über Zeit geprüft, Anker nach Strg-Klick, `geschaltet = nil`, Blättern der Liste. **Prüfbefehl:** Zähltest `TestMerkerImZugEinmal`, Mutanten MS41ff./GR30ff. | offen (Karte #391) |
| F-975 | [D] | P3 | Sprint 14 | **CLAUDE.md nennt GR1–GR25 statt GR1–GR29** (Sitzungs-Kritik K-6, QA Q-2, Rest F-967). | offen (Karte #391) |
| F-976 | [D] | P2 | Sprint 14 | **docs/format.md nennt `s:gruppe` in §8.3, in der Zeile `format_version` und in der Beispieltabelle §14 nicht** (Sitzungs-QA Q-1). **Prüfbefehl:** Beispieltabelle §14 enthält eine Zeile `groups`. | offen (Karte #391) |
| F-977 | [D] | P3 | Sprint 14 | **Glossar „Hauptauswahl“/„Mehrfachauswahl“ und Anfang des CLAUDE.md-Abschnitts beschreiben den Stand vor ADR-0039 Nachtrag 4** (Sitzungs-QA Q-3). | offen (Karte #391) |
| F-978 | [D] | P3 | Sprint 14 | **Ältere ADRs ohne Verweis auf ADR-0044 und ADR-0039 Nachtrag 4** (Sitzungs-QA Q-4): 0032 (Strg+H/L, Liste), 0038 (Kontextmenü), 0037 (Schnellstil an mehreren), 0021 (Feld Schatten); ADR-0044 Punkt 2 ohne Kontextmenü, Kopie eines Mitglieds. **Prüfbefehl:** `grep -rn ADR-0044 docs/adr` trifft auch 0032, 0037, 0038, 0021. | offen (Karte #391) |
| F-979 | [D] | P3 | Sprint 14 | **README en/de: Kontextmenü-Absatz ohne Gruppieren/Aufheben** (Sitzungs-QA Q-5). | offen (Karte #391) |
| F-980 | [D] | P3 | Sprint 14 | **Kommentar `internal/ui/menue.go` „Ebene: 18“ veraltet (20)** (Sitzungs-QA Q-6). | offen (Karte #391) |
| F-981 | [D] | P3 | Sprint 14 | **ADR-0044 Nachtrag 2 benennt nicht, welche Sätze von Nachtrag 1 Punkt 1 bleiben** (Sitzungs-QA Q-7). | offen (Karte #391) |
| F-982 | [V] | P3 | Sprint 14 | **Prüfmuster F-1 in CLAUDE.local.md meldet Falschtreffer in Mutanten-JSON** (Sitzungs-QA Q-8). | offen (Karte #391) |
| F-983 | [V] | P3 | Sprint 14 | **Vorbilder in ADR-0044 ohne Spurensicherung** (Sitzungs-QA Q-9): Verhalten von Snagit, Office, Inkscape u. a. aus Kenntnis, nicht nachgeschlagen. | offen (Karte #391) |
| F-899 | [V] | P3 | Sprint 14 | **Papierkorb mehrerer: Zweig `ErrIndex` ohne Test** (Nacharbeit #361, Kritik-Mutant F): Indexfehler nach gelungenem Verschieben zählt als Erfolg mit Abgleich; kein Test fährt ihn. **Prüfbefehl:** Mutant „ErrIndex-Zweig entfernt“ in `markierung.json` → Soll rot; braucht eine Naht für `InPapierkorb`. | offen (Karte #373) |
| F-889 | [K] | P3 | Sprint 14 | **Papierkorb vieler: Index je Aufnahme ganz gelesen und geschrieben, ohne Fortschritt, ohne Rückfrage; ShowItems ohne Obergrenze** (K-3). Strg+A und Entf in der ungefilterten Ansicht legen die ganze Bibliothek ohne Rückfrage weg (O(n·m)); Schließen wartet 5 s. **Prüfbefehl:** Messung `papierkorbArbeit` über 2000 Einträge in `t.TempDir()`. Entscheidung offen: Rückfrage ab welcher Zahl. | offen (Karte #373) |
| F-196 | [D] | P3 | Sprint 6 | **Export unter Flatpak: Temporärdatei neben der gewählten Datei und angehängte Endung liegen außerhalb des Schreibrechts des Dokumenten-Portals** (Kritik Fable A-9; ADR-0025 Nachtrag 1). **Prüfbefehl:** Export im Flatpak-Bau. **Widerlegungsbedingung:** das Portal gewährt den Ordner. | teilweise behoben c6b0c1f: erlaubt der Ordner keine neue Datei (EACCES, EPERM, EROFS), schreibt der Export ein vorhandenes, reguläres Ziel direkt (vorher vollständig im Speicher gerendert); ADR-0025 Nachträge 3 und 4. Offen, nur im Flatpak-Bau prüfbar: unter dem Dokumenten-Portal keine Endung anhängen, current_name mit Endung je Filter; dazu prüfen, ob der Rückfall (ein Deskriptor mit `O_RDWR` und `O_NONBLOCK`, `Truncate(0)`, F-330, F-338) auf dem FUSE des Portals geht — vor einer Flathub-Einreichung (Karte #134) |
| F-375 | [D] | P3 | Sprint 10 | **Gio-Sign-off unter Pseudonym, Offenlegung weich** (Sitzungs-QA 22.11., Fable, Q-12): `Signed-off-by: Andiministrator <…noreply…>`; DCO-Projekte erwarten oft einen Klarnamen. „written with an AI assistant“ sagt nicht, dass die KI alles schrieb und niemand den Code Zeile für Zeile durchsah. **Prüfbefehl:** erste Review-Antwort in PR #204/#205 enthält die Klarstellung; Klarname nur auf Andis Entscheidung. | backlog (Karte #157; Andi 22.11.2026: Klarname im Sign-off ja, falls Gio danach fragt) |
