# Besuchte Quellen

Log **aller** URLs, die für dieses Projekt besucht wurden — auch Fehlschläge.
**Vor jeder Web-Recherche hier prüfen**, ob die URL schon besucht wurde und wann.
Nur erneut abrufen, wenn der letzte Besuch veraltet ist (siehe Aktualitäts-Sichtung)
oder sich die Frage geändert hat.

**Eine Zeile pro URL.** Wird eine URL erneut besucht, wird **„Letzter Besuch" aktualisiert**
(und ggf. Status/Inhalt) — keine zweite Zeile. „Erstbesuch" bleibt stehen.
(Format übernommen aus einem anderen Projekt des Maintainers, Festlegung Andi 2026-09-25, für Schnapp 2026-11-19.)

**Format:** `| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |`

- **Datum:** ISO `YYYY-MM-DD`
- **Status:**
  - `live` — erreichbar, Inhalt gelesen
  - `indirect` — Original nicht erreichbar oder nicht geöffnet, Inhalt über Sekundärquelle (Suchtreffer-Auszug, Archiv)
  - `broken` — 404, DNS-Fehler, dauerhaft nicht erreichbar
  - `blocked` — Login, 403, JS-Challenge o. ä. verhindern anonymen Zugriff
  - `redirect` — leitet auf eine andere URL/Domain weiter (Ziel im Feld „Inhalt")
  - `leer` — erreichbar, aber für die Frage ohne Ertrag
- **Weg:** `WebFetch` · `WebSearch` (Treffer nur aus der Suchergebnisliste, Seite nicht geöffnet) · `Browser` · `curl` · `Download`
- **Zieldatei:** Datei in `knowledge/` oder `docs/`, in die der Inhalt eingeflossen ist, sonst `—`

Suchanfragen selbst (ohne geöffnete Seite) stehen im Abschnitt „Suchanfragen" am Ende, mit Datum.

**Zählstände werden gemessen, nicht fortgeschrieben:**

```bash
awk -F'|' '/^\| *http/ {gsub(/ /,"",$5); print $5}' knowledge/sources-visited.md | sort | uniq -c
```

## Aktualitäts-Sichtung

Halbjährlich: Zeilen, deren „Letzter Besuch" älter als 12 Monate ist, auf Veränderung prüfen
(KWin-Schnittstellen, Plattformregeln und Lizenzen ändern sich). Ergebnis hier eintragen.

- Letzte Sichtung: — (Log angelegt 2026-11-19) · Nächste: ~2027-05

## Methoden & Zugangshürden

Was beim Abrufen auffiel (Sperren, Umwege, verlässliche Primärquellen).

- **KWin-Quelltext** (2026-11-19): `invent.kde.org/<gruppe>/<repo>/-/raw/master/<pfad>` per WebFetch lesbar.
  WebFetch gibt LGPL-Dateien nur sinngemäß wieder, nicht wörtlich → für Wortlaut per `curl` holen.
  Installierte `.desktop`-Dateien unter `/usr/share/applications/` sind eine verlässliche Primärquelle (Spectacle).
- **Flathub-API** (2026-11-19): `flathub.org/api/v2/search?q=…` per GET → 405; Suche erwartet vermutlich POST.

---

## Plattformen und Regeln zu KI-Code

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://gioui.org/doc/contribute | 2026-11-21 | 2026-11-21 | 200 | web-fetch | Mailingliste gio-patches (Hauptweg), PRs am GitHub-Spiegel als Ersatz, Sign-off Pflicht, Präfix PATCH gio, kein SourceHut-Konto nötig, nichts zu KI-Code | topics/plattformen-und-ki-regeln.md |
| https://codeberg.org/codeberg/org/src/branch/main/TermsOfUse.md | 2026-11-19 | 2026-11-19 | live | WebFetch | § 2 Abs. 1 Nr. 7: überwiegend KI-geschriebene Projekte verboten | topics/plattformen-und-ki-regeln.md |
| https://linuxiac.com/codeberg-bans-projects-mostly-written-by-generative-ai/ | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Codeberg-Verbot Juli 2026, Abstimmung | topics/plattformen-und-ki-regeln.md |
| https://www.gamingonlinux.com/2026/10/flathub-walked-back-their-generative-ai-ban-replaced-with-a-disclosure-policy/ | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Flathub: Offenlegung statt Verbot, Manifest/Einreichung ohne KI | topics/plattformen-und-ki-regeln.md |
| https://linuxiac.com/flathub-reverses-ai-generated-app-ban-now-requires-disclosure/ | 2026-11-19 | 2026-11-19 | indirect | WebSearch | dito | topics/plattformen-und-ki-regeln.md |

## KWin und Wayland

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://invent.kde.org/libraries/plasma-wayland-protocols/-/raw/master/src/protocols/fake-input.xml | 2026-11-20 | 2026-11-20 | live | curl | org_kde_kwin_fake_input v6, Requests und Opcodes | topics/lehren.md |
| https://invent.kde.org/plasma/kwin/-/raw/master/src/backends/fakeinput/fakeinputbackend.cpp | 2026-11-20 | 2026-11-20 | live | curl (Pfad über GitLab-API tree) | authenticate setzt authenticated ohne Prüfung („TODO: make secure“) | topics/lehren.md |
| https://invent.kde.org/plasma/plasma-workspace/-/raw/master/klipper/klipper.kcfg | 2026-11-20 | 2026-11-20 | live | curl | `SaveImages` „Save images“, Voreinstellung false | topics/gio-wayland.md |
| https://invent.kde.org/plasma/plasma-workspace/-/raw/master/klipper/configdialog.cpp | 2026-11-20 | 2026-11-20 | live | curl | Bezeichnung „Image data copied explicitly“ unter „Include in history“ | topics/gio-wayland.md |
| https://invent.kde.org/plasma/plasma-workspace/-/raw/master/klipper/historymodel.cpp | 2026-11-20 | 2026-11-20 | live | curl | Bild ohne Text nur mit SaveImages oder `x-kde-force-image-copy` in den Verlauf | topics/gio-wayland.md |
| https://invent.kde.org/graphics/spectacle/-/raw/master/src/ExportManager.cpp | 2026-11-20 | 2026-11-20 | live | curl | Spectacle setzt `x-kde-force-image-copy` (leer) | topics/gio-wayland.md |
| https://invent.kde.org/plasma/kwin/-/raw/master/src/plugins/screenshot/org.kde.KWin.ScreenShot2.xml | 2026-11-19 | 2026-11-19 | live | WebFetch | Methoden, Optionen, Rückgabe, Pipe | topics/kwin-screenshot.md |
| https://invent.kde.org/plasma/kwin/-/raw/master/src/plugins/screenshot/screenshotdbusinterface2.cpp | 2026-11-19 | 2026-11-19 | live | WebFetch, curl | Pipe, raw-Format; Prüfung entfernt (nur noch determineCallerPid für hide-caller-windows) | topics/kwin-screenshot.md |
| https://invent.kde.org/plasma/kwin/-/raw/Plasma/6.7/src/plugins/screenshot/screenshotdbusinterface2.cpp | 2026-11-19 | 2026-11-19 | live | WebFetch (Sonnet-Agent #42) | checkPermissions ruft fetchRestrictedDBusInterfacesFromPid (F-76) | topics/kwin-screenshot.md |
| https://invent.kde.org/plasma/kwin/-/raw/Plasma/6.7/src/utils/serviceutils.h | 2026-11-19 | 2026-11-19 | live | WebFetch (Sonnet-Agent #42, nur Zusammenfassung) | exec() per QProcess::splitCommand zerlegt, kanonischer Pfad verglichen (F-76) | topics/kwin-screenshot.md |
| https://invent.kde.org/plasma/kwin/-/raw/Plasma/6.7/src/utils/serviceutils.cpp | 2026-11-19 | 2026-11-19 | 404 | WebFetch (Sonnet-Agent #42) | gibt es nicht (inline im Header), auch nicht in master | — |
| https://invent.kde.org/api/v4/projects/plasma%2Fkwin/repository/commits?path=src/plugins/screenshot/screenshotdbusinterface2.cpp | 2026-11-19 | 2026-11-19 | live | curl | Commit-Geschichte: 0ea31ae7 „Drop X-KDE-DBUS-Restricted-Interfaces“, 5be04288 „memfd“ | topics/kwin-screenshot.md |
| https://invent.kde.org/api/v4/projects/plasma%2Fkwin/repository/commits/0ea31ae7 (auch /diff, /refs?type=tag) | 2026-11-19 | 2026-11-19 | live | curl | Prüfung entfernt 18.06.2026, Tags v6.7.90–v6.8.3; alter Code checkPermissions, KWIN_SCREENSHOT_NO_PERMISSION_CHECKS | sources/kwin-screenshot-berechtigung.md |
| https://invent.kde.org/api/v4/projects/plasma%2Fkwin/repository/commits/5be04288 (auch /diff, /refs?type=tag) | 2026-11-19 | 2026-11-19 | live | curl | ScreenShot3 mit memfd, 17.11.2026, in keinem Tag | sources/kwin-screenshot-berechtigung.md |
| https://invent.kde.org/plasma/kwin/-/raw/master/src/plugins/screenshot/org.kde.KWin.ScreenShot3.xml | 2026-11-19 | 2026-11-19 | live | curl | Methoden wie ScreenShot2, fd h als Ausgabe | topics/kwin-screenshot.md |
| https://invent.kde.org/api/v4/projects/plasma%2Fkwin/search?scope=blobs | 2026-11-19 | 2026-11-19 | blocked | curl | 401, Codesuche nur angemeldet | — |
| https://sourceware.org/git/?p=glibc.git;a=blob_plain;f=intl/dcigettext.c;hb=HEAD | 2026-11-19 | 2026-11-19 | live | curl | guess_category_value: LANGUAGE ignoriert bei Locale C/C.<enc> (nicht POSIX); Liste endet bei C, C.<enc>, POSIX | docs/adr/0014-mehrsprachigkeit.md (Nachtrag 1, F-44) |
| https://invent.kde.org/plasma/spectacle/-/raw/master/desktop/org.kde.spectacle.desktop | 2026-11-19 | 2026-11-19 | broken | curl | 404 — die Datei heißt .desktop.cmake | — |
| https://invent.kde.org/api/v4/projects/plasma%2Fspectacle/repository/tree (path=, desktop, src) | 2026-11-19 | 2026-11-19 | live | curl | Dateilisten zum Auffinden | — |
| https://invent.kde.org/plasma/spectacle/-/raw/master/desktop/org.kde.spectacle.desktop.cmake | 2026-11-19 | 2026-11-19 | live | curl | X-KDE-Shortcuts im Hauptteil und je Desktop-Aktion (Print, Shift+Print, Meta+Print …) | docs/adr/0017-tastenkuerzel.md |
| https://invent.kde.org/plasma/spectacle/-/raw/master/src/ShortcutActions.cpp | 2026-11-19 | 2026-11-19 | live | curl | Kürzel-Namen passen zu den Aktionen der .desktop-Datei | docs/adr/0017-tastenkuerzel.md |
| https://bugs.kde.org/show_bug.cgi?id=435696 | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Exec-Pfad muss absolut sein | topics/kwin-screenshot.md |
| https://mail.kde.org/pipermail/kde-frameworks-devel/2020-June/111948.html | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Einführung X-KDE-DBUS-Restricted-Interfaces | topics/kwin-screenshot.md |
| https://gitlab.freedesktop.org/wayland/wayland-protocols/-/raw/1.45/staging/fractional-scale/fractional-scale-v1.xml | 2026-11-19 | 2026-11-19 | live | curl | Protokoll gebrochene Skalierung (preferred_scale in 1/120) | extern/gio-ausgabe.patch (erzeugter Code) |
| https://gitlab.freedesktop.org/wayland/wayland-protocols/-/raw/1.45/stable/viewporter/viewporter.xml | 2026-11-19 | 2026-11-19 | live | curl | Protokoll Viewport (Zielgröße der Fläche) | extern/gio-ausgabe.patch (erzeugter Code) |

## Name und Lizenz

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://api.github.com/search/repositories?q=schnapp+in:name | 2026-11-19 | 2026-11-19 | live | curl | 129 Treffer, pricerc/schnapp ähnlich | docs/open-decisions.md (E-08) |
| https://aur.archlinux.org/rpc/v5/search/schnapp | 2026-11-19 | 2026-11-19 | live | curl | 0 Treffer | docs/open-decisions.md (E-08) |
| https://flathub.org/api/v2/search?q=schnapp | 2026-11-19 | 2026-11-19 | leer | curl | HTTP 405 (falsche Methode, kein Ergebnis) | — |
| https://api.github.com/repos/Andiministrator/schnapp/commits/main | 2026-11-19 | 2026-11-19 | live | curl | Autor/Committer = noreply-Adresse | docs/findings.md (F-1) |
| https://api.github.com/repos/Andiministrator/schnapp/actions/runs | 2026-11-21 | 2026-11-19 | live | curl | CI-Lauf test: success; 21.11.: Status der CI-Läufe #154–#170 | — |
| https://www.gnu.org/licenses/gpl-3.0.txt | 2026-11-19 | 2026-11-19 | live | curl | GPL-3.0-Text | LICENSE |
| https://api.github.com/repos/Andiministrator/schnapp/commits/<sha>/check-runs | 2026-11-24 | 2026-11-23 | live | curl | CI-Status je Commit: e08a477 rot (wurzel), ab 1cb0d08 grün; 24.11.: 14aee47 grün, d6a182b grün (F-602) | docs/findings.archive.md (F-527) |
| https://api.github.com/repos/Andiministrator/schnapp/actions/jobs/138058366566/logs | 2026-11-23 | 2026-11-23 | 403 | curl | Job-Log nur angemeldet („Must have admin rights“), wie in Lehren vermerkt | — |
| https://github.com/Andiministrator/schnapp/actions/runs/45279053677/job/138058366566 | 2026-11-23 | 2026-11-23 | live | Chrome (nur lesen, Tab zu) | Schritt „Editor-Zeichnen ohne Fenster“: beide Tests SKIP, `gl: no OpenGL implementation could be loaded (libGLESv2.so.3.0, .so.2)` → F-602 | tmp/sprint12/ci/f602-ci-befund.md |
| https://github.com/Andiministrator/schnapp/actions/runs/45170788427/job/137679469084 | 2026-11-23 | 2026-11-23 | live | Chrome (nur lesen) | Job-Log: TestAbbruchWaehrendDesAufrufs rot unter -race | docs/findings.archive.md (F-527) |
| https://www.techsmith.com/snagit/ | 2026-11-23 | 2026-11-23 | live | Chrome (nur lesen, Cookie-Banner nicht bedient) | Seite trägt Camtasia-Marke, Snagit-Symbol nicht sichtbar | — |
| https://en.wikipedia.org/wiki/Snagit | 2026-11-23 | 2026-11-23 | live | Chrome (nur lesen) | Symbole: weißes, kantiges S auf blauer Kachel; Snagit Capture auf rot-oranger Kachel | Board-Karte #217 |
| https://techsmith.com/blog/wp-content/uploads/2025/01/Snagit-Help-2024-EN.pdf | 2026-11-23 | 2026-11-23 | live | WebSearch-Auszug | Markenliste nennt „S Icon“ als TechSmith-Marke | Board-Karte #217 |

## GitHub Actions: Node-24-Fassungen und SHA-Pinning (Sprint 9, Karte #29, F-43/F-309)

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://api.github.com/repos/actions/checkout/releases/latest | 2026-11-21 | 2026-11-21 | live | curl | neueste Fassung v7.0.1 (veröffentlicht 2026-07-20) | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/actions/setup-go/releases/latest | 2026-11-21 | 2026-11-21 | live | curl | neueste Fassung v7.1.0 (veröffentlicht 2026-11-17) | knowledge/sources/github-actions-node24.md |
| https://raw.githubusercontent.com/actions/checkout/v7.0.1/action.yml | 2026-11-21 | 2026-11-21 | live | curl | `runs.using: node24` (Zeile 116) | knowledge/sources/github-actions-node24.md |
| https://raw.githubusercontent.com/actions/setup-go/v7.1.0/action.yml | 2026-11-21 | 2026-11-21 | live | curl | `runs.using: 'node24'` (Zeile 48); Eingabe `cache` Standard `true` | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/actions/checkout/git/ref/tags/v7.0.1 | 2026-11-21 | 2026-11-21 | live | curl | leichtes Tag, zeigt direkt auf Commit 3d3c42e5 | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/actions/setup-go/git/ref/tags/v7.1.0 | 2026-11-21 | 2026-11-21 | live | curl | leichtes Tag, zeigt direkt auf Commit d8cbb804 | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/actions/checkout/releases?per_page=8 | 2026-11-21 | 2026-11-21 | live | curl | Release-Notes v7.0.0, v7.0.1, v6.1.0 (Bruchänderung Fork-PR nur bei pull_request_target/workflow_run) | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/actions/setup-go/releases?per_page=8 | 2026-11-21 | 2026-11-21 | live | curl | Release-Notes v7.0.0 (ESM), v7.1.0, v6.2.0 | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/actions/checkout/releases/tags/v6.0.0 | 2026-11-21 | 2026-11-21 | live | curl | Release-Notes v6.0.0 (Zugangsdaten in separater Datei) | knowledge/sources/github-actions-node24.md |
| https://api.github.com/repos/Andiministrator/schnapp/commits/<sha>/check-runs, …/check-runs/<id>/annotations | 2026-11-21 | 2026-11-21 | live | curl | Annotationen: Lauf #158 eine („Node.js 20 is deprecated“), #159 keine (F-309) | knowledge/topics/lehren.md (Sprint 9) |

## Model Context Protocol (Karte #173, ADR-0030 Nachtrag 2)

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://modelcontextprotocol.io/specification/2026-07-28/changelog | 2026-11-22 | 2026-11-22 | live | WebFetch | Handshake, ping, setLevel entfallen; `resultType`; `-32022` | `knowledge/sources/mcp-2026-07-28.md` |
| https://modelcontextprotocol.io/docs/2026-07-28/learn/versioning | 2026-11-22 | 2026-11-22 | live | WebFetch | aktuelle Version 2026-07-28; Zwei-Epochen-Regel | `knowledge/sources/mcp-2026-07-28.md` |
| https://modelcontextprotocol.io/specification/2026-07-28/basic/lifecycle | 2026-11-22 | 2026-11-22 | live | WebFetch | „Versioning and Compatibility“ | `knowledge/sources/mcp-2026-07-28.md` |
| https://modelcontextprotocol.io/specification/2026-07-28/basic/index | 2026-11-22 | 2026-11-22 | live | WebFetch | `_meta`-Schlüssel je Anfrage, `-32602` | `knowledge/sources/mcp-2026-07-28.md` |
| https://modelcontextprotocol.io/specification/2026-07-28/basic/transports/stdio | 2026-11-22 | 2026-11-22 | live | WebFetch | Zeilen-JSON, stdout nur Protokoll, Ende bei EOF, Probe mit `server/discover` | `knowledge/sources/mcp-2026-07-28.md` |
| https://modelcontextprotocol.io/specification/2026-07-28/server/discover | 2026-11-22 | 2026-11-22 | live | WebFetch | Antwortform `server/discover` | `knowledge/sources/mcp-2026-07-28.md` |
| https://modelcontextprotocol.io/specification/2026-07-28/server/tools | 2026-11-22 | 2026-11-22 | live | WebFetch | Werkzeug- und Ergebnisform, `isError`, Bildinhalt | `knowledge/sources/mcp-2026-07-28.md` |

## OCR (Karte #195, ADR-0035)

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://blogs.kde.org/2025/11/15/this-week-in-plasma-ocr-in-spectacle-and-many-ui-improvements/ | 2026-11-24 | 2026-11-24 | indirect | WebSearch | Spectacle bekommt OCR über Tesseract, Text in die Zwischenablage (Plasma 6.6) | docs/adr/0035-text-erkennen.md |
| https://invent.kde.org/plasma/spectacle/-/merge_requests/462 | 2026-11-24 | 2026-11-24 | indirect | WebSearch | MR „Add OCR functionality to Spectacle“, Tesseract-Engine | docs/adr/0035-text-erkennen.md |
| https://bugzilla.redhat.com/show_bug.cgi?id=2435519 | 2026-11-24 | 2026-11-24 | indirect | WebSearch | Spectacle bindet libtesseract; ohne Entwicklungspaket grau — Bibliotheksbindung ist zerbrechlich | docs/adr/0035-text-erkennen.md |
| https://man.archlinux.org/man/tesseract.1.en | 2026-11-24 | 2026-11-24 | indirect | WebSearch | Aufruf `tesseract FILE OUTPUTBASE`; `stdin`/`-` als Eingabe, `stdout`/`-` als Ausgabe (5.x) | docs/adr/0035-text-erkennen.md |
| https://gitlab.gnome.org/World/OpenPaperwork/paperwork/issues/735 | 2026-11-24 | 2026-11-24 | indirect | WebSearch | Flatpak-Bau mit eigener Tesseract-Kopie: Sprachdaten nur im Sandbox-tessdata | docs/adr/0035-text-erkennen.md |
| https://flathub.org/apps/xyz.rescribe.rescribe | 2026-11-24 | 2026-11-24 | indirect | WebSearch | OCR-Programm auf Flathub mit gebündeltem Tesseract | docs/adr/0035-text-erkennen.md |

## Suchanfragen

| Datum | Anfrage | Ergebnis |
|---|---|---|
| 2026-11-19 | Codeberg terms of use AI generated code projects not allowed 2026 | Verbot bestätigt |
| 2026-11-19 | Flathub policy AI generated applications submission requirements LLM | Offenlegungspflicht seit Okt. 2026 |
| 2026-11-19 | KWin X-KDE-DBUS-Restricted-Interfaces desktop file Exec path … | absoluter Exec-Pfad nötig |
| 2026-11-19 | "schnapp" screenshot tool linux github OR flathub OR AUR | kein gleichnamiges Werkzeug |
| 2026-11-21 | Gio gioui contributing patches mailing list sourcehut send-email | gio-patches auf lists.sr.ht, git send-email, --signoff Pflicht, GitHub-PR als Ersatz |
| 2026-11-21 | gioui.org latest release version 2026 | v0.10.3 aktuell (Newsletter Sept. 2026) |
| 2026-11-21 | gioui contribute AI generated code policy LLM patches Elias Naur | keine Gio-eigene KI-Regel gefunden |
| 2026-11-22 | Model Context Protocol specification latest protocol version 2026 stdio lifecycle initialize (nur modelcontextprotocol.io) | aktuelle Revision 2026-07-28 ohne initialize-Handshake |
| 2026-11-23 | Snagit logo icon design TechSmith 2024 | kein Designbeleg; Markenliste mit „S Icon“ in der Snagit-Hilfe 2024 |
| 2026-11-24 | Spectacle KDE text extraction OCR tesseract Plasma 6 | Spectacle OCR über Tesseract seit Plasma 6.6, Paketierungsfehler bei libtesseract |
| 2026-11-24 | tesseract command line stdin stdout | stdin/stdout bzw. „-“ als Ein- und Ausgabe (5.x) |
| 2026-11-24 | Flathub flatpak bundle tesseract OCR module traineddata extension | keine gemeinsame Flathub-Erweiterung; Apps bündeln Tesseract und Sprachdaten selbst |
