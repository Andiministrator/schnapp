# ADR-0004: QA- und Kritik-System

## Status
Akzeptiert (19.11.2026)

## Kontext
Ein Bildwerkzeug mit Schwärzen, eingebetteten Originalen und Metadaten kann still Daten preisgeben. Fehler dieser
Art fallen beim Benutzen nicht auf, sondern erst, wenn jemand die Datei auseinandernimmt.

## Entscheidung
Zwei Ebenen nach dem aBoard-Framework:
1. **Schritt-QA** nach jedem Schritt (`docs/qa/schritt-qa.md`), mit Gegenprobe und ADR-Gegenrichtung.
2. **Sprint-Abschluss** (`docs/qa/sprint-abschluss.md`, Command `/sprint-abschluss`): Abhängigkeits-Check,
   Kritiker-Lauf als Sub-Agent, Verify-Lauf über die Nacharbeit (anderes Modell, frischer Kontext), Triage.

Befunde stehen in `docs/findings.md` mit Typ (`[S]`, `[K]`, `[B]`, `[V]`, `[D]`) und Priorität, jeder mit Board-Karte.
Für Schnapp gelten als `[S]`-Pfad ausdrücklich: Schwärzen/Verpixeln, das eingebettete Original, Metadaten aller
Exporte, Zwischenablage-Inhalt, die Bibliothek (Speicherort, Rechte).

## Konsequenzen
- Jeder Exportweg bekommt einen Test, der prüft, dass geschwärzte Pixel in **keinem** Teil der Datei (sichtbares Bild,
  SVG-Block, Original) überleben.
- Kein `v*`-Tag ohne bestandenen Sprint-Abschluss (ADR-0006).
