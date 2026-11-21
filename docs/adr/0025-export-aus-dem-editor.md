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

## Nachtrag 1 (20.11.2026): Nacharbeit aus der Kritik (Fable)

- **Rechte (A-3), ersetzt Punkt 5 „Rechte 0644“:** Die temporäre Datei entsteht mit `O_EXCL` und Modus 0666; der Kern
  wendet die umask an. Ersetzt der Export eine vorhandene reguläre Datei, übernimmt die neue deren Rechte (eine 0600-Datei
  bleibt 0600). Ein Fehler beim Übernehmen der Rechte (etwa exFAT) bricht nicht ab, er erscheint nur im Debug-Log. Es gibt
  keinen festen Wert mehr.
- **Kein stilles Überschreiben bei angehängter Endung (A-1):** Hängt der Editor die Endung an (der Dialog bestätigte
  nur den Namen ohne Endung) und gibt es das Ziel schon, schreibt er nicht, sondern meldet „A file with this name
  already exists: <Dateiname>“. Bestätigt der Dialog den vollständigen Namen, bleibt das Ersetzen wie in Punkt 5.
- **Getrennte Busverbindung (A-2):** Schließt der Bus die Verbindung, während der Dialog offen ist, beendet das
  Abonnement seine Goroutine und liefert einen Fehler (`ErrVerbindung`); `Speichern` kehrt damit zurück, der
  Export-Knopf wird wieder bedienbar. Endet der Kontext, während `SaveFile` noch läuft, wird die Anfrage unter dem
  erwarteten Pfad geschlossen (A-8).
- **Absenderprüfung (A-7):** Die Match-Regel für `Request.Response` nennt den Absender `org.freedesktop.portal.Desktop`;
  zusätzlich verwirft der Filter Signale, deren Absender weder dieser Name noch dessen eindeutiger Besitzer ist
  (`GetNameOwner` beim Anmelden; scheitert das, gilt nur der Name).
- **Meldungen (A-6, A-12):** Fehler des Exports erscheinen übersetzt und ohne Pfad (bei Betriebssystemfehlern nur die
  Ursache). Solange der Dialog offen ist, zeigt die Statusleiste „File dialog is open.“; der Dialogtitel heißt „Export“
  ohne Auslassungspunkte. Ein laufender Text wird vor dem Schreiben bestätigt, ein laufendes Ziehen lehnt den Export
  ab (A-5).
- **Offen vor einer Flathub-Einreichung (A-9, nur Hinweis, nicht umgesetzt):** Unter Flatpak gibt das Dokumenten-Portal
  nur Schreibrecht auf die gewählte Datei. Die temporäre Datei daneben und die angehängte Endung liegen außerhalb
  dessen; Punkt 5 funktioniert dort so nicht und braucht dann einen anderen Weg (etwa direkt in die gewählte Datei
  schreiben oder den Dateideskriptor des Portals nutzen).

## Nachtrag 2 (20.11.2026): Verify der Nacharbeit (Fable)

- **Besitzer des Portals nachschlagen (N-1):** Der Bus nennt als Absender eines Signals immer den eindeutigen Namen
  (`:1.x`), nie `org.freedesktop.portal.Desktop`. Ist der Besitzer beim Anmelden noch unbekannt (das Portal startet
  erst mit `SaveFile` per Aktivierung), fragt die Prüfung ihn beim ersten fremden Absender erneut ab; sonst verwürfe
  sie die echte Antwort, und der Export hinge. Ersetzt den Satz „scheitert das, gilt nur der Name“ aus Nachtrag 1.
- **Rechte vor dem Schreiben (N-2):** Ersetzt der Export eine vorhandene Datei, bekommt die temporäre Datei deren
  Rechte, bevor der Inhalt geschrieben wird — sie ist nie offener als das Ziel.
- **Symbolischer Link als Ziel (N-3):** Geschrieben wird in die Datei, auf die der Link zeigt, mit deren Rechten; der
  Link bleibt. Ein Link ins Leere wird abgelehnt.
- **Angehängte Endung ohne Ersetzen (N-4):** Hat der Editor die Endung angehängt, legt er die Datei per hartem Link an,
  der an einem inzwischen entstandenen Ziel scheitert (`ErrZielVorhanden`), statt nur vorher zu prüfen. Kann das
  Dateisystem keine harten Links, wird bestmöglich vorher geprüft. Ein Prüffehler außer „nicht vorhanden“ (etwa keine
  Rechte) erscheint als solcher, nicht als „vorhanden“.

## Nachtrag 3 (21.11.2026): Rückfall ohne temporäre Datei (F-196, Karte #134)

**Status:** Akzeptiert (Claude, Andi informiert; umgesetzt von Claude Sonnet 5.5). *Karte berichtigt 21.11.2026 abends
(F-327): F-196 liegt auf #134, nicht auf #94.*

- Darf neben dem Ziel keine temporäre Datei entstehen (`EACCES`/`EPERM`/`EROFS`), schreibt `SchreibeAtomar` ein **vorhandenes, reguläres, beschreibbares** Ziel direkt (`O_TRUNC`), nachdem das Dokument vollständig in den Speicher gerendert wurde; ein Renderfehler lässt das Ziel unberührt. Das ist nicht atomar und steht im Debug-Log. *Seit Nachtrag 4 mit Sicherung des alten Inhalts und `O_NOFOLLOW`.* `SchreibeOhneErsetzen` und neue Ziele behalten den bisherigen Fehler.
- **Offen (nur unter Flatpak prüfbar):** Das Dokumenten-Portal gibt nur die gewählte Datei frei. Eine angehängte Endung führt zu einem anderen, nicht freigegebenen Namen. Die richtige Lösung ist, die Endung dem Dialog zu überlassen (Vorschlagsname `current_name` mit Endung je Filter) und unter `/run/user/<uid>/doc/` nie anzuhängen. Das braucht einen Flatpak-Bau zur Prüfung (Finding F-196 bleibt offen bis Flathub).

## Nachtrag 4 (21.11.2026): Rückfall mit Sicherung (F-322, Karte #136)

**Status:** Akzeptiert (Claude, Andi informiert).

- Der direkte Schreibweg aus Nachtrag 3 kürzt das Ziel, bevor der neue Inhalt steht. Er liest deshalb die vorhandene
  Datei vorher in den Speicher, höchstens `MaxRueckfallAlt` (256 MiB), und nur, wenn die gelesene Länge der Größe beim
  `Lstat` gleicht und es dieselbe Datei ist. Größere oder veränderte Dateien werden nicht direkt beschrieben, es gilt der
  ursprüngliche Fehler.
- Scheitern `Write`, `Sync` oder `Close`, schreibt er den alten Inhalt zurück und meldet den Schreibfehler. Misslingt
  auch das, endet der Export mit `ErrZielBeschaedigt`, und die Statusleiste nennt die Datei ausdrücklich als beschädigt.
- Ziel und Sicherung werden mit `O_NOFOLLOW` geöffnet.
- Der Rückfall bleibt auch im gewöhnlichen schreibgeschützten Ordner: Er greift dort nur, wenn die gewählte Datei selbst
  beschreibbar ist, und dann ist der Wunsch, genau diese Datei zu ersetzen, erfüllbar. Unter Flatpak (Dokumenten-Portal)
  ist er der einzige Weg.
- Nicht abgedeckt bleibt ein Prozessabbruch oder Stromausfall zwischen Kürzen und Zurückschreiben (nicht atomar, bewusst).
- *Ergänzt 21.11.2026 abends (F-330, Karte #144):* Das Ziel wird ohne `O_TRUNC` geöffnet (`O_NOFOLLOW`, `O_NONBLOCK`),
  über den geöffneten Deskriptor als reguläre Datei und mit `os.SameFile` gegen die gesicherte Datei geprüft und erst
  dann per `Truncate(0)` gekürzt; dasselbe beim Zurückschreiben. Tauscht jemand den Namen zwischen Sichern und Öffnen,
  bleibt die fremde Datei unverändert, ein untergeschobenes FIFO blockiert nicht. Offen und nur unter Flatpak prüfbar:
  ob `Truncate` nach dem Öffnen auf dem FUSE des Dokumenten-Portals geht (F-196).
- *Ergänzt 21.11.2026 abends (F-338, Karte #149):* Das Ziel wird **einmal** geöffnet (`O_RDWR|O_NOFOLLOW|O_NONBLOCK`);
  Sichern, Kürzen, Schreiben und Wiederherstellen laufen über diesen einen Deskriptor. Damit sichert der Rückfall genau
  den Inhalt, den er dann überschreibt, und stellt ihn in genau diese Datei wieder her — auch wenn der Name inzwischen
  woanders hinzeigt oder das Dateisystem die Inode einer gelöschten Datei wiederverwendet hat. Hält ein anderes
  Programm eine Lease, scheitert das Öffnen mit `O_NONBLOCK` sofort mit `EAGAIN`; es wird bis zu 20-mal im Abstand von
  50 ms wiederholt, danach gilt der Fehler der temporären Datei (Datei unverändert). Offen und nur unter Flatpak
  prüfbar: ob das Dokumenten-Portal `O_RDWR` zulässt (F-196).
- *Ergänzt (F-344, Karte #153):* `ErrZielBeschaedigt` nur, wenn die Datei verändert wurde (Kürzen gelang) und die
  Wiederherstellung scheiterte; scheitert schon das erste Kürzen, kommt die Ursache zurück. Ein `Close`-Fehler nach
  gelungenem `Sync` ist kein Exportfehler (der Inhalt steht vollständig), er geht in die Diagnose.
