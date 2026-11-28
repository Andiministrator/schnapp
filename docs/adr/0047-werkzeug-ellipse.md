# ADR-0047: Werkzeug Ellipse (Kreis mit Umschalt) als Form des Kastens

## Status
Akzeptiert (Claude Opus 5.5, 28.11.2026, Karte #328 in Sprint 14; Vorschlag Claude, von Andi am 25.11.2026 übernommen;
Andi 28.11.: „arbeite selbstständig“). Ergänzt ADR-0021 (Zeichenwerkzeuge), ADR-0008 (Format), ADR-0037 (Schnellstile),
ADR-0041 (Stil je Werkzeug).

## Kontext
Zum Hervorheben fehlt ein Kreis bzw. eine Ellipse — in Snagit, Spectacle und ksnip ein Grundwerkzeug. Die Karte nennt
die Ellipse eine Variante des Kastens: dieselben Eigenschaften (Strich mit Breite, Farbe und Strichart, Füllung,
Ebene, Schatten), dieselben acht Griffe am umschließenden Rechteck, dasselbe Aufziehen mit Umschalt für gleiche Seiten.
Die Werkzeugleiste mit Varianten je Werkzeug (#223) braucht erst ein Muster.

## Entscheidung
1. **Daten: eine Form am Kasten.** `document.Kasten` bekommt `Form` (`Kastenform`: `FormRechteck` = Nullwert,
   `FormEllipse`). Die Ellipse ist das größte in `Bereich` passende Oval; `Eckradius` gilt für sie nicht (bleibt
   gespeichert). Alles, was den Kasten über `Bereich`, `Strich`, `Fuellung` und die Ebene behandelt — Griffe, Verschieben,
   Einrasten an den Kanten und Mitten des Bereichs, Ausrichten, Gruppen, Kopieren, Schatten, Deckkraft, Mehrfachstil —
   gilt unverändert.
2. **Eine Rechnung für alle.** `document.EllipsenUmriss` tastet die Ellipse als geschlossenen Linienzug ab, gleichmäßig im
   Parameter, so dicht, dass die Sehne höchstens 0,02 px vom Bogen abweicht (wie `bogenSehne`), mit den vier Scheiteln als
   Punkten, höchstens 1024 Punkte je Viertel. Renderer und Editor füllen diesen Umriss und zeichnen den Strich als
   überlappungsfreie Teile des geschlossenen Linienzugs (`render.ZerlegeEllipse` über `neuerLinienStrich`), je
   sichtbarem Bereich beschnitten wie der runde Kasten (ADR-0023 Nachträge 17 bis 21). Gemustert (ADR-0021 Nachtrag 25)
   teilt `document.EllipsenAbschnitte` den Umriss in vier Viertel von Scheitel zu Scheitel mit halbem Strich bzw. Punkt an
   jedem Scheitel (wie der runde Kasten je Rundung). Ein Bereich ohne Breite oder Höhe wird wie ein Kasten ohne Fläche
   gezeichnet (Balken).
3. **Treffer:** mit Füllung im Umriss, ohne Füllung nur auf dem Strich (Abstand zum Umriss höchstens halbe Breite plus
   Toleranz) — beim Rechteck wie bisher.
4. **Werkzeug Ellipse**, Taste **E** (war frei), Symbol `icons.Ellipse`, in der Werkzeugleiste nach dem Rechteck.
   Aufziehen von Ecke zu Ecke, **Umschalt = Kreis** (dieselbe Rechnung wie das Quadrat des Kastens, `umschaltEnde`).
   Eigener Stil (ADR-0041), Abschnitt `[ellipse]` in `tools.ini`, Voreinstellung und Farbpaar wie der Kasten.
5. **Schnellstile:** eigene Art `ellipse` in `styles.ini`, eingebaut dieselben Stile wie der Kasten (wie die Sprechblase
   die des Texts, ADR-0021 Nachtrag 13); die Kacheln zeigen eine Ellipse.
6. **Format:** Attribut `s:form="ellipse"` am Kasten (§8.2), nur bei der Ellipse geschrieben. **FormatVersion 2** mit einer
   Ellipse (`versionEllipse`); in Version 1 ist `s:form` unbekannt und wird ignoriert (Regel 3, §11). Die Arbeitsdatei
   schreibt das Element `<ellipse cx cy rx ry>` (der Leser unterscheidet die Elemente nicht), der Export ebenso, gemustert
   als Gruppe mit Füllung und Stücken wie der gemusterte Kasten.
7. **Bedienung ohne Fenster:** `--annotate`/MCP kennen den Typ `ellipse` mit denselben Feldern wie `box` (ohne
   `corner_radius`).
8. **Ebenenliste und Leiste** nennen das Objekt „Ellipse“; Vorschau in der Liste als Ellipse.

## Erwogene Alternativen
- **Eigener Objekttyp `document.Ellipse`:** sauber getrennt, aber jede Stelle, die heute den Kasten kennt (Griffe,
  Einrasten, Mehrfachstil, Schnellstile, Schatten, Strichart, Treffer, Format, Zwischenablage), bräuchte einen zweiten
  Zweig mit derselben Logik. Die Form am Kasten hält diese Stellen gleich.
- **`stroke` im Editor über Gios `clip.Ellipse`:** Gios Strich deckt sich nicht mit dem Renderer (andere Teilung, kein
  Muster, Überdeckung bei Deckkraft unter 100 %); dieselben Teile wie der Renderer sind die Regel seit ADR-0023 Nachtrag 21.

## Konsequenzen
- Neue Formen des Kastens (später etwa Raute) kommen als weiterer Wert von `Kastenform`.
- Ältere Schnapp-Stände (Version 1) zeigen eine Ellipse aus einer Datei der Version 2 nicht; sie lehnen Version 2 ab,
  wie bei Lupe und Sprechblase.
