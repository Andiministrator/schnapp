# ADR-0025: Export aus dem Editor (PNG, JPG, SVG) über den Dateidialog des Portals

## Status
Akzeptiert (Claude Opus 5.5, 20.11.2026, Karte #90 in Sprint 6). Andi hat die Karte freigegeben („Export-Dialog im
Editor für PNG, JPG, SVG; weitergeben ohne Bearbeitungsdaten“), die Einzelheiten nicht vorgegeben. Setzt A-50 teilweise
um (PDF und Dateinamen-Schema folgen).

## Kontext
`fileformat` kann PNG, JPG und SVG ohne Bearbeitungsdaten schreiben (ADR-0008); der Editor speichert bisher nur die
Arbeitsdatei in die Bibliothek. Zum Weitergeben braucht es „Exportieren unter …“ mit Ort und Format. Gio hat keinen
Dateidialog. Unter Wayland ist der übliche Weg das Desktop-Portal `org.freedesktop.portal.FileChooser` über D-Bus: KDE
zeigt seinen eigenen Dialog (mit Rückfrage beim Überschreiben), Flatpak bekommt Schreibrecht genau für die gewählte
Datei. `godbus` ist schon Abhängigkeit (Aufnahme, ADR-0007/ADR-0011). Ein eigener Dialog in Gio wäre viel Arbeit und
fremd; `zenity`/`kdialog` wären Laufzeit-Fremdprogramme (ADR-0011).

## Entscheidung

1. **Auslöser:** Knopf „Export…“ in der Werkzeugleiste (vor „Speichern“, rechts außen bleibt „Speichern“; die Leiste
   bricht seit F-148 um) und Tastenkürzel **Strg+Umschalt+E** (wie „Exportieren als“ in GIMP). Während der Texteingabe
   wirkungslos (wie Strg+C).
2. **Dialog:** Portal `FileChooser.SaveFile` auf dem Sitzungsbus, Elternfenster leer (Gio gibt kein xdg-foreign-Handle
   heraus), `modal` wahr, `current_name` = Name des Dokuments mit `.png`, Filter in dieser Reihenfolge: PNG
   (`image/png`), JPEG (`image/jpeg`), SVG (`image/svg+xml`); `current_filter` PNG. Die Antwort kommt asynchron über das
   Signal `Request.Response`; der Editor wartet in einer Goroutine und weckt das Fenster per `Invalidate`, geschrieben
   wird in der eigenen Ereignisschleife (wie die Signalwache, F-77). Abbruch im Dialog (Antwort 1) tut nichts und meldet
   nichts.
3. **Format:** aus der Dateiendung der gewählten URI (`.png`, `.jpg`/`.jpeg`, `.svg`, Groß-/Kleinschreibung egal);
   ohne bekannte Endung aus dem gewählten Filter, und die passende Endung wird angehängt. Nur `file://`-URIs; andere
   lehnt der Editor mit Meldung ab.
4. **Inhalt:** PNG und JPG aus demselben Renderer wie Arbeitsdatei und Zwischenablage (ADR-0020), JPG mit Qualität 90
   (Transparenz auf Weiß); SVG über `fileformat.ExportiereSVG`. Keine `snAP`/`snAO`-Blöcke, keine Metadaten mit Pfaden,
   Programm- oder Rechnernamen (Kernprinzip 3). Geschwärztes ist darin endgültig wie in jedem Export (ADR-0008); der
   bestehende Test über alle Exporte muss den Editor-Weg mit abdecken.
5. **Schreiben:** in eine temporäre Datei im Zielordner, dann umbenennen (nie eine halbe Datei am Ziel); Rechte 0644.
   Das Überschreiben bestätigt der Dialog. Fehler erscheinen übersetzt in der Statusleiste, Erfolg als „Exported to
   <Dateiname>“ (nur Name, kein Ordner).
6. **Ohne Portal** (kein Dienst, Fehler beim Aufruf): Meldung in der Statusleiste, kein Rückfall auf eine
   Ersatzablage — still woanders hinzuschreiben wäre schlimmer als nicht zu exportieren.

## Konsequenzen
- Neue Oberflächentexte (Englisch, Deutsch in `po/de.po`): Knopf, Filter-Namen, Meldungen.
- Die Portal-Anbindung liegt außerhalb von `ui` in einem testbaren Paket (Aufbau der Anfrage, Auswertung der Antwort
  und der URI mit Attrappe statt echtem Bus); `ui` ruft nur auf.
- Rendern und Schreiben laufen im Fensterthread wie Speichern und Kopieren (F-143, F-178); große Dokumente frieren das
  Fenster kurz ein, bis die Karte zu F-143 das ändert.
- PDF-Export und ein Dateinamen-Schema bleiben offen (A-50).
