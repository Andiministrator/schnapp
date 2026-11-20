# ADR-0020: Editor — wann er öffnet, wie er gebaut ist, wer die Pixel rendert

## Status
Akzeptiert (Andi, 19.11.2026: „ja zu deinem vorschlag“)

## Kontext
Schnapp nimmt auf und legt in der Bibliothek ab (Sprint 4); der Editor (Karte #35) fehlt. Offen sind drei Dinge:

1. **Wann öffnet der Editor?** Snagit öffnet ihn nach jeder Aufnahme. Schnapp gibt heute den Pfad auf stdout aus und
   beendet sich (für Skripte und Agenten, E-14, wichtig). Eine vorhandene Datei lässt sich noch gar nicht öffnen.
2. **Aufbau:** Wie im Overlay soll die Logik ohne Gio testbar sein (ADR-0003, Karte #34).
3. **Pixel der Datei:** Die Arbeitsdatei trägt ein fertiges Bild (ADR-0008), Exporte PNG/JPG und die Zwischenablage
   ebenso. Dafür müssen die Objekte (Pfeil, Kasten, Text, Nummer) in Pixel gerendert werden. Der eigene Rasterer
   `zeichnen/raster` kann Pfade, Rechtecke, Kreise und Linien, aber weder Text noch `<image>`.

## Entscheidung
1. **Ablauf:**
   - Nach einer interaktiven Aufnahme (`--region`, `--screen`, `--window`) öffnet der Editor im selben Prozess,
     nachdem die Aufnahme schon in der Bibliothek liegt und ihr Pfad auf stdout steht. Abschaltbar mit der
     Einstellung `[capture] open-editor = false` (Voreinstellung `true`) und für einen Aufruf mit `--no-editor`.
     `--repeat` öffnet ihn wie die anderen Modi.
   - `schnapp DATEI.png` öffnet eine vorhandene Datei im Editor (Arbeitsdatei mit Ebenen oder fremdes PNG/JPG).
   - Speichern schreibt eine Bibliotheksaufnahme an ihren Platz zurück (Bearbeitet = wahr). Eine Datei außerhalb
     der Bibliothek wird beim ersten Speichern als neue Aufnahme in die Bibliothek gelegt; das Original bleibt
     unberührt.
2. **Aufbau:** Paket `canvas` ohne Gio (Abbildung, Zoom zum Zeiger, Treffer, Griffe, Ziehen); Änderungen nur über
   Befehle und Züge von `document` (Rückgängig ohne Grenze, F-18: nicht während eines Zugs). `ui` zeichnet Bild und
   Objekte mit Gio-Vektorpfaden und übersetzt Eingaben. Der Editor öffnet erst, wenn alle Overlay-Fenster zu sind
   (F-82).
3. **Rendern:** ein Go-Renderer ohne GPU, der Ansichtsbild und Objekte zu einem `image.NRGBA` zusammensetzt — für
   Arbeitsdatei, PNG/JPG-Export und Zwischenablage. Formen über `golang.org/x/image/vector`, Schrift über
   `golang.org/x/image/font` mit den Go-Schriften (beides steckt über Gio schon im Abhängigkeitsbaum, BSD-Lizenz,
   ADR-0011). *Geändert bei der Umsetzung (#58):* nicht `zeichnen/raster` — das Wurzelmodul müsste dafür das eigene
   Modul `zeichnen` einbinden (require + replace), kurz vor dessen Umbenennung (#57), und dessen Strich-Funktionen
   sind nicht öffentlich. Das
   Bild auf dem Bildschirm zeichnet Gio; Abweichungen in Kantenglättung sind erlaubt, Lage und Größe nicht (Test:
   Treffer und Grenzen gleich).

## Konsequenzen
- Neue Karte „Renderer“ vor dem Speichern mit Objekten; der Editor kann vorher öffnen, zoomen und auswählen.
- `--no-editor` gehört zur Kommandozeile (ADR-0017), `[capture] open-editor` zu den Einstellungen (ADR-0018); beide
  kommen mit dem Editor (#35).
- Für Skripte bleibt der Pfad auf stdout das Erste, was Schnapp ausgibt; ohne Bildschirm (kein Wayland) öffnet kein
  Editor.

## Nachtrag 1 (19.11.2026): Speichern beim Schließen, Ausweichen (Kritik B-8, A-1, A-4)

- Ungespeicherte Änderungen werden beim Schließen ohne Rückfrage gespeichert (Claudes Entscheidung, Andi informiert)
  — auch wenn das Fenster mit Fehler endet. Bei einer Datei außerhalb der Bibliothek entsteht dabei eine neue
  Aufnahme der Bibliothek.
- Scheitert das Zurückschreiben in die Bibliotheksdatei — sie fehlt, ist fremd oder neuer, oder das Schreiben
  scheitert aus einem anderen Grund (Rechte, Datenträger; Verify V-1) —, weicht Speichern auf eine neue Aufnahme aus,
  statt die Bearbeitung zu verlieren. Das Ausweichen ist sichtbar (F-116): Titelleiste und Statusleiste nennen die
  neue Datei; ist die alte stehen geblieben, als Warnung, beim Schließen auf der Fehlerausgabe. Mit dem Schwärzen
  (eigene Karte) ist zu entscheiden, was mit einem stehen gebliebenen ungeschwärzten Stand geschieht.
- Scheitert das Vorschaubild, wird das alte entfernt; bleibt es trotzdem liegen, meldet der Editor das (F-115).
  Fehler beim Speichern erscheinen übersetzt und ohne Pfad (F-117).
- Bildschirm und Renderer nutzen dieselben Go-Schriften (keine Systemschriften); Tabulator wird als vier Leerzeichen
  gezeichnet. Schriftgrad und Nummer-Radius sind auf `document.MaxSchrift` = 4096 begrenzt ([S], Speicher); Zeichen je Text (`MaxTextZeichen` = 10000) und Objekte je Dokument (`MaxObjekte` = 10000)
  sind begrenzt, Überschreitung ist ein Fehler auch beim Lesen einer Arbeitsdatei ([S], Rechenzeit, F-118); der Renderer
  rastert Zeilen und Glyphen nicht, die das Bild nicht schneiden. Innerhalb dieser Grenzen schützt ein Rechenbudget
  gegen präparierte Dateien (F-125, F-131, F-132, Karte #71): Beim Öffnen im Editor rechnet `render.Pruefe` den
  Aufwand per Trockenlauf desselben Zeichners (Geometrie und Kosten, ohne Rastern; Millisekunden, außer bei sehr vielen Streifen — F-136) und lehnt das
  Dokument ab, wenn die Summe `render.Budget`, die Glyphen `render.GlyphBudget` oder die Kanten `render.KantenBudget`
  überschreiten; der Editor meldet „zu viele oder zu große Beschriftungen, um sie zu öffnen“. Kostenmodell: je Glyphe
  max(Grad², Grundkosten), je Füllung gewichtete Fläche des beschnittenen Rechtecks, je Polygonkante die Zeilenschritte
  des Rasterizers samt Ecken. **Speichern bricht nie ab:** `render.Bild` rastert immer vollständig, eine im Editor
  gemachte Bearbeitung geht nicht am Budget verloren (langsam ist besser als verloren). Gemessen 19.11.2026
  (Entwicklungsrechner, ein Kern): ein alltagsnahes großes Dokument (50 bildgroße Rahmen, 50 diagonale Pfeile, 100
  Nummern, 200 Texte auf 7680×2160) braucht 48 s und liegt bei knapp einem Fünftel des Budgets; ein
  Dokument knapp unter dem Budget rastert in knapp 5 min — das ist die Wartezeit, die eine präparierte Datei beim
  Speichern erzwingen kann (bis rund 5 min). Glyphen allein (`GlyphBudget` = 1e9) bringen höchstens rund 9 s, wenn sie
  sichtbar gezeichnet werden; Kanten allein 1,5 bis 4,5 s. Senken lässt sich das Budget
  erst mit schnellerem Füllen (F-130). Die Wartezeit bis knapp 5 min ist bewusst hingenommen (F-135, Claudes
  Entscheidung im Sprint-Abschluss 4, Andi informiert). *Zahlen überholt durch schnelleres Füllen: Nachtrag 3.*
- Offen (eigene Karten): Speichern bei Signal/Absturz und Konflikt zweier Editoren (F-107, F-110), Angleichen der
  Maße an den SVG-Export (F-108). *Erledigt bis auf Absturz und F-108: Nachtrag 2.*

## Nachtrag 2 (19.11.2026): Schließen mit Rückfrage an Gio, Signale, gespeicherter Stand, Konflikt (F-107, F-109, F-110)

- **Schließen vor dem Zerstören:** Gio liefert vor dem Zerstören ein abbrechbares `app.ClosingEvent`
  (`third_party/gio/app/system.go`; unter Wayland aus `xdg_toplevel.close` und `Perform(ActionClose)`,
  `app/os_wayland.go` `requestClose`). Der Editor speichert dort; die Entscheidung trifft `canvas.Sitzung.Schliessen`
  (testbar ohne Gio). Scheitert das Speichern — auch das Ausweichen auf eine neue Aufnahme —, bleibt das Fenster offen
  und die Statusleiste sagt „Speichern gescheitert: … Erneutes Schließen verwirft die Änderungen.“ Erst eine zweite
  Anfrage ohne neue Änderung schließt und verwirft (Claudes Entscheidung, von Andi am 20.11.2026 bestätigt: ohne Dialog die einzige
  Möglichkeit, den Editor bei voller Platte überhaupt zu verlassen). Weicht Speichern mit Warnung aus (F-116, F-110),
  bleibt das Fenster einmal offen, damit die Warnung gelesen wird; die nächste Anfrage schließt. Ein gewöhnlicher
  Hinweis ohne Warnung geht wie bisher auf die Fehlerausgabe.
- **Signale:** SIGTERM, SIGINT und SIGHUP lösen ein geordnetes Speichern und Schließen aus, ohne das Fenster offen zu
  halten; die Signal-Goroutine weckt nur das Fenster (`Window.Invalidate`), gespeichert wird in der Ereignisschleife
  (F-77). *Geändert 20.11.2026 (F-152, Andis Entscheidung):* Weitere Signale (SIGTERM, SIGINT, SIGHUP) werden
  abgefangen, bis der Editor zurückkehrt — ein zweites Strg+C oder SIGHUP nach SIGTERM beim Abmelden bricht ein
  laufendes Speichern (bis rund 45 s, Nachtrag 3) nicht mehr ab; danach beendet sich der Editor geordnet. Erst nach der
  Rückkehr gilt wieder die Voreinstellung. SIGKILL bleibt unabfangbar. Die Wache (`canvas.Signalwache`) ist ohne echte
  Signale getestet (eigener Kanal). Kommt das Signal, während das Fenster nach einem gescheiterten Speichern mit der
  Fehlermeldung offen steht, speichert der Editor noch einmal (samt Ausweichen auf eine neue Aufnahme) — das Verwerfen
  bestätigt nur eine zweite Schließen-Anfrage, kein Signal — und schließt danach in jedem Fall, statt bis zum SIGKILL
  offen zu hängen; scheitert es wieder, bleiben Fehlerausgabe und Rückgabewert, beim Zerstören folgt kein weiterer
  Versuch (`canvas.Sitzung.SchliessenNachSignal`).
- **Ohne Anfrage zerstört** (Wayland-Verbindung weg): Speichern beim `DestroyEvent` bleibt (Nachtrag 1); dort gibt es
  nur noch die Fehlerausgabe und den Rückgabewert.
- **Kein Autospeichern, kein Wiederherstellungsort:** Gegen Absturz oder SIGKILL schützt das nicht. Bewusst so, weil
  ein Zwischenspeichern bei großen Dokumenten bis zu Minuten rendert (Nachtrag 1) und die Bibliothek mit Zwischenständen
  füllte; Gegenstand einer eigenen Karte, falls Abstürze auftreten.
- **Gespeicherter Stand (F-109):** Der Verlauf vergibt je Änderung eine nie wiederkehrende Kennung
  (`document.Dokument.Stand`); „geändert“ heißt aktuelle Kennung ≠ Kennung beim letzten Speichern. Rückgängig bis zum
  gespeicherten Stand ist ungeändert; Strg+S ohne Änderung schreibt nicht (Statusleiste „Keine Änderungen zu
  speichern.“). Folge: Eine Datei von außen landet erst mit der ersten Änderung in der Bibliothek (von Andi am 20.11.2026
  zur Kenntnis genommen).
- **Konflikt zweier Editoren (F-110):** Beim Öffnen merkt sich der Editor den Stand der Datei (Gerät, Inode, Größe,
  Änderungszeit; `library.Stand`, gelesen vor dem Inhalt). `library.Speichern` vergleicht ihn vor dem Lesen und noch
  einmal kurz vor dem Umbenennen und überschreibt bei Abweichung nicht (`ErrGeaendert`); der Editor weicht auf eine
  neue Aufnahme aus und warnt („Die Datei wurde seit dem Öffnen anderswo geändert; …“). Race-arm, nicht race-frei
  (Millisekunden zwischen Vergleich und Umbenennen), ohne Sperrdateien.

## Nachtrag 3 (20.11.2026): Schnelleres Füllen, kleineres Budget, Warnung beim Speichern (F-130, F-138, F-140)

- **Füllen (F-130):** `x/image/vector` zeichnet auf `*image.NRGBA` nur über seinen allgemeinen Weg (rund 36 ns je
  Pixel). Der Renderer lässt sich die Deckung jetzt über `draw.Src` mit deckender Quelle in ein eigenes Ziel
  (`render.einmischer`) reichen und mischt dort mit derselben Formel wie `draw.Over` — bitgleich (Test gegen den alten
  Weg, auch auf durchsichtigem Grund), rund 7 ns je Pixel; der Rasterizer wird je Bild wiederverwendet. Der schnelle
  Weg von `vector` über `*image.Alpha` scheidet aus: Er rundet die Deckung auf 8 Bit, Pixel wichen um bis zu 4 ab.
  Gemessen (Entwicklungsrechner, ein Kern): zwei gefüllte, umrandete Kästen auf 8000×8000 7,8 s → 1,9 s, ein
  diagonaler Pfeil über 8000×8000 2,4 s → 0,55 s, das Alltagsdokument aus Nachtrag 1 48 s → 11 s.
- **Budget:** `fuellGewicht` 100 → 25, `render.Budget` 8e11 → 1,25e11. Das Alltagsdokument liegt gut 3-fach darunter;
  ein Dokument knapp unter dem Budget (39 gefüllte, umrandete Kästen auf 8000×8000) rastert in rund 35 s statt knapp
  5 min, mit Glyphen und Kanten an ihren Grenzen höchstens rund 45 s. Für 30 s müsste das Budget auf rund 1e11 sinken,
  dann läge das Alltagsdokument nur 2,5-fach darunter — der Abstand zu echter Arbeit geht vor (F-131).
- **Speichern über dem Budget (F-138):** Speichern bricht weiter nie ab. Liegt das gespeicherte Dokument über dem
  Budget, warnt der Editor („Gespeichert, aber die Datei hat zu viele oder zu große Beschriftungen, um sie wieder zu
  öffnen. Vor dem Schließen einige entfernen.“); die Warnung hält das Fenster beim Schließen einmal offen (Nachtrag 2).
  Eigene Dateien beim Öffnen auszunehmen, scheidet aus: Eine Arbeitsdatei der Bibliothek ist von einer präparierten
  nicht zu unterscheiden. Ein Öffnen trotz Überschreitung (Rückfrage oder Schalter) bleibt offen.
- **Messungen (F-140):** `BenchmarkAufwand…` in `render/render_test.go` (Alltag, Füllen, knapp unter den drei
  Grenzen) laufen nur mit `-bench`.

## Nachtrag 4 (20.11.2026): Arbeitsziel nach dem Speichern einer Datei von außen (F-155)

- **Neues Arbeitsziel:** Das erste Speichern einer Datei außerhalb der Bibliothek legt eine neue Aufnahme an (die
  Ausgangsdatei bleibt unberührt, ADR-0016). Diese Aufnahme ist danach das Arbeitsziel: Die Titelleiste nennt ihren
  Namen, jedes weitere Strg+S und das Speichern beim Schließen schreiben in genau diese Datei, nie in eine dritte
  (das war schon so, jetzt per Test `TestSpeichernDateiVonAussenArbeitsziel` festgehalten).
- **Meldung:** „Saved as a new capture in the library: schnapp/JJJJ-MM/<Name>.png (the original file is unchanged)“ —
  Pfad relativ zum Bilderordner, ohne Benutzernamen. Wer die Ausgangsdatei erneut öffnet, findet die Zeichnung dort
  nicht; die Meldung sagt, wo sie liegt.
- **Kein Pfad auf stdout:** Die Kommandozeile `schnapp DATEI` gibt wie bisher nichts auf stdout aus (ADR-0017: dort
  steht nur der Pfad einer neuen Aufnahme vor dem Editor). Der Pfad der neuen Aufnahme entsteht erst beim Speichern,
  mitten in der Sitzung; eine Ausgabe beim Schließen wäre ein neues Verhalten ohne Bedarf. Die Meldung geht bei
  geschlossenem Fenster auf die Fehlerausgabe (Nachtrag 2).
