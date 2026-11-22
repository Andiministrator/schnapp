# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> Verweise auf ADR-0043, ADR-0062, ADR-0067, F-Nummern, die weder in `docs/findings.md` noch in
> `docs/findings.archive.md` stehen, und Pfade unter `~/.aboard-framework/` oder
> `~/.claude/` beziehen sich auf das Arbeits-Framework (aBoard) des Maintainers, nicht auf dieses Repo. Projekteigene
> ADRs liegen unter `docs/adr/` (0001–0030), projekteigene Findings in `docs/findings.md` (erledigte
> in `docs/findings.archive.md`).

## Vor jeder Antwort — Pflicht

**Hook-Output auf `AKTION VOR ANTWORT`-Blöcke scannen — falls vorhanden: zuerst ausführen, DANN antworten.**

- Gilt auch bei kurzen Acknowledgments („ok", „sehr gut", „danke") — kein Überspringen
- Die erste Zeile der Antwort ist die Ausführung, kein erklärender Text davor

## Board-Checkpoints

An diesen Punkten immer `aboard-sync.json` neu lesen — nie aus dem gecachten Zustand urteilen:

| Zeitpunkt | Was prüfen |
|---|---|
| Session-Start | Vollständiger Board-Check (→ Session-Einstieg) |
| Nach Karten-Abschluss | Ready-Spalte neu lesen — was liegt noch an? |
| Vor „los"/Arbeitsauftrag | Aktuellen Ready-Inhalt bestätigen bevor begonnen wird |
| Sprint-Abschluss | Board-Konsistenz: alle Karten qa:true + col-erledigt? |

> **Board-Dateien sind nicht versioniert** (E-11, Andi 19.11.2026): `aboard.html`, `aboard-sync.json`, base, Inbox
> stehen in `.gitignore`, weil das Repo öffentlich wird. `git add` auf Board-Dateien entfällt also; die Kurzregel
> „drei Board-Dateien gemeinsam stagen“ gilt hier nicht.

## Context-Management

Den eigenen Context-Verbrauch im Blick behalten (viele Dateien gelesen, viel Code generiert, lange Session = Context wird voll).

**Regel:** Wenn ~50%+ Context verbraucht und weitere umfangreiche Arbeit geplant → sauberen Schnitt an natürlichem Haltepunkt einplanen. Aktuellen Schritt zu Ende führen + committen, dann `/chat-wechsel` ausführen → legt die Übergabe im Board (`▶ HIER STARTEN`-Kommentar an der Einstiegskarte) und im Memory ab. Kein Oneliner zum Kopieren — der nächste Chat findet sie ohne Vorgabe. Nie mitten in einer Änderung abbrechen.

## Session-Einstieg

**Zu Beginn jeder neuen Session — immer ausführen:**

1. `aboard-sync.json` lesen (aBoard Projekt-Modus, migriert)
2. `docs/findings.md` prüfen — gibt es P0-Blocker?
3. `docs/open-decisions.md` prüfen — blockiert etwas den nächsten Schritt?
4. `git log --oneline -5` — was passierte in der letzten Session?
5. **Topic-Index der Wissensbasis lesen** (`knowledge/README.md`) — die Tabelle, nicht die Topic-Dateien; eine Synthese vollständig lesen, wenn die anstehende Arbeit ihr Thema berührt
6. Claude-zugewiesene Karten ohne `complexity`/`effort` → sofort schätzen und eintragen

> **Hinweis:** Der globale UserPromptSubmit-Hook (`~/.claude/hooks/aboard-context.sh`) injiziert den Board-Zustand automatisch als Kontext, wenn `aboard-sync.json` im Projektverzeichnis liegt. Kein manuelles Lesen nötig sobald aBoard im Projekt-Modus läuft.

Für einen vollständigen Compliance-Check: `/projekt-start`

Bei Unklarheit über den aktuellen Stand: **erst fragen, nicht raten.**

## Sub-Agents

QA-Checks, Kritiker-Läufe und aufwendige Analysen werden als **Sub-Agents** ausgeführt (Agent-Tool) — schont den Hauptchat-Context, vermeidet Bias beim Kritiker.

**⚠️ Sub-Agents haben keinen Memory-Zugriff.** Jeder Sub-Agent-Prompt muss vollständig selbst-enthaltend sein: alle Regeln explizit, alle Infos als Dateipfade. Niemals implizit auf Memory-Wissen verweisen.

**Was in jeden Sub-Agent-Prompt wörtlich gehört (F-2455):** (1) der Block „⛔ ABSOLUTE AUFLAGE — lokale Werkzeuge mit Geheimnis- oder Personenbezug“ samt Meldepflicht aus `~/.claude/commands/sprint-abschluss.md` Phase 2 (kopieren, nicht nachbauen); (2) „Nichts nach /tmp“ mit dem erlaubten Ort `tmp/<…>/` im Projekt; (3) keine gemeinsam beschriebenen Dateien, keine git-Eingriffe über die eigene Datei hinaus, kein Build — jeweils mit erlaubtem Weg; (4) neue Findings nur als Text mit Kartenvorschlag; (5) kein `rm` außerhalb von `tmp/<sprint>/<lauf>/`, auch dort nur mit ausgeschriebenem Pfad (keine Globs, keine Variablen), Testläufe und Stubs mit eigenem HOME; (6) Testdaten mit Shell-Befehlen nur mit dem Write-Werkzeug oder aus Python, nie per Heredoc (F-2548). Das gilt für **jede** Delegation, auch die einer Karte an das Owner-Modell, nicht nur für Kritiker-Läufe. Der Hook `agent-prompt-check.py` weist einen Agent-Start ohne (1) oder (2) zurück.

**Was für Sub-Agents gilt — im Prompt ausdrücklich verbieten, nicht auf Umsicht hoffen:**
1. **Keine gemeinsam genutzten Dateien schreiben**, vor allem nicht den Finding-Backlog (`docs/findings.md`). Der Sub-Agent liefert seinen Eintrag als fertigen Text im Abschlussbericht, der Orchestrator trägt ihn ein. Grund: `git add <datei>` staged die Datei als Ganzes und nimmt fremde Zeilen in einen anderen Commit mit; `git add -p` gibt es im Agenten-Betrieb nicht.
2. **Keine git-Befehle, die über die eigene Datei hinausgreifen** — kein `git stash`, `git checkout <datei>`, `git reset`, `git clean`. Sie wirken auf den gemeinsamen Arbeitsbaum (F-696). Rückbau einer Gegenprobe: Datei vorher kopieren, danach zurückkopieren, mit `git diff` nachweisen.
3. **Gegenrichtung (F-779):** Solange ein Lauf mit definiertem Prüfgegenstand aktiv ist, schreibt auch der **Orchestrator** nicht in die Dateien, die er dem Agenten als Lektüre genannt hat. Änderungen sammeln, nach dem Lauf einspielen.
4. **Kein Build, wenn mehr als ein Strang läuft (F-790).** `go build ./...` und `go test ./...` fangen den ganzen Baum ein, auch die halbfertige Arbeit der anderen. Der Orchestrator baut einmal am Ende; wer zwischendurch bauen muss, baut in einer Kopie.
5. **Jedes Verbot nennt den erlaubten Weg (Fehlerklasse 106).** Wer Dateien oder `rm` verbietet, gibt für Mess-Skripte `tmp/<sprint>/<lauf>/` frei (bleibt liegen, der Orchestrator räumt ab).

**Cherry-Pick aus einem Worktree-Sub-Agent nimmt nur Quelldateien mit, nie eingecheckte Build-Artefakte.** Gilt hier für generierte Dateien (z. B. Stempel-Index), sobald es welche gibt: nach dem Cherry-Pick im Hauptbaum neu erzeugen.

## Sprache

Deutsch für projektinterne Doku, Kommentare, Commits und ADRs (ADR-0002). **Oberfläche und Nutzer-Doku zweisprachig**
(ADR-0014): Oberflächentexte englisch als Ausgangssprache, Übersetzung Deutsch über PO-Dateien, Sprache vom System,
sonst Englisch; `README.md`/`CONTRIBUTING.md` englisch, deutsche Fassung als `*.de.md`, beide im selben Commit pflegen. **Verzeichnisse und Go-Pakete sind englisch**
(ADR-0019; `zeichnen` → `draw` seit 20.11.2026, Nachtrag 1). Go-Bezeichner (Typen,
Funktionen) dürfen deutsch oder englisch sein, innerhalb eines Pakets einheitlich; Fremd-APIs (Gio, KWin, D-Bus)
behalten ihre Namen. Die Oberfläche ist von Anfang an übersetzbar anzulegen (Texte nicht im Code verstreuen), weil das
Repo öffentlich wird.

## Projektüberblick

**Schnapp** (Befehl `schnapp`) ist ein Screenshot- und Beschriftungswerkzeug für Linux/Wayland, Vorbild Snagit:
schnell aufnehmen, beschriften (Pfeile auch gebogen, Kästen, Schritt-Nummern, Stempel), schwärzen, stauchen, teilen —
und nebenbei ein einfaches Bildbearbeitungsprogramm. Zielgruppe zuerst Andi selbst (KDE), später öffentlich
(GPL-3.0, GitHub, ggf. Flathub). ksnip und Spectacle sind zu umständlich und können zu wenig.

Herkunft: Anforderungen und Entscheidungen des Maintainers vom 18.11.2026, übernommen in [`docs/anforderungen.md`](docs/anforderungen.md) sowie ADR-0003 und ADR-0007 bis ADR-0013.

## Kernprinzipien

1. **Nicht zerstörend.** Original bleibt erhalten; Beschriftungen sind Vektorebenen, Stauchen ist eine Angabe, kein
   Schnitt in den Pixeln (ADR-0008).
2. **Schwärzen ist endgültig.** Schwärzen und Verpixeln überschreiben die Pixel auch im mitgespeicherten Original —
   die einzige gewollte Ausnahme von Prinzip 1. Ein geschwärzter Inhalt darf in keiner gespeicherten oder
   exportierten Datei rekonstruierbar sein (ADR-0008, [S]-Pfad).
3. **Keine Spuren nach außen.** Keine Metadaten mit lokalen Pfaden, Programm- oder Rechnernamen; Autor einstellbar.
   Das Repo ist öffentlich: keine Kunden-, Firmen- oder internen Inhalte in Code, Kommentaren, Testbildern.
4. **Schnell zur Hand.** Bereich ist Voreinstellung, ein Tastenkürzel, sofort in der Bibliothek gespeichert — Bedienung
   vor Funktionsfülle.
5. **Austauschbare Aufnahmeschicht.** KDE/KWin zuerst, Portal später; der Rest des Programms kennt nur die Schicht
   (ADR-0007).
6. **Fremdpakete nur mit spürbarem Nutzen** (ADR-0011).

## Architektur

Produktcode (Stand 22.11.2026, v0.5.0 nach Sprint 10): `capture`, `document`, `fileformat`, `library`, `settings`, `selection`,
`canvas`, `render` stehen mit Tests, dazu Overlay und Editor in `ui` (Tests für Sperre, Menü, Schließen und Tastenfilter ohne Fenster über `input.Router`; Zeichnen selbst ungetestet, Logik liegt in `selection`/`canvas`);
Schnapp nimmt auf, legt in der Bibliothek ab und öffnet den Editor (Auswählen, Verschieben, Griffe, Zoom, Rückgängig,
Speichern, Zeichenwerkzeuge Pfeil/Kasten/Text/Schritt-Nummer mit Farbwahl, ADR-0021; Schwärzen/Verpixeln ADR-0022;
Zwischenablage Strg+C, ADR-0015 Nachtrag 2, bleibt nach dem Schließen über den Halter `wlclip`, ADR-0024; Zuschneiden C und Stauchen X mit wählbarer Kante an der Naht (Abriss mit Spalt und Schatten als Voreinstellung, Welle, Zacken; Menüeintrag „Kantenform“ reihum, Einstellung `[capture] seam`, ADR-0023 Nachtrag 6), Band bis an den Rand wird Zuschnitt, zurück über den Knopf „Ganzes Bild“, ADR-0023; Export PNG/JPG/SVG über den Portal-Dialog, Strg+Umschalt+E,
ADR-0025; Menü mit allen Aktionen und Kürzeln, auch allein per Tastatur, Tastentabelle `internal/ui/kuerzel.go`, ADR-0026 mit Nachträgen 2 bis 4; seit Sprint 8: Speichern und Kopieren im Hintergrund, ADR-0020 Nachträge 6 bis 8; Nachtrag 9 — Frist erkennt Fortschritt über `internal/saveprogress`, umgesetzt in Sprint 9). Probeläufe ohne Andi: `scripts/probe.sh`
(virtueller KWin, `knowledge/topics/lehren.md`). Installation per `install.sh` (`~/.local`, Version aus `git describe`, ohne erreichbaren Tag `0.0.0-dev+<Kurzhash>[.dirty]`, ADR-0006 Nachtrag 1; getestet von `scripts/install-test.sh` mit go-Attrappe, in der CI, F-324). Daneben
der Wegwerf-Prototyp `prototype/` (entfernt in Sprint 9, ADR-0027; Stand in der git-Geschichte). Alle Programmpakete liegen seit Sprint 9 unter `internal/` (ADR-0027; Modulpfad
`github.com/Andiministrator/schnapp/internal/<paket>`), öffentlich ist nur das Modul `draw/`. Schichten:

- **capture** — Schnittstelle + KWin-Umsetzung (D-Bus), später Portal; Transport als Naht `dbusTransport` (Deskriptor-Ebene mit Attrappe getestet; gegen echtes KWin ist memfd/ScreenShot3 ungeprüft, F-56). `Quelle.Alle` setzt alle Monitore über KWin `CaptureWorkspace` zu einem Bild zusammen: `[capture] screen = all` ist damit umgesetzt (ADR-0017 Nachtrag 7, F-119).
- **document** — Ebenenmodell (Pixelebene, Vektorebenen, Original, Stauchangaben), Rückgängig-Verlauf; `Abzug` (unabhängige Kopie mit gleichem Stand, ohne Verlauf, für Speichern und Kopieren im Hintergrund), `PruefeInhalt` (Objekte, Zuschnitt, Streifen ohne Pixel prüfen, gleiche Grenzen wie das Laden). Rückgängig-Verlauf höchstens `verlaufGrenze` = 512 MiB Pixelkopien, die ältesten Schritte fallen still weg, der jüngste bleibt (ADR-0022 Nachtrag 3, F-64). Text-Schild = gemessene Tinte + 0,25 Schriftgrad (`TextInnenabstand`), `WirksamerEckradius`, `Nummer.Grenzen` mit Ziffern: Renderer, SVG und Editor setzen gleich (ADR-0021 Nachtrag 3, F-108). Kante der Stauch-Naht in `kante.go` (ADR-0023 Nachtrag 6): Form je Dokument `Kante` (tear/wave/zigzag; `SetzeKante` als Rückgängig-Schritt, `VorgabeKante` ohne Schritt nur bei leerem Verlauf; `Neu` legt tear an), `Kantenband(form, naht, länge, faktor, von, bis)` mit Querlagen in ganzen Hundertsteln und `KantenSchichten` (Spalt, Schatten, Kantenabdunklung): Renderer, SVG-Export und Editor zeichnen je Naht alle Schichten; `ZackenBandHalb` = 11.
- **fileformat** — Arbeitsdatei PNG mit Blöcken `snAP`/`snAO` lesen/schreiben, öffentlich spezifiziert in `docs/format.md` (englisch, normativ; Beispiele `testdata/format/`, Konformitätstest `format_konform_test.go`). Spezifikation, `FormatVersion` und Beispiele ändern sich im selben Commit, Beispiele nur mit `go test ./internal/fileformat -run TestFormatBeispiele -update` neu erzeugen (ADR-0008 Nachtrag 3); Exporte SVG, PNG/JPG (PDF noch offen); `LeseMetadaten` liest nur `snAP` und die Köpfe, keine Pixel (Neuaufbau des Index), mit demselben Urteil und derselben Größe wie das volle Lesen (F-272), ohne Vorabreservierung angekündigter Blocklängen (F-273). `SchreibeArbeitsdatei` schreibt in 1-MiB-Stücken (mit `Sync` und Puls je Stück, ADR-0020 Nachtrag 10) und kodiert Bild und Original nebenläufig (Nachtrag 11). Ein JPEG, dessen Öffnen geschätzt mehr als 2 GiB anforderte (progressiv: Koeffizienten), lehnt `Oeffne` vorab ab (`jpegbedarf.go`, ADR-0017 Nachtrag 8, F-361). Ein JPEG mit EXIF-Ausrichtung 2 bis 8 wird beim Öffnen gedreht, mit genau einer Vollbild-Kopie, die Angabe selbst fällt weg (ADR-0017 Nachtrag 7); `Ausrichten` liest über `image.RGBA64Image` (`pixelLeser`), `blockStrom.Read` mit leerem Puffer liefert 0 (F-323, F-332). Kantenform als `s:kante` an `s:ansicht` (fehlend oder unbekannt: zigzag, kein Schaden; FormatVersion 1); auch die Rettungskopie trägt sie (`Angaben.Kante`).
- **draw** (früher `zeichnen`) — die aus einem Schwesterprojekt übernommenen Pakete `svg`, `raster`, `geo` (eigenes Go-Modul, das
  das Schwesterprojekt wieder einbindet, ADR-0012); geplant: Weichzeichnen (Schrift liegt in `render`; Verpixeln ist ein Mosaik in `document`, ADR-0022 Nachtrag 1).
- **library** — Autospeichern, Vorschau, Suche, Schlagworte; neue Aufnahmen exklusiv per Hardlink, Rückfall `O_EXCL` (`exklusiv.go`, F-48; „Rückfall“ heißt in Schnapp dreierlei, siehe Glossar); Vorschau nebenläufig verkleinert und erst nach dem Ersetzen geschrieben, die Altdatei weiter voll gelesen (ADR-0020 Nachtrag 11); eigene Reste `.schnapp-tmp-*`, älter als eine Minute, räumen Neuaufbau und Speichern, sonst `ErrTempRest`; `VerwaisteVorschauen` listet nur und meldet `ErrAblageLeer` bei leerer Ablage (ADR-0016 Nachtrag 3).
- **settings** — `settings.ini` lesen (ADR-0018); Warnungen statt Abbruch. `[capture] seam` (tear|wave|zigzag, ADR-0018 Nachtrag 3) wirkt in `cmd/schnapp` bei Aufnahme (`speichere`) und beim Öffnen eines einfachen Bilds (`oeffneImEditor`, `g.Fremd`).
- **selection** — Logik des Overlays ohne Gio (Abbildung, Bereich, Monitor per Klick, Lupe, letzter Bereich).
- **canvas** — Logik der Editor-Leinwand ohne Gio (Zoom zum Zeiger, Einpassen, Treffer, Griffe, Verschieben;
  Zeichenwerkzeuge, Schwärzen und Texteingabe `werkzeuge.go`; Zuschneiden/Stauchen `zuschnitt.go` (Zuschneiden zeigt die Vollansicht mit Rahmen und Griffen, `Bildflaeche`/`VollVersatz`, ADR-0023 Nachtrag 5); Menü-Aktionen ohne Werkzeug `menue.go` (Gestauchtes zurückholen, ADR-0026); Statusleiste `status.go` `StatusZeile`; Speichern im Hintergrund
  `hintergrund.go` (`SpeichernJetzt`, `SchliessenAnfrage`, `SignalAnfrage`, `SpeichernEnde`: die Oberfläche speichert einen Abzug in einer Goroutine, solange ruht das Bearbeiten, ADR-0020 Nachtrag 6; eine synchrone Schließen-Hülle gibt es nicht mehr, Tests fahren die zwei Schritte über die Helfer `schliesse` und `schliesseNachSignal` in `internal/canvas/schliessen_test.go`, F-284; Menü-Navigationstasten stehen als Ausnahme neben der Tastentabelle, ADR-0026 Nachtrag 4), Signalwache `signal.go`; `Wecker` (`wecker.go`) weckt die Gio-Schleife wiederholt, bis sie eine
  Antwort abholt, F-231; ADR-0020 bis ADR-0023).
- **render** — Ansichtsbild + Objekte zu Pixeln ohne GPU (`golang.org/x/image` font, Go-Schriften; ADR-0020);
  Füllen und Glyphen über den eigenen Rasterkern `internal/raster16` (16-Bit-Maske, aus x/image/vector übernommen, bitgleich, ADR-0028) und `einmischer`; `render.Pruefe` zählt die Textzeichen (`ZeichenBudget` 1e6, F-224) und schätzt den
  Aufwand per Trockenlauf (`Budget` 1,25e11, ADR-0020 Nachtrag 3), der abbricht, sobald eine Grenze überschritten ist (F-418); die Kante rechnet ihre Mittellinie einmal je Achse (`document.Kantenlinie`), der Editor lehnt präparierte Dateien beim Öffnen ab
  (`ladeFuerEditor`; `--ignore-limit` überspringt nur diese Prüfung, ADR-0017 Nachtrag 6), Speichern bricht nie ab und
  warnt über dem Budget.
- **wlclip** — Halter der Zwischenablage ohne Fenster (data-control, reines Go ohne libwayland, ADR-0024): Strg+C
  startet `schnapp --internal-keep-clipboard` abgelöst; endet bei neuem Inhalt oder nach 12 Stunden. Er schließt geerbte Deskriptoren ab 4 (`schliesseGeerbte`, ADR-0024 Nachtrag 5); mit `SCHNAPP_DEBUG` meldet er deren Zahl (im virtuellen KWin: 3), seine Fehlerausgabe geht dafür nie in eine Pipe (F-325).
- **portal** — Desktop-Portal über D-Bus ohne Gio: Dateidialog `FileChooser.SaveFile` (Abo vor dem Aufruf, Attrappe im
  Test), `file://`-URI zu Pfad (ADR-0025).
- **export** — Export aus dem Editor: Format aus Endung oder Filter, PNG/JPG aus `render`, SVG aus `fileformat`,
  atomares Schreiben (ADR-0025; `SchreibeAtomarPrivat` legt eine neue Datei mit 0600 an, für `--output`, KA-4). Darf neben dem Ziel keine temporäre Datei entstehen, schreibt der **Rückfall** ein vorhandenes reguläres Ziel direkt: das Ziel wird **einmal** mit `O_RDWR|O_NOFOLLOW|O_NONBLOCK` geöffnet, Sichern (höchstens `MaxRueckfallAlt` = 256 MiB), Kürzen, Schreiben und Wiederherstellen laufen über diesen Deskriptor, bei Lease bis zu 20 Wiederholungen; misslingt auch das Wiederherstellen, meldet `ErrZielBeschaedigt` die Datei als beschädigt (ADR-0025 Nachträge 3 und 4, F-322, F-330, F-338). Fehlertexte beginnen mit dem Paketnamen (ADR-0019 Nachträge 2 und 3, F-339, F-362; Hülle `internal/errprefix`), umgestellt in document, fileformat, export, library, rescue, capture, render, canvas, portal, settings (F-379); `saveprogress` hat keine Fehlerrückgabe, seine Leser/Schreiber reichen Fehler des Stroms bewusst unverändert durch (`io.EOF`).
- **ui** — Gio: Vollbild-Overlay zur Auswahl (fertig), Editor, Bibliothek. Das Schließen wartet auf ein laufendes Kopieren; gibt es keinen Halter (Rückfall: das Fenster hält die Kopie), hält `fensterWarnen` es einmal an und die Statusleiste warnt; scheitert das Kopieren, bleibt das Fenster einmal offen, ein Signal schließt immer (ADR-0024 Nachtrag 6, F-295, F-329, F-337). Ein Speicher-Ende mit Fehler oder Hinweis verwirft die wartende Kopier-Anfrage, die jüngste Kopierantwort entscheidet, und eine haltende Meldung (Verwerfen-Warnung, Hinweis zu einer offenen Anfrage, Fenster-Warnung; `meldeHaltend`) wird von keiner späten Antwort verdrängt — Kopieren, Export, Fehler der Zwischenablage gehen über `meldeNebenbei` (ADR-0024 Nachtrag 7, F-352, F-354, F-356). Übernimmt ein anderes Programm die Zwischenablage, meldet der Gio-Patch das über `app.ClipboardCancelled`; der Editor setzt dann `fensterHaeltKopie` zurück, die Warnung beim Schließen entfällt (ADR-0024 Nachtrag 8, F-328; gegen einen echten Compositor noch ungeprüft).
- **rescue** — Rettungskopie ohne Rendern (ADR-0029 mit Nachträgen 1 bis 3, F-144): JSON mit Kennung der Ausgangsdatei (Pfad, Größe, mtime, SHA-256), Angaben als snAP-SVG und Schwärzungen seit dem letzten Speichern, in `$XDG_STATE_HOME/schnapp/rescue/` (0700/0600); der Editor schreibt gebündelt (2 s Ruhe, höchstens 10 s, `internal/ui/rettung.go`), löscht nach gelungenem Speichern und wendet sie beim Öffnen derselben unveränderten Datei an, nur bis zum Flächenbudget der Schwärzungen (4 × Fläche ≤ `document.VerlaufPlatz()`, sonst `ErrZuGross`, F-412); die Kopie trägt nie das Objekt eines laufenden Zugs (`document.ObjekteOhneZug`, F-409); eine nicht anwendbare legt es beiseite (`….alt.json`); Aufräumen nach 30 Tagen beim Start (ADR-0029 Nachtrag 2). Die CI prüft `scripts/mutanten` mit `-trocken` und fährt `go test -race ./internal/...`. Mutationsprobe als Go-Werkzeug: `scripts/mutanten` (F-371), Gruppen u. a. `rescue` (F-385), `fileformat-jpeg` (F-361/F-389), `fileformat-format` (F-393), `ui-zwischenablage` (F-391), `cmd-agenten` und `canvas-zuschnitt` (Kritik Sprint 11); Echtprobe nach SIGKILL im virtuellen KWin: `scripts/probe.sh --innen scripts/drehbuecher/rueckfall-rettung-kill9.sh` (Aufruf im Skriptkopf).
- **internal/saveprogress** — Fortschritt des Speicherns: `Marke` je Schritt (Zeitstempel mit `SCHNAPP_DEBUG`),
  `Puls` aus langen Schleifen (Rastern, PNG-Kodierung, Lesen, Vorschau), `Zuletzt` für die Frist im Editor (F-297, F-298,
  ADR-0020 Nachtrag 9); die Uhr (`SetzeUhr`) und der Haken je Schritt (`SetzeHaken`) sind für Tests austauschbar, `Puls` und `Marke` nebenläufig sicher (Nachtrag 11).
- **internal/raster16** — Rasterkern aus `golang.org/x/image/vector` v0.26.0 (BSD, `LICENSE` im Ordner) mit
  16-Bit-Maske `Maske()`/`Bloecke()`; bitgleich zu vector, geprüft durch `TestMaskeGleichVector` und `TestMaskeGleichVectorZufall` (ADR-0028).
- **internal/probeguard** — `NurVirtuell`: Probe-Builds (Tag `probe`, Schnapp und Fernbedienung) laufen nur im
  virtuellen KWin von `scripts/probe.sh` (F-229); das Paket selbst hat kein Build-Tag.

Gio trägt, mit einem kleinen Patch (Monitorwahl, gebrochene Skalierung, Mauszeiger bei gemischter Skala F-78, Bild-Typ in
der Zwischenablage ADR-0015 Nachtrag 2, Rückmeldung `ClipboardCancelled` Nachtrag 3) —
ADR-0003 Nachtrag 2, ADR-0015.

## Entwicklungsstand

- **Zeichenpakete** (eigenes Modul `github.com/Andiministrator/schnapp/draw`, Ordner `draw/`, ADR-0012):
  `cd draw && go vet ./... && go test ./...`. Ein Schwesterprojekt bindet dieses Modul ein — öffentliche Schnittstellen nur nach
  Abstimmung ändern (wer, steht in `CLAUDE.local.md`).
- **Gepatchtes Gio** `third_party/gio/` (ADR-0015): eigenes Modul, eingebunden per `replace gioui.org => …/third_party/gio`.
  Änderungen dort, dann `third_party/gio-patch.sh erzeugen` (Patch im selben Commit); Prüfung `third_party/gio-patch.sh pruefen`.
- **Wurzelmodul** `github.com/Andiministrator/schnapp` (bindet Gio per `replace gioui.org => ./third_party/gio`, ADR-0015):
  Gio braucht unter Linux die Build-Tags `novulkan,nox11` (keine Vulkan-Header, nur Wayland) und die cgo-Bibliotheken
  `libwayland-dev libegl-dev libxkbcommon-dev`. Die Tags kommen über die Umgebungsvariable, nicht über `go env -w`:
  `export GOFLAGS=-tags=novulkan,nox11`, dann `go vet ./... && go test ./...`, zusätzlich mit Tag `probe`
  `GOFLAGS=-tags=novulkan,nox11,probe go vet ./cmd/... ./internal/ui && GOFLAGS=-tags=novulkan,nox11,probe go test ./cmd/schnapp-fernbedienung ./cmd/schnapp ./internal/probeguard`
  (der Normalbau sieht `cmd/schnapp-fernbedienung` und `cmd/schnapp/probe_test.go` nicht); Programm `go build -trimpath -o schnapp
  ./cmd/schnapp`. Pakete: `i18n` (Sprachwahl, PO-Leser, `i18n.T/TK/N`), `po` (eingebettete `internal/po/*.po`, ADR-0014
  Nachtrag 1), `ui` (Schicht zu Gio; `Starte` hält die Gio-Hauptschleife, `Overlay` je Monitor),
  `cmd/schnapp` (Argumente `--region` [Voreinstellung], `--screen[=NAME]`, `--list-screens`, `--window`, `--repeat`,
  `--no-editor`, `DATEI`, `--ignore-limit DATEI`, `--version`, `--help`; für Skripte und Agenten `--json`,
  `--output=DATEI` (statt Bibliothek, ohne Editor) und `--export=ZIEL DATEI` in `agenten.go`, ADR-0030, `docs/agents.md`; der Editor fängt SIGTERM/SIGINT/SIGHUP bis zur
  Rückkehr ab und speichert im Hintergrund (ADR-0020 Nachträge 2, 6 und 7: Fristen 15 s / 2 min / 10 s in `internal/ui/editor.go`); bleiben Änderungen ungesichert, endet Schnapp mit Fehler, auch nach einer Aufnahme (`ungesichert` in `cmd/schnapp/editor.go`, Nachtrag 8), Strg+C-Proben nur im eigenen Terminal; `aufnehmen.go` nimmt auf und legt in der Bibliothek ab, Pfad auf
  stdout, dann Editor — erst nach der Rückkehr des Aufnahme-Aufrufs, mit dem abgelegten Dokument und dem Dateistand vom Ablegen (ADR-0020 Nachtrag 6, F-275), Exit 3 bei Abbruch; `editor.go` öffnet Dateien und speichert in die Bibliothek;
  `--screen` mit `[capture] screen = all` nimmt alle Monitore als ein Bild auf, ADR-0017 Nachtrag 7; `schnapp DATEI` auch über „Öffnen mit“ im Dateimanager, `%f` und `MimeType` in `data/schnapp.desktop.in`). **Tests, die Claude selbst fahren kann, fährt
  Claude selbst** (Andi, 20.11.2026): `scripts/probe.sh` (virtueller KWin, Build-Tag `probe`, nie ausliefern), auch den
  Editor per Drehbuch (`--drehbuch DATEI --bild BILD`, Fernbedienung `cmd/schnapp-fernbedienung` über
  `org_kde_kwin_fake_input`, Fotos mit Read ansehen; `internal/wlwire` ist der gemeinsame Wayland-Client von `wlclip`
  und Fernbedienung; ein Teilschreiben schickt den Rest nach, statt die Verbindung zu zerreißen, F-207). Nur was der virtuelle KWin nicht kann (Portal-Dialog, Klipper, fremde Programme, echte Monitore)
  startet Andi selbst per `!`-Befehl, gebündelt, Anleitung davor (`knowledge/topics/lehren.md`); vorher mit `go build -race` nach `tmp/` bauen,
  Diagnose mit `SCHNAPP_DEBUG=1` (`[overlay]`/`[editor]`-Zeilen, `knowledge/topics/gio-wayland.md`).
  Oberflächentexte nur als Literal über `i18n.T/TK/N`; der Test verlangt für jeden Text einen fertigen Eintrag in
  `internal/po/de.po`, und `TestVorlage` hält die Vorlage `internal/po/schnapp.pot` aktuell (neu erzeugen mit
  `go test ./internal/i18n -run TestVorlage -update`); Datum und Zoom folgen der Sprache über die Kontexte
  `date format`/`status bar`, die Sprache wählt `[general] language` (ADR-0014 Nachtrag 2). `capture` ist die Aufnahmeschicht (`Quelle`, KWin über D-Bus: ScreenShot3/memfd bevorzugt, sonst
  ScreenShot2/Pipe; Tests mit Attrappe, keine echte Aufnahme in `go test`). `data/schnapp.desktop.in` ist die Vorlage der Desktop-Datei (ADR-0017). `tmp/go.mod` trennt `tmp/` vom
  Wurzelmodul. `library` ist die Bibliothek (ADR-0016: Ablage `<Bilderordner>/schnapp/<JJJJ-MM>/…png`, Index-Cache
  als JSON-Zeilen (Neuaufbau nur aus Metadaten), Suche, Aufräumen über den freedesktop-Papierkorb); Tests nur mit eigener Umgebung unter `t.TempDir()`.

## Marke

Logo, Farben, Schrift und Regeln für Oberfläche und Doku: [`brand/corporate-design.md`](brand/corporate-design.md) —
verbindlich für alles Sichtbare. Logo-Dateien sind erzeugt (Go-Werkzeug `brand/werkzeug/`, `go run . paket|cd|alles`), nie von Hand ändern.

## Glossar & Offene Entscheidungen

Projektspezifische Begriffe: [`docs/glossar.md`](docs/glossar.md)
Offene Produktfragen: [`docs/open-decisions.md`](docs/open-decisions.md)
Anforderungen: [`docs/anforderungen.md`](docs/anforderungen.md)

## Projektstatus

Aktueller Entwicklungsstand: Board (`aboard-sync.json`, nicht versioniert).

## Repos und Remotes

- **GitHub** `github.com/Andiministrator/schnapp` ist das öffentliche Haupt-Repo, Remote `origin` (ADR-0010).
- Ein **privates Zweit-Repo** im Heimnetz ist zusätzliche Push-URL von `origin`. Adresse, Zugang und Eigenheiten
  (ein Fehlversuch genügt, nicht wiederholen) stehen in **`CLAUDE.local.md`** (nicht versioniert).
- Codeberg entfällt (verbietet überwiegend KI-geschriebene Projekte, `knowledge/topics/plattformen-und-ki-regeln.md`).
- Commits nur mit der GitHub-noreply-Adresse (repo-lokal eingestellt), nie mit einer privaten Mailadresse.
- **Interne Angaben** (Herkunft, Hausnetz, Schwesterprojekt, Abstimmungspartner) gehören in `CLAUDE.local.md`, nie in
  versionierte Dateien (F-1).
- **`gh` ist im Normalfall nicht angemeldet** (F-376; Rat der framework-Sitzung vom 22.11.2026): Auf diesem Rechner
  laufen alle Sitzungen als derselbe Benutzer und könnten einen Token lesen; eine Sperre per Hook hülfe nur gegen
  Versehen. Für einen konkreten PR meldet Andi sich selbst an (`gh auth login` im eigenen Terminal) und danach wieder ab
  (`gh auth logout`). Lesendes `gh` für öffentliche Repos geht ohne Anmeldung (mit Ratenlimit). Ist `gh` angemeldet:
  lesen ohne Rückfrage, **jede schreibende Aktion** (PR, Issue, Kommentar, Review-Antwort, Release, Fork) **nur nach
  Andis Freigabe**, Text vorher zeigen. Sub-Agents rufen `gh` nie auf. Eine echte Grenze (befristeter Token per
  KI-Widget, verwalteter Hook) baut framework für den 13 Pro. Gio-Fork `Andiministrator/gio`, PRs gioui/gio #204 und
  #205 (Karte #157); Push zum Fork über den SSH-Alias `github` (geht ohne `gh`).

## Commit-Konvention

Conventional Commits mit deutschen Beschreibungen: `<typ>(<bereich>): <beschreibung>`

Typen: `feat` · `fix` · `docs` · `refactor` · `chore` · `style` · `test` · `perf` — Details in ADR-0005.

**`Co-Authored-By` nennt, wer die Arbeit gemacht hat — nicht nur, wer committet.** Sonst ist der Owner-/Modell-Abgleich (ADR-0043 Check 1) aus dem Repo nicht nachprüfbar. Vier Fälle, und der zweite ist der, den man übersieht:

| Wer hat gearbeitet oder geprüft? | Trailer-Zeilen |
|---|---|
| Sub-Agent, ein **anderes** Modell committet | 2 — ausführendes bzw. prüfendes Modell zuerst, Orchestrator zuletzt |
| Sub-Agent und Orchestrator sind **dasselbe** Modell | 1 — zwei identische Zeilen belegen nichts; die Rollen gehören in den Fließtext |
| Mehrere parallele Kritiker, mehrere Nacharbeits-Commits | pro Commit nur das Modell, **dessen** Befunde er behebt, plus Orchestrator |
| Nur der Orchestrator | 1 |

Als „Arbeit" zählt auch das reine **Prüfen**: Ein Kritiker- oder Verify-Lauf ist Arbeit am Commit, und die Auflage „anderes Modell" ist ohne Trailer nicht nachprüfbar.

⚠️ **Kontext-Varianten desselben Basismodells sind dasselbe Modell** — „Opus 5" und „Opus 5 (1M context)" ergeben **eine** Zeile, nicht zwei; die Kontextgröße ist eine Eigenschaft der Sitzung, kein anderes Modell. Und für den Verify-Lauf heißt das: **eine größere Kontextgröße erfüllt die Auflage „anderes Modell" nicht** — dafür wechselt man die Familie, nicht die Variante.

## Versionierung

SemVer: `MAJOR.MINOR.PATCH`, Start bei `v0.1.0`. Tags mit `v`-Präfix. Details in ADR-0006.

Release-gebundene Arbeit (mündet in `v*`-Tag / ausgeliefertes Artefakt, inkl. Hotfixes) läuft in einer `kind:sprint`-Swimlane; **kein `git tag v*` ohne vorher bestandenen `/sprint-abschluss`** (inkl. Kritiker-Lauf). Backlog/`kind:lane` = Ideen-Parkplatz. Details in ADR-0062.

**Arbeits-/Planungseinheit ist der Sprint — keine „Phasen"-Prozessebene.** „Fassung 1/2/3" in den Anforderungen ist ein Themenbündel, kein Prozessbegriff; sie wird in Sprints abgearbeitet. (Referenz: ADR-0062 im aBoard-Projekt.)

## Autonomie & Arbeitsweise

Selbstständig arbeiten. Rückfragen nur bei den 3 K's:
- **Kritische Entscheidungen** — nicht durch ADR abgedeckt, schwer reversibel
- **Kritische Auffälligkeiten** — Sicherheit, Datenverlust, unerwarteter Zustand
- **Klärungsbedarf** — Anforderung unklar, Raten = Halluzinationsrisiko

Immer ohne Rückfrage: `git commit`, `git push`, Web-Recherchen.
Nur auf ausdrückliche Ansage: `git tag v*` — ein Release-Tag ist ein Auslieferungsakt, kein gewöhnlicher git-Vorgang. Nach einem bestandenen Sprint-Abschluss **vorschlagen**, nicht setzen.
Immer mit Rückfrage: `git push --force`, Dateien/Branches löschen, Aktionen außerhalb des Projektverzeichnisses.
**Vor jedem Push** den Diff auf Kunden-/Firmenbezug, Pfade, Mailadressen und Hausnetz-Angaben prüfen — das Repo ist
öffentlich.

## QA & Qualitätssicherung

- Nach jedem Schritt: [`docs/qa/schritt-qa.md`](docs/qa/schritt-qa.md)
- Nach jedem Sprint: [`docs/qa/sprint-abschluss.md`](docs/qa/sprint-abschluss.md)
- Finding-Backlog: [`docs/findings.md`](docs/findings.md)
Schritt-QA: nur P0 blockiert den nächsten Schritt. Sprint-Release: blockierend sind alle P0, alle [S] (jede Priorität; enger Ausnahmepfad für bewusste Design-Entscheidungen P2/P3, siehe dort), [K] P0/P1 und alle [B]. Maßgeblich: `~/.claude/commands/sprint-abschluss.md` §Sprint-Release-Check.

## Wissenssammlung & Recherche

Spurensicherung-System: [`knowledge/spurensicherung.md`](knowledge/spurensicherung.md)

## Entscheidungen (ADR)

Unter `docs/adr/`, Schema `NNNN-kurzer-titel.md`. Erst ADR, dann umsetzen.
Status: `Vorgeschlagen` · `Akzeptiert` · `Abgelehnt` · `Überholt`

**Die nächste Nummer im Verzeichnis nachsehen, nicht im Kartentext** (F-334):

```bash
ls docs/adr/[0-9]*.md | sed 's|.*/\([0-9]*\)-.*|\1|' | sort -n | tail -1
```

Eine Nummer, die in einer Backlog-Karte oder einem Entwurf steht, ist **keine
Reservierung** — sie sagt nur, was jemand einmal vorhatte. Sie wird zum
Planungszeitpunkt notiert und zum Umsetzungszeitpunkt gebraucht, und dazwischen
verbrauchen andere Karten Nummern.

## Was zu vermeiden ist

- **Nichts aus dem Schwesterprojekt außer `svg`, `raster`, `geo`** — keine fremden Illustrationen, keine Kunden-
  oder Firmenbezüge (ADR-0012, `CLAUDE.local.md`).
- **Keine echten Screenshots von Andis Bildschirm** als Testdaten oder Doku-Bilder im Repo — sie zeigen Mails, Pfade,
  Namen. Testbilder synthetisch erzeugen oder bewusst ausgewählte, geprüfte Bilder nehmen.
- **Keine Pfade, Benutzer- oder Rechnernamen** in Code, Tests, Metadaten, Golden-Dateien (`/home/<name>` & Co.).
- **„Schnapp“ in Pfaden oder eigenen Dateinamen** — dort immer klein `schnapp` (`~/Bilder/schnapp/`, `schnapp.desktop`);
  im Fließtext und in der Oberfläche „Schnapp“ (Andi, 19.11.2026, ADR-0016).
- **X11-Wege** (xdotool, XGetImage, `import`) — Schnapp ist nur Wayland (ADR-0007).
- **Schwärzen als Vektorebene über dem Original** — dann steht das Geschwärzte im eingebetteten Original im Klartext.
- **Keine Python-Werkzeuge im Repo** — Python ist seit 22.11.2026 ganz aus dem Repo (Andi, 19.11.2026; Karte #23): Logos und
  Grafiken mit Schnapp selbst oder per Go (`brand/werkzeug/`). Wegwerf-Einzeiler während der Arbeit bleiben erlaubt,
  landen aber nie im Repo.
- **`cgo`-Abhängigkeiten ohne Not** — erschwert Flatpak und Kreuzbau; vorher ADR (für Gio samt Patch: ADR-0003 Nachtrag 2).

## Board-Felder (aBoard)

> Typ A lokal, migriert (Op-Inbox) seit 19.11.2026, Ordner verbunden. aBoard nur per `update-aboard.sh` aktualisieren.
> Langfassung: `~/.aboard-framework/framework/claude-md-board-felder.md`.

Board-Mutation = Op über `node ~/.aboard-framework/tools/aboard-op.mjs <op-typ> --<schlüssel> <wert> --actor "<wer>" --dir .` — `aboard-sync.json` ist projektor-geschrieben und read-only.

### Board-Felder und Karten-Workflow — Kurzregeln

*[Typ A/A-remote]* **Langfassung:** `~/.aboard-framework/framework/claude-md-board-felder.md` — lesen, sobald eine Regel hier nicht reicht: Feldtabellen, Sonderfälle der Meta-Karten, Verweis-Typen, Aktivitäts-Log, Tastatur-Shortcuts, Einrichtung (aBoard holen, Ordner verbinden), Pre-Card-Check-Hook.

- **Spalten:** Backlog (`col-backlog`, Parkplatz) → Ready (`col-ready`, Commitment) → In Arbeit (`col-in-arbeit`) → Erledigt (`col-erledigt`). Claude arbeitet **nur Ready-Karten** ab, außer der Mensch beauftragt eine Karte ausdrücklich.
- **Ready + eigener Owner = Arbeitsauftrag:** bei „los“/„mach“ sofort beginnen. Ablauf: übernehmen → In Arbeit → umsetzen → `qa: true` → Erledigt.
- **Vor jedem Karten-Start:** (1) **Modell-Match** über `owner`, frisch gelesen (`Claude` → jedes Modell, `Claude Opus/Sonnet/Haiku` → nur dieses, Username → Aufgabe des Menschen, leer → fragen); passt es nicht, übernimmt ein Sub-Agent mit dem passenden Modell. (2) **Konfliktprüfung** bei parallelen Karten (gleiche Dateien, Module, Verweise → nacheinander). (3) Nennt die Karte eine F-Nummer: steht sie nur noch im Archiv, gegen den Code prüfen und ohne inhaltliche Änderung schließen. Dann `card.claim` (siehe unten).
- **Flags:** `critical` (sprint-kritisch), `blocked` (wartet extern — plus Kommentar mit Grund), `qa` (geprüft). Eine neue Karte beginnt mit allen drei `false`.
- **Schätzung:** jede neue Karte sofort; beim Sitzungsbeginn Claude-Karten ohne Schätzung oder mit leerem `agentContext` **vor** der Arbeit ergänzen. `effort` ist eine **Zahl** in Stunden (`aboard-op` lehnt `"3"`/`"S"` ab). `complexity` nur `""`, `einfach`, `mittel`, `hoch` — bei anderem Text **warnt** `aboard-op` nur und schreibt trotzdem. Regelfall einfach → Haiku, mittel → Sonnet, hoch → Opus; ein gesetzter `owner` hat Vorrang.
- **`owner`** ist verantwortlich und entscheidet, wer bearbeitet; **`assignees`** sind weitere Beteiligte und lösen keinen Sub-Agent aus.
- **`agentContext`** bei Claude-Karten befüllen (bei Anlage, spätestens vor Delegation): `## Relevante Dateien` (Pfad:Zeilen), `## Hinweise / Gotchas`, `## Verwandte ADRs / Findings`, verwandte Karten mit Kurzfassung. Keine Dateiinhalte, keine Aufgabe.
- **Datum:** `dueDate`/`startDate` als `YYYY-MM-DD`, Uhrzeit nur in `dueTime`/`startTime` (`HH:MM`). Zeitkritische Karten bekommen ein `dueDate`.
- **Archiv:** `archived` verbirgt Karten oder ganze Swimlanes. Die Karten einer archivierten Swimlane stehen weiter im JSON — **nie Arbeit in eine archivierte Swimlane legen**.
- **Swimlanes:** `kind` = `sprint` (Meta-Karten, `/sprint-abschluss`), `lane` (ohne Sprint-Mechanik), `backlog` (`__backlog__`), `service` (Mail-Eingang, nur Server-Board).
- **Meta-Karten** (📋 Sprint-Start, 🏁 Sprint-Abschluss, `kind: 'sprint-meta'`): legt nur der Browser an, nie ein Werkzeug. Nicht löschen, nicht archivieren, kein Drag-and-Drop; sonst nur Beschreibung und Kommentare ändern. Start-Karte in Erledigt = Sprint läuft. Auf Ansage „Starte Sprint N“: Start-Karte per `card.move` nach `col-erledigt` und `qa: true`. Beim `/sprint-abschluss` Recap in die 🏁-Karte, dann `qa: true` und Erledigt.
- **Verweise** (`links`): `blocks`↔`blocked-by`, `depends-on`↔`required-for`, `related`↔`related`, immer beidseitig — nur über `card.listAdd`/`card.listRemove` ändern, die pflegen die Rückseite (`card.field`/`card.upsert` auf `links` nicht, dort warnt `aboard-op` nur). Ready-Karte mit offenem `blocked-by` überspringen oder fragen; vor dem Verschieben offene Blocker nennen.
- **[nur Typ A lokal]** aBoard nur per `update-aboard.sh` holen und aktualisieren, nie per `cp`. Werkzeuge aus `~/.aboard-framework/`, **nie aus einem aBoard-Arbeitsbaum**; dort nichts bauen (er trägt fremde Zwischenstände).

### Board ändern — Kurzregeln

**Langfassung:** `~/.aboard-framework/framework/claude-md-board-konvention.md` (fehlt sie: `<aBoard>/framework/install.sh tools`). Lesen, sobald eine Regel hier nicht reicht: Op-Typen-Liste, Fehlerwege der Meldungen, Kompaktierung, `ts`/`seq`, awaitingUser, A-remote-Einzelheiten, Export für Fremdwerkzeuge.

- **[nur Typ A lokal] Migriert** (`aboard-sync.inbox/` und `aboard-sync.base.json` liegen neben `aboard-sync.json`): Ändern nur über Ops, **nie** `aboard-sync.json` schreiben — die schreibt der Projektor. Nicht migriert: Direktschreiben nur unter dem F-283-Schutzprotokoll (Langfassung).
- **Werkzeug:** `node ~/.aboard-framework/tools/aboard-op.mjs <op-typ> --<schlüssel> <wert> … --actor "<Akteur>" --dir .` — prüft die Payload, bevor es schreibt. `--help` listet Op-Typen und Schlüssel, `--dry-run` zeigt die Op. Ops **nie von Hand als JSON** bauen (`ts`/`seq` gehen sonst schief).
- **Exit-Codes:** `0` geschrieben · `1` Laufzeitfehler · `2` Aufruf-/Payload-Fehler, nichts geschrieben · `3` Bedingung hält nicht (z. B. Karte hält ein anderer) · `4` nicht bestätigt · `5` A-remote: im Ausgang, Server weg — **nicht neu schreiben**, später `push`.
- **Karten über die `id` (UUID) ansprechen**, nie über die Kartennummer `displayId` (eine Zahl; ein Vergleich gegen `'116'` trifft nichts). Eine neue Kartennummer nie vorhersagen.
- **Payload-Namen weichen von den Feldnamen ab:** `card.move` nimmt `cardId`, `toSprintId`, `toColId`; `card.upsert` nimmt `card` (verschachtelt), `sprintId`, `columnId`. Verbindlich ist `aboard-op.mjs --help`, nie die Projektion. Ein Typ, den `--help` nicht nennt, ist ungültig (es gibt kein `card.add`).
- **Feldnamen nicht erfinden:** `card.field --field desc` statt `description` legt still ein neues Feld an, `aboard-op` warnt nicht. Die Spalte zeigt `columnId`; das Feld `status` wird nicht gepflegt.
- **`card.upsert` ersetzt die ganze Karte.** Bestandskarten nur vollständig upserten, ein einzelnes Feld mit `card.field`.
- **Nach jeder Mutation beide Proben:** (1) die Meldung des Projektors lesen — headless `node ~/.aboard-framework/tools/aboard-projektor.mjs --dir .`, auch wenn ein Browser-Tab offen ist; ungültige Ops landen in `aboard-sync.inbox/.rejected/` (Beweismittel, keine Vorlage). (2) die Karte zurücklesen und **das geschriebene Feld** gegen das Gewollte halten, nicht nur Spalte und Nummer — bei eigenem Op-Werkzeug auch den `ts`. Liegt `aboard-sync.inbox/.projektor-warnungen.log`, ist eine Meldung offen: lesen, dann `--warnungen-quittieren`.
- **[nur Typ A lokal] Projektor Exit 3 ist kein Fehler:** base und Projektion passen nicht zusammen (etwa nach einem Rückspielen), er schreibt nichts. Auflösen in genau einer Richtung, und das entscheidet der Mensch: `--rebase-from-projection` (die Projektion gilt) oder `--force` (die base gilt).
- **[nur Typ A lokal] Kompaktierung** ab 50 gültigen Ops ist der Aufräum-Lauf, kein Defekt (viele gelöschte Inbox-Dateien im `git status`). Die drei Board-Dateien immer gemeinsam stagen, soweit versioniert: `git add aboard-sync.json aboard-sync.base.json aboard-sync.inbox/` (sind base und Inbox gitignoriert, bricht das ab — dann nur `aboard-sync.json`).
- **Kommentar:** `comment.add` über `--payload @datei.json` mit `{"cardId":"<id>","comment":{"id":"<uuid4>","author":"<Akteur>","text":"…","created":"YYYY-MM-DD"}}`. Langen Text nie über die Shell quoten.
- **Übernahme:** vor dem Beginn `card.claim --cardId <id> --session <ABOARD_SESSION>`. Exit 3: ein anderer hält sie, nicht beginnen. Exit 4: einmal mit derselben Sitzung wiederholen, dann nicht beginnen.
- **awaitingUser** nur per `board.awaiting` (Person, Sitzung, Wert), nie per `board.field`.
- **[nur Typ A-remote]** Kein lokales `aboard-sync.json`. Lesen aus `aboard-remote.mirror.json` (nur lesen), schreiben nur über `aboard-op.mjs … --dir .`. Einzelne Felder per `card.field`, nie per `card.upsert`. Freitext (`title`, `description`, `agentContext`) erst nach `node ~/.aboard-framework/tools/aboard-remote.mjs karte <id> --dir .`. `card.claim` geht direkt an den Server. Das Token steht nie im Projekt, in einer Karte oder in einem Commit.

### Sub-Agents (Board-Mutation)

Sub-Agents haben **kein Memory** — jeder Prompt ist selbst-enthaltend. **Wörtlich hinein gehören:** (1) der Block „⛔ ABSOLUTE AUFLAGE — lokale Werkzeuge mit Geheimnis- oder Personenbezug“ samt Meldepflicht aus `~/.claude/commands/sprint-abschluss.md` Phase 2 (kopieren, nicht nachbauen; ein Datei-Verweis reicht nicht); (2) „Nichts nach /tmp“ mit dem Ort `tmp/<…>/` im Projekt; (3) keine gemeinsam beschriebenen Dateien, keine git-Eingriffe über die eigene Datei hinaus, kein Build — je mit erlaubtem Weg; (4) neue Findings nur als Text mit Kartenvorschlag; (5) kein `rm` außerhalb von `tmp/<sprint>/<lauf>/`, auch dort nur mit ausgeschriebenem Pfad; (6) Testdaten mit Shell-Befehlen nur per Write-Werkzeug oder Python, nie per Heredoc. Der Hook `agent-prompt-check.py` weist einen Start ohne (1) oder (2) zurück. Ops anhängen dürfen Sub-Agents, `aboard-sync.json` schreiben nicht; bei nicht migriertem Board ändert nur der Hauptchat.

### Jedes Finding hat eine Karte

*[Typ A/A-remote] — mit Finding-Register (`docs/findings.md`)*

Jedes offene Finding (jeder Typ, jede Priorität, auch geparkt) steht mit seiner F-Nummer in Titel oder Beschreibung mindestens einer offenen, nicht archivierten **Arbeitskarte**. Einstiegs-, Übergabe- und Sprint-Meta-Karten zählen nicht, Sammelkarten schon. Wer ein Finding anlegt, legt im selben Zug die Karte an; wer eine Karte schließt, prüft ihre Findings. Die Karte ist ein Zeiger, der Befund steht im Register. Sub-Agents liefern nur den Vorschlag. Prüfung (nur lesend): `node ~/.aboard-framework/tools/findings-karten-abgleich.mjs --dir .` — Exit 0 verbunden, 1 Befund, 2 nicht auswertbar.

### Fragen und Befunde gesammelt am Ende

*Gilt für jeden Projekttyp.*

Zwischentexte sind kurze Statuszeilen, denn der Mensch sieht beim Anhalten nur den letzten Block. Fragen, Entscheidungen, Befunde und Antworten auf Zwischenfragen stehen gesammelt im Schlussblock, auch wenn sie mittendrin schon fielen. Eine Rückfrage zu einer gleich anstehenden Freigabe kommt sofort **und** am Ende. Der Stop-Hook `schluss-fragen-stop.py` verlangt einen Nachtrag, wenn eine Frage aus einem Zwischentext im Schluss fehlt (abschalten: `ABOARD_SCHLUSS_FRAGEN=0`).
