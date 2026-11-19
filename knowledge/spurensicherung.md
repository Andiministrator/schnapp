# Spurensicherung — Recherche nachvollziehbar halten

Ziel: Jede fachliche Aussage (KWin/Wayland, Gio, Dateiformate, Lizenzen, Plattformregeln) lässt sich auf eine Quelle
zurückführen. Was nicht belegt ist, wird als Vermutung gekennzeichnet. Wissensbasis: Deutsch, Markdown, `knowledge/`.

## Pflicht-Workflow

1. **Vor der Recherche:** `knowledge/README.md` (Topic-Index) und `knowledge/sources-visited.md` prüfen — wurde das
   schon recherchiert, und wann?
2. **Jede besuchte URL** in `knowledge/sources-visited.md` eintragen, auch Fehlschläge — eine Zeile pro URL, bei
   erneutem Besuch „Letzter Besuch" aktualisieren. Format, Status-Werte und Zählbefehl stehen nur dort im Kopf.
3. **Ergiebige Quellen** als eigene Datei unter `knowledge/sources/` festhalten (Schema unten).
4. **Erkenntnisse verdichten** in eine Topic-Datei unter `knowledge/topics/` und im Topic-Index eintragen.
5. **In der Antwort** auf die Quelle verweisen, nicht auf das Gedächtnis.

**Sub-Agents** schreiben nicht in `sources-visited.md` (gemeinsame Datei); sie liefern ihre Zeilen im
Abschlussbericht, der Orchestrator trägt ein.

## Source-Datei-Schema (`knowledge/sources/<kurzname>.md`)

```
# <Titel der Quelle>
- URL: …
- Abgerufen: YYYY-MM-DD
- Herausgeber / Stand: … (bei Software: Version)
- Verlässlichkeit: offiziell · Hersteller · Community · unbekannt
## Kernaussagen
- … (mit Abschnitts-/Zeilenangabe, Zitate höchstens 20 Wörter)
## Offene Punkte
```

## Topic-Dateien

`knowledge/topics/<thema>.md`; beginnen mit `Stand: YYYY-MM-DD`, nennen ihre Quellen (über `sources-visited.md`
oder Abschnitt `## Quellen`). Jede Topic-Datei steht im Topic-Index `knowledge/README.md`.

## Regeln

- **Belegt vs. vermutet trennen:** Unbelegtes als „⚠️ unbelegt" markieren, nie als Tatsache formulieren.
- **WebFetch-Zusammenfassungen sind keine Zitate:** Wortlaut per `curl` oder im Browser gegenprüfen, bevor er in ein
  ADR oder in Code-Kommentare geht (bei PDFs halluziniert WebFetch nachweislich, Erfahrung aus einem anderen Projekt).
- **Öffentliches Repo:** `knowledge/` wird mit veröffentlicht. Keine Pfade, Namen, Hausnetz- oder Kundenangaben.
- Aussagen über Software immer mit Version (z. B. KWin 6.8.3).
- Nur öffentliche Seiten; nichts anmelden, keine Formulare absenden. Quelltext über öffentliche Repos oder
  Distributions-Pakete lesen, nie laufende Prozesse beobachten.
- Widersprechen sich Quellen, beide nennen und den Widerspruch festhalten.
- Arbeitsdateien nach `tmp/` im Projekt, nie nach `/tmp`.
