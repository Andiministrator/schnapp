# ADR-0014: Mehrsprachige Oberfläche und Nutzer-Doku (Englisch und Deutsch)

## Status
Akzeptiert (Andi, 19.11.2026: „Die Dokumentation und Bedienoberfläche soll mehrsprachig werden (am Anfang mind.
deutsch und englisch – Voreinstellung ist englisch, außer wir können die Sprache des Nutzers am System erkennen)“).
Ändert ADR-0002 (erste Oberflächensprache) und entscheidet E-07.

## Kontext
Schnapp wird öffentlich (GitHub, später ggf. Flathub). ADR-0002 hatte Deutsch als erste Oberflächensprache festgelegt
und Englisch auf „vor der Veröffentlichung“ verschoben (E-07). Nachträgliches Übersetzen ist teurer als von Anfang an
übersetzbar zu bauen; Flathub und Übersetzungsplattformen (z. B. Weblate) erwarten gängige Formate.

## Entscheidung

### Oberfläche
- Sprachen zum Start: **Englisch und Deutsch**. Weitere Sprachen kommen als Übersetzungsdateien dazu, ohne Code.
- **Ausgangssprache der Texte ist Englisch** (Schlüssel und Rückfall); Deutsch ist eine Übersetzung wie jede andere.
- **Sprachwahl:** Einstellung in Schnapp (falls gesetzt) → Systemsprache nach POSIX-Reihenfolge `LANGUAGE`
  (Liste, z. B. `de:en_US`), `LC_ALL`, `LC_MESSAGES`, `LANG` → **Englisch**. Aus einem Wert wie `de_DE.UTF-8` zählt
  die Sprache (`de`); `C` und `POSIX` heißen „keine Angabe“. Im Flatpak setzt die Laufzeit dieselben Variablen.
- **Format:** gettext-PO-Dateien (`po/<sprache>.po`), ins Programm eingebettet (`go:embed`). PO kennen Übersetzer,
  Weblate und Flathub-Werkzeuge; Mehrzahlformen sind vorgesehen. Ob ein eigener kleiner PO-Leser oder ein Paket
  (z. B. `github.com/leonelquinteros/gotext`, MIT) liest, entscheidet die Umsetzung nach ADR-0011.
- Texte stehen nie verstreut im Code, sondern laufen über eine Übersetzungsfunktion; Zahlen- und Datumsformate folgen
  der gewählten Sprache.
- Desktop-Datei und AppStream-Metadaten tragen beide Sprachen (`Name[de]=…`, `<name xml:lang="de">`).

### Dokumentation
- **Nutzer- und Beitragenden-Doku zweisprachig:** `README.md`, Handbuch, `CONTRIBUTING.md` und Store-Texte auf
  **Englisch als Hauptfassung**, Deutsch daneben als `*.de.md`. Jede Fassung verlinkt oben die andere.
- **Projektinterne Doku bleibt Deutsch** (ADR-0002): ADRs, Findings, Wissensbasis, QA-Listen, Commits,
  Code-Kommentare. Ein Übersetzen dieser Arbeitsdokumente lohnt nicht; die englische README sagt das.
- Ändert sich eine Fassung, wird die andere im selben Commit nachgezogen. Fehlt die Übersetzung ausnahmsweise, steht
  oben ein sichtbarer Hinweis „noch nicht übersetzt“.

## Konsequenzen
- Die Sprachwahl ist eine kleine, testbare Funktion (Umgebung rein, Sprache raus) — mit Tests für die Reihenfolge,
  `C`/`POSIX` und unbekannte Sprachen.
- Der Prototyp (`prototyp/`) bleibt einsprachig; er ist Wegwerf-Code.
- Neue Oberflächentexte brauchen in jedem Commit beide Sprachen; ein Test prüft, dass `de.po` keinen Schlüssel
  vermisst.
- CLAUDE.md, ADR-0002 und E-07 werden angepasst.

## Nachtrag 1 (19.11.2026): Umsetzung — eigener PO-Leser (Karte #19)

**Status:** Akzeptiert (Claude Opus 5.5 im Rahmen von Karte #19; die Wahl war der Umsetzung überlassen).

- **Eigener kleiner PO-Leser statt `gotext`** (ADR-0011): Schnapp braucht nur PO lesen, Kontext, Mehrzahl und die
  Sprachwahl — rund 400 Zeilen mit Standardbibliothek, voll getestet. `gotext` bringt MO-Dateien, Locale-Ordner und
  eigene Sprachwahl mit, die Schnapp nicht nutzt; ein Paket weniger zu prüfen und zu pflegen.
- **Pakete im Wurzelmodul** `github.com/Andiministrator/schnapp`: `sprache` (Sprachwahl `Waehle`, PO-Leser `LesePO`,
  Mehrzahlregel aus `Plural-Forms` mit allen C-Ausdrücken, die gettext dort kennt, `Katalog` mit `T`, `TK` für
  Kontext, `N` für Mehrzahl; Paketfunktionen `sprache.T/TK/N` nach `sprache.Starte`) und `po` (bettet `po/*.po` ein).
- **Sprachwahl genauer als oben:** aus `LC_ALL`, `LC_MESSAGES`, `LANG` zählt — wie bei POSIX — nur die erste gesetzte
  Variable; ist sie `C`, `POSIX` oder eine Sprache ohne PO-Datei, gilt Englisch (kein Weiterfallen auf `LANG`).
  `LANGUAGE` ist dagegen eine Liste, deren Einträge der Reihe nach probiert werden. Je Eintrag erst `ll_CC`, dann `ll`.
- **Prüfung:** Ein Test durchsucht alle `.go`-Dateien des Wurzelmoduls nach `sprache.T`, `sprache.TK`, `sprache.N`.
  Jeder Text braucht in jeder `po/*.po` einen fertigen Eintrag (nicht `fuzzy`, nicht leer, passende Mehrzahl), jeder
  Eintrag einen Text im Code (sonst veraltet), und jedes Argument muss ein Zeichenketten-Literal sein.
- **Noch nicht umgesetzt:** Vorlage `po/schnapp.pot` für Übersetzer und Weblate (aus derselben Code-Durchsuchung zu
  erzeugen, sobald es Texte gibt), Zahlen- und Datumsformate je Sprache, die Einstellung in Schnapp (Parameter
  `einstellung` ist vorgesehen). Karte im Board.
