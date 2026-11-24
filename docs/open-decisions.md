# Offene Entscheidungen

Stand 21.11.2026. Je Eintrag: Frage · Impact · Tendenz · muss entschieden sein vor.

## E-01: Wann hat der Gio-Prototyp bestanden? — ENTSCHIEDEN 19.11.2026 (ADR-0003 Nachtrag 1)
- **Frage:** Messbare Kriterien für „Gio trägt" (ADR-0003): z. B. Pfeil biegen ohne sichtbares Ruckeln auf 4K, Overlay
  über alle Monitore unter KWin, Reaktionszeit Tastenkürzel → Overlay.
- **Impact:** hoch — entscheidet Go oder Rust.
- **Tendenz:** Kriterien vor dem Prototyp festschreiben (Nachtrag ADR-0003), Andi bestätigt.
- **Vor:** Beginn des Prototyps.

## E-02: KWin-Schnittstelle im Einzelnen — ENTSCHIEDEN 19.11.2026 (ADR-0007 Nachtrag 1, Prototyp-Ergebnis Nachtrag 2; Berechtigung: F-15)
- **Frage:** Welche D-Bus-Schnittstelle (`org.kde.KWin.ScreenShot2`), welche Berechtigung (`.desktop`-Eintrag
  `X-KDE-DBUS-Restricted-Interfaces`?), welches Rückgabeformat, wie „Fenster per Klick", Mauszeiger, mehrere Monitore
  und Skalierung? Ab welcher Plasma-Version?
- **Impact:** hoch — bestimmt, ob Aufnahme ohne Dialog geht und wie installiert werden muss.
- **Tendenz:** aus dem öffentlichen KWin-Quelltext und der Doku prüfen (Spurensicherung), Ergebnis als Nachtrag zu
  ADR-0007.
- **Vor:** Prototyp Schritt 4 (Aufnahme).

## E-03: Name und Aufbau des PNG-Blocks — ENTSCHIEDEN 19.11.2026 (ADR-0008 Nachtrag 1)
- Zwei Blöcke: `snAP` (SVG, komprimiert), `snAO` (Original binär); nicht kopiersicher (4. Buchstabe groß).
- **Frage:** Chunk-Name (privat, nicht sicherheitskritisch, kopierbar?: Groß-/Kleinschreibung der vier Buchstaben
  bestimmt das), komprimiert oder nicht, Original als eingebettetes PNG im SVG oder als eigener Chunk?
- **Impact:** mittel — einmal veröffentlicht, schwer zu ändern.
- **Tendenz:** eigener privater Chunk, SVG komprimiert, Original als `<image>` mit Daten-URI im SVG (ein Block).
- **Vor:** erste Fassung, Dateiformat.

## E-04: Modulpfad und Ort der Zeichenpakete — ENTSCHIEDEN 19.11.2026 (ADR-0012)
- Eigenes Modul `github.com/Andiministrator/schnapp/zeichnen` im Ordner `zeichnen/` (seit 20.11.2026 `…/schnapp/draw` im Ordner `draw/`, ADR-0019 Nachtrag 1).

## E-05: CI — ENTSCHIEDEN 19.11.2026: GitHub Actions (ADR-0010)
- Eingerichtet 19.11.2026: `.github/workflows/test.yml` (gofmt, vet, test für `zeichnen/`), grün. Heute Jobs Wurzelmodul, `draw`, `gio-patch` (`prototype` entfiel mit ADR-0027).

## E-06: Lizenzvariante — ENTSCHIEDEN 19.11.2026: GPL-3.0-or-later (ADR-0009)
- Rechte an den Zeichenpaketen liegen beim Maintainer; Veröffentlichung unter GPL-3.0 erlaubt (Andi, 19.11.2026).

## E-07: Englische Oberfläche und README — ENTSCHIEDEN 19.11.2026 (ADR-0014)
- Oberfläche Englisch + Deutsch ab Start, Systemsprache erkennen, sonst Englisch; Nutzer-Doku zweisprachig.

## E-08: Namensprüfung vor Veröffentlichung — GEPRÜFT 19.11.2026, kein Hindernis
- **Ergebnis:** AUR und Arch-Paketquellen: kein `schnapp`. GitHub: 129 Treffer mit „schnapp“ im Namen, davon ein
  ähnliches Werkzeug `pricerc/schnapp` (periodische Screenshots mit Web-Annotation, 0 Sterne, ohne Paket);
  `schnapps` ist ein Snapshot-Werkzeug von Turris OS (Router, anderes Feld). Flathub-Suche über die API nicht
  auswertbar, Websuche ohne Treffer. Marke nicht geprüft. Vor einer Flathub-Einreichung dort erneut suchen.
- **Frage:** Gibt es „Schnapp" auf GitHub, Flathub, in Debian/Arch/Fedora, als Marke?
- **Impact:** mittel — Umbenennung nach Veröffentlichung ist teuer.
- **Tendenz:** Prüfung kurz vor dem ersten öffentlichen Push, Ergebnis in Spurensicherung.
- **Vor:** Flathub-Einreichung.

## E-09: Bibliothek — Ort und Format — ENTSCHIEDEN 19.11.2026 (ADR-0016)
- `~/Bilder/schnapp/JJJJ-MM/`, Index `~/.local/share/schnapp/bibliothek.jsonl` als Cache, Aufräumen nach einstellbaren
  Tagen in den Papierkorb (abschaltbar, Voreinstellung 90 Tage, nur ohne Schlagworte/Bearbeitung). `schnapp` in Pfaden klein.
- **Frage:** Wo liegen Aufnahmen (`~/Bilder/Schnapp`? XDG-Datenverzeichnis?), Index als Datei oder SQLite (cgo!),
  wie werden Quellprogramm und Schlagworte gespeichert, Aufbewahrung/Aufräumen?
- **Impact:** mittel — Datenschutz (Aufnahmen enthalten oft Vertrauliches), Rechte 0600/0700.
- **Tendenz:** Bilder als Arbeitsdateien unter XDG-Bilderordner, Index als JSON-Zeilen-Datei, kein cgo.
- **Vor:** erste Fassung, Bibliothek.

## E-10: Globales Tastenkürzel unter KDE — ENTSCHIEDEN 19.11.2026 (ADR-0017)
- `.desktop`-Aktionen, Voreinstellung Druck für „Bereich“ (Spectacle-Kürzel abschalten); Portal später.
- **Frage:** Über KGlobalAccel (D-Bus), das Portal `GlobalShortcuts` oder eine `.desktop`-Aktion, die der Nutzer in
  den Systemeinstellungen bindet?
- **Impact:** mittel — Kern der Bedienung („schnell zur Hand").
- **Tendenz:** zunächst `.desktop`-Aktion + Kurzbefehl in den Systemeinstellungen (am einfachsten), später Portal.
- **Vor:** erste Fassung.

## E-11: Board-Dateien im öffentlichen Repo? — ENTSCHIEDEN 19.11.2026: nein
- Board-Dateien und interne Angaben (`CLAUDE.local.md`) stehen in `.gitignore`; das Board ist über die Sicherung des
  Arbeitsrechners gesichert.

## E-12: Lizenz und Nutzungsregeln für Logo und Symbol — ENTSCHIEDEN 19.11.2026 (Andi folgt der Tendenz)
- Code GPL; Logo mit kurzer Markenregel. Umsetzung: Markenregel in `brand/` und README vor der ersten Paket-Veröffentlichung.
- **Frage:** Bleiben Logo und Symbol unter GPL-3.0 wie der Code (jeder darf sie ändern und weiterverwenden), oder
  bekommen sie eine eigene Lizenz (z. B. CC BY-SA) bzw. Regeln zur Markennutzung (Abwandlungen dürfen nicht
  „Schnapp“ heißen)?
- **Impact:** mittel — betrifft Forks und Pakete in Distributionen; nachträglich schwer zurückzuholen.
- **Tendenz:** Code GPL, Logo mit kurzer Markenregel (frei nutzbar, um auf Schnapp zu verweisen; Abwandlungen nicht
  als Schnapp ausgeben) — wie bei vielen freien Projekten.
- **Vor:** erste Veröffentlichung eines Pakets (Flathub).

## E-13: Ansprache in der deutschen Oberfläche — ENTSCHIEDEN 19.11.2026 (Andi folgt der Tendenz)
- Möglichst ohne Anrede formulieren; wo nötig „du“.
- **Frage:** Spricht die deutsche Übersetzung den Nutzer mit „du“ oder „Sie“ an?
- **Impact:** niedrig, aber durchgängig; spätere Umstellung betrifft jeden Text.
- **Tendenz:** möglichst ohne Anrede formulieren („Bild gespeichert“, „Bereich wählen“); wo nötig „du“ (freundlich,
  passt zur Haltung im Corporate Design).
- **Vor:** erste Oberflächentexte (Karte Übersetzbarkeit).

## E-14: Schnittstelle für Agenten und lokale KI (Kommandozeile, MCP-Server) — ENTSCHIEDEN 19.11.2026 (Andi folgt der Tendenz)
- Kommandozeile mit `--output`/`--json` in der ersten Fassung, `schnapp mcp` (nur stdio, Einwilligung getrennt für Aufnehmen und Bibliothek-Lesen, Hinweis bei jeder MCP-Aufnahme) in der zweiten, kurze englische `docs/agents.md`. Umsetzung: ADR zuerst (Karte #43). ADR-0030 (22.11.2026): Fassung 1 umgesetzt (`--json`, `--output=DATEI`, `--export=ZIEL DATEI`, `docs/agents.md`), Fassung 2 (MCP) entschieden, eigene Karte.
- **Frage:** Bekommt Schnapp von Anfang an eine Schnittstelle, über die KI-Agenten und lokale Modelle aufnehmen,
  beschriften, schwärzen, exportieren und die Bibliothek durchsuchen können — als skriptfähige Kommandozeile, als
  MCP-Server, beides? Braucht es dafür eine eigene Doku für Agenten? (Andi, 19.11.2026)
- **Impact:** hoch — prägt die Architektur (Kern ohne Oberfläche) und ist sicherheitsrelevant: Wer die Schnittstelle
  bedienen darf, sieht den Bildschirm und die Bibliothek.
- **Tendenz:**
  1. **Architektur ab jetzt API-fähig** (ist schon so: dokument, datei, aufnahme, bibliothek kennen Gio nicht).
  2. **Erste Fassung: skriptfähige Kommandozeile** als erste Schnittstelle, ohne Fenster, z. B.
     `schnapp --screen=NAME --output bild.png`, `schnapp export`, `schnapp redact --area x,y,b,h`; Ausgabe
     maschinenlesbar mit `--json`. Kostet wenig, nützt Skripten und Agenten gleichermaßen.
  3. **MCP-Server als `schnapp mcp`** (nur stdio, nie Netz) in der zweiten Fassung, auf derselben Kernschicht:
     Werkzeuge aufnehmen, Bibliothek suchen/lesen, Objekte hinzufügen, schwärzen, exportieren. Eigene ADR davor:
     **Einwilligung** (in den Einstellungen ausdrücklich einschalten, getrennt für Aufnehmen und Bibliothek-Lesen),
     sichtbarer Hinweis bei jeder Aufnahme über MCP, Fremdpaket für MCP nach ADR-0011 abwägen.
  4. **Doku für Agenten:** keine getrennte große Doku. Die MCP-Werkzeuge beschreiben sich selbst; dazu eine kurze
     englische `docs/agents.md` (Kommandozeile, MCP, Beispiele, Grenzen) für Menschen und Agenten.
- **Vor:** Zuschnitt der Fassung 2; die Kommandozeilen-Form vor der ersten Veröffentlichung (sie ist Oberfläche).

## E-15: Dokumentation des Bildformats (intern und öffentlich) — ENTSCHIEDEN 19.11.2026 (Andi folgt der Tendenz)
- Eine öffentliche, normative englische Spezifikation `docs/format.md` (ohne deutsche Fassung) mit Beispieldateien und Konformitätstest; intern bleibt ADR-0008 die Entscheidung (Karte #44).
- **Frage:** Brauchen wir eine Beschreibung des Arbeitsdatei-Formats (PNG mit `snAP`/`snAO`), intern und/oder
  öffentlich? (Andi, 19.11.2026)
- **Impact:** mittel — Dateien gehen nach außen; ohne Spezifikation kann niemand sie verlässlich lesen, und spätere
  Formatversionen brauchen einen festen Bezugspunkt (ADR-0006).
- **Tendenz:** **eine** öffentliche, normative Spezifikation `docs/format.md` (englisch, für Entwickler — Ausnahme
  von der Zweisprachigkeit nach ADR-0014, weil Fachdoku; deutsche Fassung nur auf Wunsch) mit Formatversion,
  Blöcken, SVG-Elementen und -Attributen, Grenzen, Regeln für Leser („unbekannte Version ablehnen“) und kleinen
  Beispieldateien unter `testdata/`; ein Test prüft, dass der Schreiber den Beispielen entspricht. Intern bleibt
  ADR-0008 (mit Nachtrag) die Entscheidung; keine zweite interne Beschreibung, die auseinanderläuft.
- **Vor:** erste Veröffentlichung eines Pakets; jede Formatänderung danach ändert Spezifikation und Version im selben
  Commit.

## E-16: Ablageort beim Schließen einer Datei von außen — ENTSCHIEDEN 20.11.2026 (Claude, Andi informiert)
- **Frage:** Wird eine Datei von außen ohne Strg+S geschlossen, landet die Arbeit als neue Aufnahme in der Bibliothek.
  Reicht die Meldung auf stderr, oder muss der Ablageort sichtbar werden (F-158)?
- **Entscheidung:** sichtbar — der Hinweis mit dem Ablageort hält das Fenster einmal offen, wie ein Ausweichen mit
  Warnung (ADR-0020 Nachtrag 4, Karte #85).

## E-17: Dürfen Verschieben und Griffe ein Objekt ganz aus dem Bild bringen? — ENTSCHIEDEN 20.11.2026 (Claude, Andi informiert)
- **Frage:** F-150 verhindert nur die Neuanlage außerhalb; ein vorhandenes Objekt ließ sich ganz aus dem Bild ziehen
  (unsichtbar, aber gespeichert, F-160).
- **Entscheidung:** nein — das Objekt muss das sichtbare Bild um einige Pixel schneiden; die Verschiebung wird begrenzt
  (ADR-0021 Nachtrag 2, Karte #85).

## E-18: Umschalt am Bildrand — Form oder Begrenzung? — ENTSCHIEDEN 20.11.2026 (Claude, Andi informiert)
- **Frage:** Am Rand je Achse begrenzt, verliert der Pfeil seine 45° und der Kasten sein Quadrat (F-156, F-157).
- **Entscheidung:** Form halten — entlang der Richtung kürzen, gerastet wird in Ansichtskoordinaten, damit kein
  Endpunkt in einem herausgeschnittenen Streifen liegt (Karte #84).

## E-19: Soll ein kopiertes Bild das Schließen des Editors überdauern? — ENTSCHIEDEN 20.11.2026 (Andi): ja
- **Frage:** Unter Wayland gehört der Inhalt der Zwischenablage dem Fenster, das ihn anbietet (ADR-0015 Nachtrag 2).
- **Befund:** Bei Andi ist das Bild nach dem Schließen des Editors weg — Klipper übernimmt es nicht (Andi, 20.11.2026;
  der frühere Eindruck „Klipper hält es“ kam von einem noch offenen Editor).
- **Entscheidung (Andi):** „ich würde das schon gern komplett in der zwischenablage haben und behalten, wenn man den
  editor schließt“. Umsetzung in Sprint 6, Karte #91; der Weg (z. B. Prozess hält die Auswahl ohne Fenster weiter, bis
  eine andere Anwendung sie übernimmt, mit harter Zeitgrenze; oder data-control-Protokoll) erst per ADR.
- **Umgesetzt 20.11.2026:** ADR-0024 — Halter-Prozess über data-control, höchstens 12 Stunden, Rückfall auf Gio.

## E-20: Claudes Entscheidungen aus Sprint 6 — ENTSCHIEDEN 20.11.2026 (Claude, Andi informiert)
- Bild-Typ in der Zwischenablage per Gio-Patch statt `wl-copy` (ADR-0015 Nachtrag 2).
- Schwärzen (B) und Verpixeln (P) überschreiben die Pixel; Verpixeln ist ein Mosaik ohne Inhaltsbezug (ADR-0022 mit
  Nachtrag 1; Andi: „klingt doch gut (sicher)“). Nach einem gescheiterten Zurückschreiben wird die alte, ungeschwärzte
  Datei endgültig entfernt (F-126).
- Zuschneiden (C) und Stauchen (X) mit Zackenkante an Stauch-Nähten; der Schwärzungsrahmen bleibt auf die Ansicht
  begrenzt, Weggeschnittenes bleibt im Original — vorher schwärzen (ADR-0023).

## E-21: Stauch-Streifen am Bildrand und Bedienung ohne Tastatur — ENTSCHIEDEN 20.11.2026 (Andi)
- **Frage (F-222):** Ein Stauch-Band, das bis an den Bildrand reicht, hat keine Naht; nach dem Speichern mit Schwärzung
  oder Wiederöffnen gibt es keinen Rückweg. Optionen: (1) Rand-Band wird zum Zuschnitt, zurück wie jeder Zuschnitt,
  (2) Doppelklick mit X nahe dem Rand, (3) eigene Aktion „Alles Herausgenommene wieder zeigen“.
- **Entscheidung (Andi):** „ich folge deiner empfehlung“ — (1): ein X-Band bis an den Bildrand wird als Zuschnitt
  gespeichert. Umsetzung Karte #99 mit ADR-0023-Nachtrag.
- **Grundsatz dazu (Andi):** „alles was mit tastenkürzeln geht, muss auch über die werkzeugleiste oder das menü gehen.“
  Das gilt auch für das Aufheben des Zuschnitts (heute nur Doppelklick mit C) und für das Entfernen einer Naht. Bestand
  prüfen und nachrüsten: Karte #104 „Jede Tastenfunktion auch über Leiste oder Menü“ (Backlog); neue Funktionen bekommen
  Knopf oder Menüeintrag im selben Zug.
- **Umgesetzt 20.11.2026:** ADR-0026 samt Nachtrag 1 (Menü mit allen Aktionen und Kürzeln, Tastentabelle
  `internal/ui/kuerzel.go`, „Gestauchtes zurückholen“), Karte #104, Commits 24bc96d und 34dd5ee. F-255 (Menü per Tastatur) erledigt in Sprint 8: ADR-0026 Nachträge 2 und 3, Commits 8cd134d und 0385204.

## E-22: Claudes Entscheidungen aus Sprint 8 — ENTSCHIEDEN 20.11.2026 (Claude, Andi informiert)
- Speichern und Kopieren laufen im Hintergrund auf einem Abzug; während des Speicherns ruht das Bearbeiten
  (ADR-0020 Nachtrag 6). Reste: F-280, F-281, F-283, F-285.
- Menü allein per Tastatur: Pfeile, Pos1, Ende, Umlauf an den Enden, Leertaste wie Enter; Fokusring am Menüeintrag
  innen statt außen (ADR-0026 Nachträge 2 und 3).
- Neue Aufnahmen werden exklusiv per Hardlink angelegt, Rückfall O_EXCL (F-48); der Index-Neuaufbau liest nur
  Metadaten (ADR-0016 Nachtrag 4).
- **Eigene tmp-Reste werden entfernt, nicht nur gemeldet** (Andi, 20.11.2026: „das überlasse ich dir“; Claudes
  Entscheidung): Neuaufbau und Speichern entfernen eigene Reste `.schnapp-tmp-*` ohne Rückfrage aus der Ablage
  (ADR-0016 Nachtrag 3, F-274). Grund: Ein Rest kann ein zweiter Name einer später geschwärzten Aufnahme sein; nur
  melden ließe den ungeschwärzten Inhalt liegen. Getroffen werden nur reguläre Dateien mit genau diesem Präfix, nie
  Symlinks, Ordner, Aufnahmen oder Vorschaubilder.
- Fristen beim Schließen: 15 s bis „Speichern antwortet nicht“, 2 min beim Zerstören ohne Anfrage, 10 s für ein
  laufendes Kopieren (ADR-0020 Nachtrag 7). Die 15-s-Frist hat Andi am 21.11.2026 eingeschränkt (Nachtrag 9, Umsetzung
  offen, Karte #119).
- Strg+C wirkt auch während des Speicherns (ADR-0020 Nachtrag 6, F-285); seit Nachtrag 8 ebenso Strg+W und der
  Menüeintrag „Schließen“ (F-299).
- Exit-Code: Bleiben Änderungen ungesichert (Speichern hängt oder scheitert unverworfen), endet Schnapp mit Fehler —
  auch nach einer Aufnahme; ein Fensterfehler allein bleibt dort Exit 0 (ADR-0020 Nachtrag 8, F-302). Für Skripte
  sichtbar.
- Menü-Navigationstasten stehen als benannte Ausnahme neben der Tastentabelle (ADR-0026 Nachtrag 4); der Wecker fällt
  ohne Quittung auf bis zu 800 ms zurück (F-258).

## E-23: Deutsche Paketnamen unter internal/ — ENTSCHIEDEN 21.11.2026 (Claude im Auftrag Andis: „entscheide du das“)
- **Frage:** `internal/speicherschritt` (Sprint 9) und `internal/probeschutz` widersprechen ADR-0019 (englische
  Verzeichnis- und Paketnamen). Umbenennen oder Ausnahme als Nachtrag zu ADR-0019? (F-312)
- **Entscheidung:** umbenennen, keine Ausnahme: `internal/saveprogress`, `internal/probeguard`. ADR-0019 regelt den
  Fall schon; eine Ausnahme hätte keine Begründung außer dem Umbauaufwand, und der ist klein (Karte #126).
  Exportierte Bezeichner (`Marke`, `Puls`, `Zuletzt`, `NurVirtuell`) bleiben, ADR-0019 erlaubt deutsche Bezeichner.

## E-24: Claudes Entscheidungen aus Sprint 10 — ENTSCHIEDEN 21.11.2026 (Claude, Andi informiert)
- **Kein Dependabot** für die CI-Aktionen: keine Bot-PRs im öffentlichen Repo; stattdessen Prüfpunkt „CI-Pins prüfen“
  beim Sprint-Abschluss (F-316, `docs/qa/sprint-abschluss.md`).
- **Verlaufsgrenze 512 MiB, still:** Der Rückgängig-Verlauf verwirft die ältesten Schritte ohne Meldung, sobald die
  Pixelkopien 512 MiB übersteigen (ADR-0022 Nachtrag 3, F-64, F-323).
- **Export-Rückfall:** Darf neben dem Ziel keine temporäre Datei entstehen, wird eine vorhandene, beschreibbare Datei
  direkt beschrieben — mit Sicherung des alten Inhalts, ohne `O_TRUNC`, nach Identitätsprüfung (ADR-0025 Nachträge 3
  und 4, F-196, F-322, F-330). Der Rückfall bleibt auch im gewöhnlichen schreibgeschützten Ordner.
- **Warnung beim Schließen ohne Halter:** einmal anhalten statt Dialog, auch nach dem Speichern und bei laufendem
  Kopieren (ADR-0024 Nachtrag 6, F-295, F-321, F-329).
- **Gio-Patch nur als Pull Request** am GitHub-Spiegel, in zwei PRs; einreichen tut Andi (Karte #17,
  `knowledge/topics/plattformen-und-ki-regeln.md` §Gio).
- **EXIF:** Die Ausrichtung wird beim Öffnen angewendet, die Angabe selbst verworfen (ADR-0017 Nachtrag 7).
- **Altdatei beim Speichern weiter voll gelesen** statt nur über `LeseMetadaten` (ADR-0020 Nachtrag 11, F-310).
- **Eigener Rasterkern:** Fork von `golang.org/x/image/vector` mit 16-Bit-Maske statt Fremdpaket-Wechsel (ADR-0028);
  die Viererfolge ist fest, ohne `x/sys/cpu` (ADR-0028 Punkt 2).
- **Fehlertexte:** Präfix-Regel gilt für alle Pakete, keine Ausnahmen; fünf Pakete werden nachgezogen (ADR-0019
  Nachtrag 2, F-339).
- **Rettungskopie gegen Absturz:** ADR-0029 akzeptiert (Claude im Auftrag Andis, 21.11.2026), umgesetzt in Sprint 11 (F-144, Karte #77; Nachträge 1 bis 10).

## E-25: Claudes Entscheidungen beim Sprint-Abschluss 12 — ENTSCHIEDEN 24.11.2026 (Claude im Auftrag Andis: „entscheide du“)

- **Agenten `edit` + `library`** (F-699): keine eigene Einwilligung; die Lesewirkung entsteht nur mit beiden Schaltern
  zusammen, wer sie nicht will, schaltet einen aus. Wirkung dokumentiert, neue Aufnahmen unter `agents/` (ADR-0030
  Nachtrag 13 Punkt 5, berichtigt F-741).
- **Text an der Stauch-Naht** (F-739): Grenze bis Karte #281 (ADR-0023 Nachtrag 12; gelöst in Sprint 13, Nachtrag 13).
- **`.claude/settings.json` im öffentlichen Repo** (Abschluss-Kritik D-11): bleibt mit der Sperrregel, die den Zweig
  `alt-main` beim Namen nennt. Die Regel verhindert, dass die alte Geschichte versehentlich veröffentlicht wird; dass es
  einen zurückgehaltenen Zweig gibt, verrät nichts, was in ihm steht. Ohne den Namen griffe die Sperre nicht.
- **Echtlauf-Skript** `scripts/echtlauf-mcp-hinweise.sh` (F-717): auf Andis Ja entfernt.

## E-26: Entscheidungen aus Sprint 11 und Sprint 12 — ENTSCHIEDEN 22. bis 24.11.2026 (Andi bzw. Claude, Andi informiert)

Nachgetragen in Sprint 13 (F-723); die Einzelheiten stehen in den genannten ADRs.

- **Agenten-Schnittstelle:** MCP-Server `schnapp mcp` über stdio, Einwilligung je Fähigkeit in `[agents]` (Aufnahme,
  Bibliothek, Beschriften aus; Bestätigung und Bild an), Bestätigungsdialog nur in der geprüften KWin-Sitzung
  (ADR-0030 Nachträge 2 bis 13).
- **Neue Oberfläche:** Textmenü mit „s“, Werkzeugleiste links, rechte Leiste mit Eigenschaften, Fußzeile mit Feldern,
  eckige Formen als Voreinstellung (ADR-0031, ADR-0021 Nachtrag 4; Symbolset von Andi am 23.11. abgenommen).
- **Ebenen** je Objekt mit Name, Sichtbarkeit, Sperre und Deckkraft (ADR-0032); **Lupe** vergrößert die Ansicht
  (ADR-0033); **Einstellungstafel** im Editor (ADR-0034); **Schnellstile** (ADR-0037).
- **Text aus dem Bild:** installiertes `tesseract`, sonst das eigene `schnapp-ocr` mit tesseract als WebAssembly und
  eingebauten Sprachdaten deu/eng, ohne cgo (ADR-0035 mit Nachträgen 1 und 2).
- **Bibliothek im Editor:** Leiste der letzten Aufnahmen und volle Ansicht, Wechsel der Aufnahme im selben Fenster
  (ADR-0036).
- **Stauch-Kante:** Abriss als Voreinstellung, durchsichtiger Spalt (ADR-0023 Nachträge 6 und 7).

## E-27: Claudes Entscheidungen aus Sprint 13 — ENTSCHIEDEN 24.11.2026 (Claude, Andi: „arbeite selbstständig“)

- **Beschriftungen an einer Stauch-Naht** sind in allen Ausgängen abgeschnitten wie das Bild (Text, Sprechblasen-
  Dreieck, Lupen-Linie, Pfeil, Nummer), statt gestaucht oder starr hinter der Naht (ADR-0023 Nachträge 13 bis 15,
  F-739, F-695, F-745). Keine Texteingabe an ganz verborgenem Text (F-740).
- **Schriftgrenze** als Zwischenspeicher mit Verdrängung statt lebenslanger Sperre (ADR-0021 Nachtrag 15, F-697).
- **Index der Bibliothek:** Abgleich ergänzt Aufnahmen ohne Zeile, Ablegen an fehlenden Index baut ihn neu auf
  (ADR-0016 Nachtrag 7, F-700).
- **Übersicht F1** ist modal wie die Einstellungstafel; beide Tafeln rollen in kleinen Fenstern (ADR-0031 Nachträge 10
  und 11, F-705, F-707, F-709).
- **Rettungskopie:** Aufräumen lässt Kopien offener Fenster liegen, Schlagworte ziehen die Kennung nach (ADR-0029
  Nachtrag 10, F-702, F-703).
- **CI:** übersprungene Tests sichtbar mit Liste erlaubter Sprünge, Echt-Tests mit tesseract, rsvg-convert und fester
  Open Sans, `govulncheck` als eigener Job; `golang.org/x/image` auf v0.48.0 gehoben (F-714, F-718, F-744).
- **Lizenztexte** der eingebauten Go-Module in `THIRD-PARTY-LICENSES/`, per Test gleich der Modulliste (F-719).
- **Testdateien** heißen nach ihrem Gegenstand, jede eigene Quelldatei trägt den SPDX-Kopf; ein Test hält beides
  (F-725).

