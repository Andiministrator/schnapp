# ADR-0019: Verzeichnisse und Go-Pakete englisch

## Status
Akzeptiert (Andi, 19.11.2026: Variante a — Verzeichnisse und Pakete englisch, Bezeichner bleiben)

## Kontext
Das Repo ist öffentlich, Oberfläche und Nutzer-Doku sind englisch (ADR-0014). Die oberste Ebene des Repos ist aber
deutsch (`aufnahme`, `bibliothek`, `oberflaeche`, …). Bei Go ist der Verzeichnisname zugleich Paketname und steht in
jedem Import und jedem Aufruf; wer von außen beiträgt, liest ihn in jeder Zeile. Umlaute müssen umschrieben werden
(`oberflaeche`). Mit dem Editor (Karte #35) kommen viele neue Dateien und Importe dazu — später wird das Umbenennen
teurer.

ADR-0002 erlaubt deutsche oder englische Go-Bezeichner, einheitlich je Paket. Das bleibt.

## Entscheidung
Verzeichnisse und Paketnamen im Repo sind englisch. Die Bezeichner in den Paketen (Typen, Funktionen, Felder) bleiben,
wie sie sind (also z. B. `capture.Quelle`, `library.Ablegen`); Kommentare, Commits, ADRs und interne Doku bleiben
deutsch (ADR-0002). Dateinamen unter `docs/` und `knowledge/` bleiben deutsch.

| bisher | neu |
|---|---|
| `aufnahme` | `capture` |
| `auswahl` | `selection` |
| `bibliothek` | `library` |
| `datei` | `fileformat` |
| `dokument` | `document` |
| `einstellungen` | `settings` |
| `oberflaeche` | `ui` |
| `sprache` | `i18n` |
| `marke` | `brand` |
| `extern` | `third_party` |
| `prototyp` | `prototype` |

Ausgenommen bis auf Weiteres: **`zeichnen`**. Es ist ein eigenes Modul (`…/schnapp/zeichnen`, ADR-0012), das ein
Schwesterprojekt einbindet; der Modulpfad ändert sich nur nach Abstimmung dort, als eigener Schritt (Ziel `draw`).

Unverändert, weil schon englisch: `cmd`, `data`, `docs`, `knowledge`, `po`, `tmp`.

## Konsequenzen
- Ein einziger Umbau-Commit (`git mv` samt Importen, `replace`-Pfaden, Skripten, CI und laufender Doku), damit die
  Geschichte an einer Stelle springt; `git log --follow` findet die Dateien weiter.
- Historische Texte (ältere ADRs, Finding-Archiv, Commit-Nachrichten) behalten die alten Namen; diese Tabelle ist die
  Übersetzung.
- Neue Pakete bekommen englische Namen.

## Nachtrag 1 (20.11.2026): `zeichnen` → `draw`

Nach Abstimmung mit dem Schwesterprojekt (es meldete „bereit für draw“) ist das Modul umbenannt: Ordner `draw/`,
Modulpfad `github.com/Andiministrator/schnapp/draw`, Pakete `…/draw/svg`, `…/draw/raster`, `…/draw/geo`, Schnittstellen
unverändert. Übergangsweise zeigt ein Symlink `zeichnen → draw` auf den neuen Ordner, damit relative Pfade des
Schwesterprojekts weiter auflösen; er wird entfernt, sobald es „steht auf draw“ meldet. Go prüft beim `replace` die
`module`-Zeile, der Symlink allein hält das Schwesterprojekt also nicht baubar — es stellt `require`, `replace` und
Importe im selben Zug um.

Symlink entfernt am 20.11.2026, nachdem das Schwesterprojekt „steht auf draw“ gemeldet hatte (dort require, replace und Importe umgestellt, Bilder bytegleich).

Fehlertexte beginnen mit dem heutigen Paketnamen (F-114, Sprint 10).

## Nachtrag 2 (21.11.2026): Reichweite der Präfix-Regel für Fehlertexte (F-327)

**Status:** Akzeptiert (Claude, Andi informiert).

- Der Satz „Fehlertexte beginnen mit dem heutigen Paketnamen“ (F-114) gilt für **jedes** Paket unter `internal/` und
  für die Fehler, die das Paket verlassen (exportierte Fehlerwerte und Rückgaben exportierter Funktionen). Innere
  Fehler, die an der Paketgrenze mit `%w` und Präfix umhüllt werden, brauchen kein eigenes.
- Umgestellt sind seit F-114 `document`, `fileformat`, `library`, `selection`, `canvas`, `wlclip`, `wlwire`.
  **Noch nicht:** `capture`, `portal`, `probeguard`, `export`, `i18n` (Sprint 10 übersehen). Keine Ausnahme — sie
  werden nachgezogen (F-339, Karte #150).
- Oberflächentexte (`i18n.T`) sind keine Fehlertexte im Sinne der Regel; sie tragen kein Präfix.

## Nachtrag 3 (22.11.2026): Hülle `internal/errprefix` (F-362, Sitzungs-QA 22.11. Q-7)

**Status:** Akzeptiert (Claude Opus 5.5, Andi informiert).

- Umgesetzt sind `document`, `fileformat`, `export`, `library` (F-362) und `rescue`; dazu `capture` (F-339). Ungeprüft
  sind `render`, `canvas`, `portal`, `settings`, `saveprogress` (F-379, Karte #168).
- **`internal/errprefix`:** `Wrap(paket, err)` hüllt mit `"<paket>: %w"` und lässt einen Text, der schon mit dem
  Präfix beginnt, unverändert. Ein `errors.Join` wird zweigweise gehüllt und bleibt ein Join, weil
  `cmd/schnapp` (`sammleOhneUebersetzt`) den Baum der Speicherfehler durchgeht. Erkannt wird der Join am Typnamen
  `*errors.joinError` der Standardbibliothek; `TestWrap` fällt, wenn Go ihn umbenennt. `Defer("paket", &fehler)` hüllt
  das benannte Fehlerergebnis einer exportierten Funktion beim Verlassen.
- **Block-Fehler in `fileformat`:** ein Fehler im Block snAO lautet „fileformat: snAO: …“ (`blockFehler`), nicht mehr
  „snAO: fileformat: …“.
- **Oberfläche:** Die Statusleiste nimmt die Paketpräfixe vor der Anzeige ab (`ohnePaketpraefix` beim Export), die
  Paketnamen sagen dem Nutzer nichts.

