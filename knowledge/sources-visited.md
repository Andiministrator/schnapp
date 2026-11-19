# Besuchte Quellen

Log **aller** URLs, die für dieses Projekt besucht wurden — auch Fehlschläge.
**Vor jeder Web-Recherche hier prüfen**, ob die URL schon besucht wurde und wann.
Nur erneut abrufen, wenn der letzte Besuch veraltet ist (siehe Aktualitäts-Sichtung)
oder sich die Frage geändert hat.

**Eine Zeile pro URL.** Wird eine URL erneut besucht, wird **„Letzter Besuch" aktualisiert**
(und ggf. Status/Inhalt) — keine zweite Zeile. „Erstbesuch" bleibt stehen.
(Format übernommen aus dem Projekt easywk, Festlegung Andi 2026-09-25, für Schnapp 2026-11-19.)

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
| https://codeberg.org/codeberg/org/src/branch/main/TermsOfUse.md | 2026-11-19 | 2026-11-19 | live | WebFetch | § 2 Abs. 1 Nr. 7: überwiegend KI-geschriebene Projekte verboten | topics/plattformen-und-ki-regeln.md |
| https://linuxiac.com/codeberg-bans-projects-mostly-written-by-generative-ai/ | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Codeberg-Verbot Juli 2026, Abstimmung | topics/plattformen-und-ki-regeln.md |
| https://www.gamingonlinux.com/2026/10/flathub-walked-back-their-generative-ai-ban-replaced-with-a-disclosure-policy/ | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Flathub: Offenlegung statt Verbot, Manifest/Einreichung ohne KI | topics/plattformen-und-ki-regeln.md |
| https://linuxiac.com/flathub-reverses-ai-generated-app-ban-now-requires-disclosure/ | 2026-11-19 | 2026-11-19 | indirect | WebSearch | dito | topics/plattformen-und-ki-regeln.md |

## KWin und Wayland

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://invent.kde.org/plasma/kwin/-/raw/master/src/plugins/screenshot/org.kde.KWin.ScreenShot2.xml | 2026-11-19 | 2026-11-19 | live | WebFetch | Methoden, Optionen, Rückgabe, Pipe | topics/kwin-screenshot.md |
| https://invent.kde.org/plasma/kwin/-/raw/master/src/plugins/screenshot/screenshotdbusinterface2.cpp | 2026-11-19 | 2026-11-19 | live | WebFetch | Pipe, raw-Format; Berechtigungsprüfung nicht in dieser Datei | topics/kwin-screenshot.md |
| https://bugs.kde.org/show_bug.cgi?id=435696 | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Exec-Pfad muss absolut sein | topics/kwin-screenshot.md |
| https://mail.kde.org/pipermail/kde-frameworks-devel/2020-June/111948.html | 2026-11-19 | 2026-11-19 | indirect | WebSearch | Einführung X-KDE-DBUS-Restricted-Interfaces | topics/kwin-screenshot.md |

## Name und Lizenz

| URL | Letzter Besuch | Erstbesuch | Status | Weg | Inhalt | Zieldatei |
|---|---|---|---|---|---|---|
| https://api.github.com/search/repositories?q=schnapp+in:name | 2026-11-19 | 2026-11-19 | live | curl | 129 Treffer, pricerc/schnapp ähnlich | docs/open-decisions.md (E-08) |
| https://aur.archlinux.org/rpc/v5/search/schnapp | 2026-11-19 | 2026-11-19 | live | curl | 0 Treffer | docs/open-decisions.md (E-08) |
| https://flathub.org/api/v2/search?q=schnapp | 2026-11-19 | 2026-11-19 | blocked | curl | HTTP 405 | — |
| https://www.gnu.org/licenses/gpl-3.0.txt | 2026-11-19 | 2026-11-19 | live | curl | GPL-3.0-Text | LICENSE |

## Suchanfragen

| Datum | Anfrage | Ergebnis |
|---|---|---|
| 2026-11-19 | Codeberg terms of use AI generated code projects not allowed 2026 | Verbot bestätigt |
| 2026-11-19 | Flathub policy AI generated applications submission requirements LLM | Offenlegungspflicht seit Okt. 2026 |
| 2026-11-19 | KWin X-KDE-DBUS-Restricted-Interfaces desktop file Exec path … | absoluter Exec-Pfad nötig |
| 2026-11-19 | "schnapp" screenshot tool linux github OR flathub OR AUR | kein gleichnamiges Werkzeug |
