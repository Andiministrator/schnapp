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
  *Genaue Regel (wie glibc-gettext): Nachtrag 1.*
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
  Sprachwahl — knapp 800 Zeilen Quelltext (`wc -l sprache/{po,mehrzahl,katalog,wahl}.go`, Stand Sprint-2-Abschluss), nur
  Standardbibliothek, voll getestet. `gotext` bringt MO-Dateien, Locale-Ordner und
  eigene Sprachwahl mit, die Schnapp nicht nutzt; ein Paket weniger zu prüfen und zu pflegen.
- **Pakete im Wurzelmodul** `github.com/Andiministrator/schnapp`: `sprache` (Sprachwahl `Waehle`, PO-Leser `LesePO`,
  Mehrzahlregel aus `Plural-Forms` mit allen C-Ausdrücken, die gettext dort kennt, `Katalog` mit `T`, `TK` für
  Kontext, `N` für Mehrzahl; Paketfunktionen `sprache.T/TK/N` nach `sprache.Starte`) und `po` (bettet `po/*.po` ein).
- **Sprachwahl genauer als oben, wie GNU gettext** (nach Kritiker-Befund F-39 angeglichen): Die Locale ist die erste
  gesetzte von `LC_ALL`, `LC_MESSAGES`, `LANG` — nur sie zählt, kein Weiterfallen. Ist sie leer, `C` oder
  `C.<Kodierung>` (etwa `C.UTF-8`), gilt Englisch, und `LANGUAGE` wird **nicht** beachtet; bei `POSIX` gilt `LANGUAGE`
  weiter (glibc `intl/dcigettext.c`, `guess_category_value`, gelesen am 19.11.2026, F-44). Sonst wird `LANGUAGE` als
  Liste der Reihe nach probiert; ein Eintrag `C`, `C.<Kodierung>` oder `POSIX` beendet sie mit Englisch. Danach die Locale selbst, sonst Englisch. Je Angabe erst `ll_CC`,
  dann `ll`; Groß-/Kleinschreibung wird nicht angeglichen (wie gettext).
- **PO-Leser strenger als msgfmt, wo es nichts kostet:** Kommentare nur vor einem Eintrag; doppelte Einträge,
  `msgid_plural` ohne `msgstr[n]`, Dateien über 8 MiB und `Plural-Forms`-Ausdrücke über 1024 Zeichen sind Fehler
  (Schutz vor Stapelüberlauf durch fremde Dateien, F-36). Byte-Order-Mark und alle C-Escapes werden gelesen (`\x` mit
  beliebig vielen Ziffern wie in C, Wert höchstens 0xff, F-45).
- **Prüfung:** Ein Test durchsucht alle `.go`-Dateien des Wurzelmoduls nach `sprache.T`, `sprache.TK`, `sprache.N`
  (auch unter Import-Alias; Punkt-Import ist verboten; Aufrufe über einen `*Katalog` findet er nicht, darum nutzt
  Programmcode nur die Paketfunktionen). Er verlangt in jeder PO-Datei den Kopf `Plural-Forms` und vergleicht die
  printf-Platzhalter zwischen Original und Übersetzung.
  Jeder Text braucht in jeder `po/*.po` einen fertigen Eintrag (nicht `fuzzy`, nicht leer, passende Mehrzahl), jeder
  Eintrag einen Text im Code (sonst veraltet), und jedes Argument muss ein Zeichenketten-Literal sein.
- **Noch nicht umgesetzt:** Vorlage `po/schnapp.pot` für Übersetzer und Weblate (aus derselben Code-Durchsuchung zu
  erzeugen, sobald es Texte gibt), Zahlen- und Datumsformate je Sprache, die Einstellung in Schnapp (Parameter
  `einstellung` ist vorgesehen). Karte im Board.
