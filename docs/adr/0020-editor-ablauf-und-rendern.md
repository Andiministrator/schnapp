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
  (testbar ohne Gio). *Seit Nachtrag 6 in zwei Schritten: `SchliessenAnfrage` und `SpeichernEnde`; die synchrone Hülle
  `Schliessen` ist seit F-284 entfernt, Tests fahren die zwei Schritte über den Helfer `schliesse`.* Scheitert das Speichern — auch das Ausweichen auf eine neue Aufnahme —, bleibt das Fenster offen
  und die Statusleiste sagt „Speichern gescheitert: … Erneutes Schließen verwirft die Änderungen.“ Erst eine zweite
  Anfrage ohne neue Änderung schließt und verwirft (Claudes Entscheidung, von Andi am 20.11.2026 bestätigt: ohne Dialog die einzige
  Möglichkeit, den Editor bei voller Platte überhaupt zu verlassen). Weicht Speichern mit Warnung aus (F-116, F-110),
  bleibt das Fenster einmal offen, damit die Warnung gelesen wird; die nächste Anfrage schließt. Ein gewöhnlicher
  Hinweis ohne Warnung geht wie bisher auf die Fehlerausgabe.
- **Signale:** SIGTERM, SIGINT und SIGHUP lösen ein geordnetes Speichern und Schließen aus, ohne das Fenster offen zu
  halten; die Signal-Goroutine weckt nur das Fenster (`Window.Invalidate`), gespeichert wird in der Ereignisschleife
  (F-77). *Seit Nachtrag 6 in einer eigenen Goroutine; in der Schleife wird nur die Antwort angenommen.* *Geändert 20.11.2026 (F-152, Andis Entscheidung):* Weitere Signale (SIGTERM, SIGINT, SIGHUP) werden
  abgefangen, bis der Editor zurückkehrt — ein zweites Strg+C oder SIGHUP nach SIGTERM beim Abmelden bricht ein
  laufendes Speichern (bis rund 45 s, Nachtrag 3) nicht mehr ab; danach beendet sich der Editor geordnet. *Seit Nachtrag 9 (umgesetzt 21.11.2026) gilt das für jedes
  Speichern, das Fortschritt meldet; nur ein Speichern, das 15 Sekunden lang keinen Fortschritt meldet, gibt ein
  weiteres Signal auf (Nachtrag 7).* Erst nach der
  Rückkehr gilt wieder die Voreinstellung. SIGKILL bleibt unabfangbar. Die Wache (`canvas.Signalwache`) ist ohne echte
  Signale getestet (eigener Kanal). Kommt das Signal, während das Fenster nach einem gescheiterten Speichern mit der
  Fehlermeldung offen steht, speichert der Editor noch einmal (samt Ausweichen auf eine neue Aufnahme) — das Verwerfen
  bestätigt nur eine zweite Schließen-Anfrage, kein Signal — und schließt danach in jedem Fall, statt bis zum SIGKILL
  offen zu hängen; scheitert es wieder, bleiben Fehlerausgabe und Rückgabewert, beim Zerstören folgt kein weiterer
  Versuch (`canvas.Sitzung.SignalAnfrage` und `SpeichernEnde`; die frühere Hülle `SchliessenNachSignal` ist seit F-284
  entfernt).
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
  5 min, mit Glyphen und Kanten an ihren Grenzen höchstens rund 45 s. *Seit ADR-0028 rund 7 s bzw. zusammen rund 10 s
  (gemessen vorher 41 s); Budget und Fristen bleiben.* Für 30 s müsste das Budget auf rund 1e11 sinken,
  dann läge das Alltagsdokument nur 2,5-fach darunter — der Abstand zu echter Arbeit geht vor (F-131).
- **Speichern über dem Budget (F-138):** Speichern bricht weiter nie ab. Liegt das gespeicherte Dokument über dem
  Budget, warnt der Editor („Gespeichert, aber die Datei hat zu viele oder zu große Beschriftungen, um sie wieder zu
  öffnen. Vor dem Schließen einige entfernen.“); die Warnung hält das Fenster beim Schließen einmal offen (Nachtrag 2).
  Eigene Dateien beim Öffnen auszunehmen, scheidet aus: Eine Arbeitsdatei der Bibliothek ist von einer präparierten
  nicht zu unterscheiden. Ein Öffnen trotz Überschreitung (Rückfrage oder Schalter) bleibt offen. *Erledigt: `--ignore-limit`, ADR-0017
  Nachtrag 6 (F-146).*
- **Messungen (F-140):** `BenchmarkAufwand…` in `render/render_test.go` (Alltag, Füllen, knapp unter den drei
  Grenzen) laufen nur mit `-bench`.
- **Ergänzung 22.11.2026 (F-418):** `Pruefe` zählt nur so weit, bis eine Grenze überschritten ist; das Urteil bleibt
  gleich, die genannte Grenze kann eine andere sein. `Aufwand` und `Bild` zählen vollständig. Nachmessung: Kanten knapp
  unter `KantenBudget` (120 Abriss-Nähte auf 32768×2100) rastern in rund 8 s (rund 0,5 ns je Kanten-Einheit), die
  Summe aller Grenzen also rund 15 s statt 10 s; die Gewichte bleiben vorerst (F-429). *Nachmessung 23.11.2026 (ADR-0023
  Nachtrag 9): 120 Abriss-Nähte jetzt rund 6,5 s, die Summe aller Grenzen rund 14 s; das Zeitversprechen lautet rund 14 s.*

**Ergänzung (23.11.2026, Karte #221, F-576):** Die Kosten je Glyphe werden nicht je Schrift skaliert: Die dichtesten
Glyphen von Systemschriften (Noto Serif CJK) rastern 1,9- bis 2,6-mal langsamer als Go (Messung 23.11.2026, ein Kern,
4000×4000, 72 000 Glyphen bei Grad 8), der Glyphenanteil des Zeitversprechens wächst also höchstens um den Faktor 3 und
nur, wenn eine präparierte Datei dieselbe Systemschrift nutzt. Das Modell liegt bei kleinen Graden rund zehnfach unter
der Messung (rund 0,5 µs gegen rund 5 µs je Glyphe); das Budget bleibt Obergrenze, die Neueichung ist F-589.

**Ergänzung (23.11.2026, Karte #224, F-589): Neueichung der kleinen Glyphen.** Messung mit `BenchmarkGlyphKosten`
(`internal/render/glyphkosten_test.go`; ein Kern, ohne `-race`, 4000×4000, Go-Schrift, Zeit des leeren Bildes
abgezogen, 5 Läufe je Grad):

| Grad | µs je Glyphe | Einheiten je Glyphe vorher (1700) | ns je Einheit vorher | Einheiten nachher (5500) | ns je Einheit nachher |
|---|---|---|---|---|---|
| 4 | 3,6 | 1700 | 2,1 | 5500 | 0,66 |
| 8 | 4,0 | 1700 | 2,4 | 5500 | 0,73 |
| 12 | 5,2 | 1700 | 3,1 | 5500 | 0,95 |
| 16 | 5,5 | 1700 | 3,2 | 5500 | 0,99 |
| 24 | 5,3 | 1700 | 3,1 | 5500 | 0,96 |
| 32 | 8,0 | 1700 | 4,7 | 5500 | 1,45 |
| 48 | 8,9 | 2304 | 3,9 | 5500 | 1,61 |
| 64 | 9,9 | 4096 | 2,4 | 5500 | 1,80 |
| 128 | 17,3 | 16384 | 1,1 | 16384 | 1,06 |
| 256 | 63,7 | 65536 | 0,97 | 65536 | 0,97 |

Große Glyphen („W“ Grad 4096) rastern mit rund 1,1 ns je Einheit; kleine lagen mit 1700 Einheiten bis zum Vierfachen
darüber (die frühere Angabe von 0,3 ns je Einheit stammt aus der Messung vom 19.11.2026 vor ADR-0028). Neu:
`glyphGrund` = 5500 (vorher 1700), damit kleine und große Glyphen rund 1 ns je Einheit kosten (höchstens 1,8 ns bei
Grad 64, 0,7 ns bei Grad 4). Die Form max(Grad², `glyphGrund`) bleibt; sie bildet die Mitte (Grad 32 bis 64, wo die Zeit
schon über dem Grundwert liegt) nur genähert ab. `GlyphBudget` steigt von 1e9 auf 3e9, weil das Alltagsdokument
(10 200 Glyphen, 5,6e7 Einheiten) den Abstand 50 zur Grenze behalten soll (`TestAlltagsdokumentWeitUnterDemBudget`):
Der Abstand ist die Eigenschaft, die echte Bearbeitungen schützt; bei einem Abstand von 50 liegt die Wartezeit der
Glyphen an der Grenze bei rund 5 s, unabhängig davon, wie die Einheiten verteilt sind (vorher hatten die kleinen Glyphen
mit rund 588 000 Stück zu 5 bis 8 µs ebenfalls bis zu 4,7 s gebraucht, das Modell hatte es nur nicht gezählt). Gemessen
am Budget (ein Kern): 177 × „W“ Grad 4096 auf 8000×8000 (2,97e9) in 3,4 s; Grad 64 mit rund 520 000 Glyphen übereinander
(2,9e9) tragen zu einem Dokument knapp unter allen drei Grenzen (90 Abriss-Nähte auf 32768×2100, Rahmen bis Budget) in
insgesamt 16,3 s bei (`BenchmarkNaehteAlleGrenzenKleineGlyphen`); mit den „W“ sind es 11,7 bis 13,8 s
(`BenchmarkNaehteAlleGrenzen`). **Das Zeitversprechen lautet damit rund 16 s** (vorher rund 14 s, gemessen mit dem
billigeren Glyphenmuster). Ein Text trägt höchstens 10 000 Zeichen; die Messung verteilt sie auf mehrere Objekte.

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
- **Ablageort beim Schließen (F-158; Claudes Entscheidung, Andi informiert):** Wer eine Datei von außen ohne Strg+S
  schließt, bekommt dieselbe Meldung wie bei Strg+S: Beim Schließen wird wie bisher als neue Aufnahme abgelegt, und der
  Hinweis mit dem Ablageort hält das Fenster **einmal** offen — wie ein Ausweichen mit Warnung (Nachtrag 2); die
  zweite Schließen-Anfrage schließt. Dafür hält jeder Hinweis das Fenster einmal offen (`ui.Gespeichert.HaeltOffen`:
  Warnung oder nicht leerer Hinweis), die Entscheidung selbst bleibt in canvas (heute `SchliessenAnfrage` und `SpeichernEnde`). Die Ausgabe auf der
  Fehlerausgabe bleibt zusätzlich bestehen (Start aus Terminal, Fenster dennoch zu). Ein weiteres Strg+S oder Schließen
  danach hat keinen Hinweis mehr und schließt sofort.

## Nachtrag 5 (20.11.2026): Verborgene Texte im Budget, Tinte zwischengespeichert (F-224)

**Status:** Akzeptiert (Claude Opus 5.5, Nacharbeit zum Kritiker-Lauf Sprint-Abschluss 6, Karte #98).

- **Befund:** Ob ein Text in die Ansicht reicht, entscheidet seine vermessene Tinte (`document.Reichweite`, F-181).
  Editor, Renderer, Trefferprüfung und `render.Pruefe` fragen das für jedes Objekt bei jedem Bild ab; ganz verborgene
  Texte rastert niemand, also zählte das Kostenmodell sie nicht. Eine präparierte Datei mit 400 Texten à 10 000 Zeichen
  außerhalb der Ansicht hatte Aufwand 0 und kostete rund 0,2 s je Durchlauf, an der Grenze der Dateigröße Sekunden je
  Bild und Klick.
- **Zwischenspeicher:** `document` merkt sich die Tinte je Zeile für Inhalt und Schriftgrad (die Lage relativ zum Ort
  hängt nur davon ab); geleert wird er, sobald die Inhalte zusammen 64 MiB übersteigen. Jedes weitere Bild kostet
  damit nur noch die Rechnung je Zeile. Die Werte sind bitgleich zur bisherigen Rechnung (Sollwert-Tests unverändert).
- **ZeichenBudget:** `render.Pruefe` zählt vor dem Trockenlauf die Zeichen (Runen) aller Texte, sichtbar oder nicht,
  und lehnt über `ZeichenBudget` = 1 000 000 ab, ohne zu vermessen (unter 1 ms). Am Budget dauert das erste Vermessen
  rund 45 ms (Entwicklungsrechner). Ein echtes Dokument eines Bildschirmfotos kommt nicht in die Nähe; über dem Budget
  gilt wie für die anderen Grenzen: Öffnen lehnt ab (`--ignore-limit` überspringt es), Speichern warnt nur (F-138).

## Nachtrag 6 (20.11.2026): Editor nach der Aufnahme, Speichern und Kopieren im Hintergrund (F-112, F-143, F-178)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #62; Claudes Entscheidung im Rahmen des freigegebenen Sprints 8).

- **Editor nach der Aufnahme (F-112):** Der Editor öffnet erst, wenn der Aufnahme-Aufruf zurückgekehrt ist: Die
  Verbindung zu KWin ist zu, die Bilder aller Monitore hängen an nichts mehr. Er bekommt das abgelegte Dokument (eine
  eigene Kopie des gewählten Bereichs), statt die eben geschriebene Datei neu zu lesen. Der Pfad steht wie bisher vor
  dem Editor auf stdout.
- **Speichern im Hintergrund (F-143, Rest F-112):** Strg+S, Schließen und Signal speichern einen **Abzug** des
  Dokuments (`document.Dokument.Abzug`, unabhängige Kopie mit gleichem Stand) in einer eigenen Goroutine. Die
  Ereignisschleife läuft weiter: Das Fenster zeichnet, beantwortet die Pings des Compositors (kein „Beenden
  erzwingen“) und zeigt „Speichern …“ in der Statusleiste. Die Antwort kommt wie beim Dateidialog über einen Kanal und
  den Wecker in die Schleife (F-77, F-231); erst dort gilt der Stand als gespeichert. Es läuft höchstens ein Speichern
  zugleich (die Speicherfunktion der Bibliothek führt Zustand: Konflikt, Ausweichen, liegen gebliebene Fassungen).
- **Bearbeiten ruht während des Speicherns:** Eingaben auf Leinwand, Leiste, Menü und Tastatur wirken nicht, bis das
  Speichern zurück ist; Zeichnen und Schließen-Anfragen gehen weiter. Grund ist ADR-0022 Punkt 3: Nach einem
  Speichern mit Schwärzung wird der Verlauf geleert. Liefe daneben eine Bearbeitung (ein Zug, ein Rückgängig), gälte
  das Leeren für einen anderen Stand als den gespeicherten, oder es fiele aus — ein Rückgängig könnte das
  Geschwärzte zurückholen. Gespeichert dauert ein Alltagsdokument Bruchteile einer Sekunde, ein Dokument am Budget
  Sekunden; die Pause ist der kleinere Preis.
- **Schließen während des Speicherns:** Die Anfrage wird angehalten und nach dem Speichern so entschieden, als wäre sie
  danach gekommen (Nachtrag 2, Nachtrag 4): gelungen ohne Hinweis → zu; mit Hinweis → einmal offen; gescheitert →
  offen mit Fehler, die nächste Anfrage verwirft. Ein **Signal** während eines Strg+S wartet ebenso; ist jenes
  gescheitert, versucht das Signal ein weiteres Speichern (wie bisher nach einem gescheiterten Schließen) und schließt
  danach in jedem Fall. Endet das Fenster ohne Anfrage (Wayland weg), wartet der Editor auf ein laufendes Speichern und
  speichert, was dann noch ungesichert ist, wie bisher im Vordergrund.
- **Kopieren (F-178):** Strg+C rendert und kodiert den Abzug ebenfalls in einer Goroutine und übergibt dort an den
  Halter der Zwischenablage (ADR-0024); nur der Rückfall auf das Fenster selbst (ADR-0015 Nachtrag 2) läuft danach in
  der Ereignisschleife. Kopieren ändert nichts am Dokument und sperrt das Bearbeiten nicht; ein zweites Strg+C während
  eines laufenden wartet nicht, sondern wird mit „Kopieren läuft schon“ beantwortet. *Ergänzt 20.11.2026 (F-285, Claudes
  Entscheidung, Karte #62):* **Strg+C wirkt auch während eines Speicherns** — als einzige Taste. Kopieren ändert nichts
  am Dokument, und während des Speicherns läuft kein Zug, der Abzug gelingt also; es still zu verwerfen oder nur zu
  melden, hieße nach Strg+S bei einem großen Bild warten zu müssen, ohne dass es etwas schützt. Die Statusleiste sagt
  beim Start „Kopieren …“. Die Meldung am Ende ist an den Stand des Abzugs gebunden: Wurde inzwischen bearbeitet, sagt
  sie „Kopiert mit dem Stand beim Kopieren; spätere Änderungen sind nicht enthalten.“, statt den Eindruck zu wecken,
  das jetzt Sichtbare (etwa eine neue Schwärzung) liege in der Zwischenablage. Schließen und Zerstören warten auf ein
  laufendes Kopieren (F-281, Nachtrag 7).
- **Sperre auch in der Sitzung (F-280, ergänzt 20.11.2026):** Die Sperre liegt nicht nur in der Oberfläche: Jeder
  Einstieg von `canvas.Sitzung`, der das Dokument ändert (Züge, Rückgängig/Wiederholen, Löschen, Farbe, Werkzeug,
  Texteingabe, Esc, Enter, Doppelklick, Ganzes Bild, Nähte), tut während des Speicherns nichts. Die Oberfläche fragt je
  Klick neu, denn mehrere Klicks können in einem Bild liegen (Speichern und Farbe). `SpeichernEnde` leert nach einem
  gelungenen Speichern mit Schwärzung den Verlauf ohne Bedingung an den Stand: Hätte sich das Dokument doch geändert,
  holte ein Rückgängig sonst das Geschwärzte zurück (ADR-0022 Punkt 3); der Verlust des Verlaufs ist der kleinere
  Schaden.

## Nachtrag 7 (20.11.2026): Fristen beim Schließen — hängendes Speichern, laufendes Kopieren (F-281, F-283)

**Status:** Akzeptiert (Claude Opus 5.5, Karte #62; Claudes Entscheidung im Rahmen des freigegebenen Sprints 8).

- **Problem (F-283):** Antwortet das Speichern im Hintergrund nie (hängender Netzordner, blockierte Platte), hielt jede
  Schließen-Anfrage das Fenster offen, ein Signal wurde nur vorgemerkt, und das Warten beim Zerstören hatte keine
  Frist. Weil das Fenster weiter die Pings des Compositors beantwortet, bietet KWin kein „Beenden erzwingen“ an; ein
  Signal fängt die Wache ab (Nachtrag 2). Übrig blieb nur SIGKILL.
- **Entscheidung:** Antwortet ein Speichern **15 Sekunden** nach seinem Beginn nicht, sagt die Statusleiste als Fehler
  „Speichern antwortet nicht. Schließen beendet jetzt ohne zu warten; Änderungen gehen womöglich verloren.“ Danach
  schließt die **nächste Schließen-Anfrage** oder ein **weiteres Signal** das Fenster ohne die Antwort. Vor der Frist
  bleibt es wie bisher (Anfrage angehalten, weitere Signale abgefangen) — ein Alltagsdokument braucht Bruchteile einer
  Sekunde, und wer die Meldung nicht sieht, wird nicht überrascht. Der Editor kehrt mit `ui.ErrSpeichernHaengt` zurück:
  Die Fehlerausgabe sagt „Speichern hat nicht geantwortet; die Änderungen sind womöglich nicht gesichert.“, der
  Exit-Code ist ein Fehler. Beim Zerstören wird dann nicht gewartet und **nicht noch einmal im Vordergrund
  gespeichert**: Die hängende Goroutine hält die Speicherfunktion, und die darf nicht zweimal zugleich laufen
  (Nachtrag 6). *Eingeschränkt durch Nachtrag 9 (umgesetzt 21.11.2026): Die 15 Sekunden zählen ab dem letzten
  Fortschritt, nicht ab dem Beginn; ein Speichern mit Fortschritt gilt nie als hängend.*
- **Zerstören ohne Anfrage** (Wayland weg) wartet höchstens **2 Minuten** auf ein laufendes Speichern — länger als ein
  Dokument am Budget braucht (rund 45 s, Nachtrag 3), damit ein langsames, aber gesundes Speichern nicht verloren geht;
  ein weiteres Signal nach 15 Sekunden ohne Fortschritt beendet das Warten ebenso (Nachtrag 9). Ein weiterer Versuch nach einem Signal läuft
  hier jetzt ebenfalls in der Goroutine, mit derselben Frist.
- **Kopieren (F-281):** Strg+C und gleich schließen verlor die Kopie: Der Prozess endete, bevor die Goroutine den Halter
  der Zwischenablage gestartet hatte. Jetzt hält eine Schließen-Anfrage, die schließen dürfte, das Fenster offen, bis
  die Antwort des Kopierens angenommen ist, und fragt dann von selbst erneut; Zerstören (auch nach Signal oder
  Speichern) wartet ebenso. Beides höchstens **10 Sekunden** ab Beginn des Kopierens (ein 4K-Bild braucht rund eine
  Sekunde); danach geht es weiter, und die Fehlerausgabe sagt „Kopieren war beim Schließen des Fensters nicht fertig;
  die Zwischenablage ist womöglich leer.“ Dasselbe steht dort, wenn das Fenster selbst hätte anbieten müssen (kein
  Halter möglich, ADR-0015 Nachtrag 2): Mit dem Fenster geht dieser Inhalt verloren.
- **Fristen als Paketvariablen** (`speichernFrist`, `zerstoerenFrist`, `kopierFrist` in `ui/editor.go`), damit Tests
  sie verkürzen; Test mit einer Speicherfunktion, die nie zurückkehrt.

## Nachtrag 8 (20.11.2026): Schließen per Taste und Menü, Exit-Code, Fehler beim Zerstören (F-299, F-300, F-302)

**Status:** Akzeptiert (Claude Opus 5.5, Nacharbeit des Kritiker-Laufs Sprint-Abschluss 8, Karte #120). Präzisiert
Nachtrag 6 und 7; die 15-s-Frist zählt seit Nachtrag 9 ab dem letzten Fortschritt (F-298, Karte #119).

- **Schließen-Anfragen gehen weiter (Nachtrag 6) gilt für jeden Weg:** Fensterknopf, Strg+W und der Menüeintrag
  „Schließen“ (Klick und Enter), auch während des Speicherns (F-299). `SchliessenAnfrage` merkt die Anfrage vor. Alles
  Ändernde bleibt gesperrt; während des Speicherns wirken nur Strg+C und Schließen. Das Menü lässt sich während des
  Speicherns nicht neu öffnen; der Eintrag „Schließen“ wirkt also nur, wenn das Menü schon offen war. Der Weg per
  Tastatur ist Strg+W (Probelauf `knowledge/messungen/f-299-probe.md`, Rückfall-Drehbuch
  `scripts/drehbuecher/rueckfall-schliessen-waehrend-speichern.txt`).
- **„Der Exit-Code ist ein Fehler“ (Nachtrag 7) gilt, wenn Änderungen ungesichert bleiben:** `ui.ErrSpeichernHaengt`
  und ein gescheitertes Speichern, das der Nutzer nicht verworfen hat — auf dem Weg `schnapp DATEI` und nach einer
  Aufnahme (F-302). Auf dem DATEI-Weg heißt die Meldung dann „Der Editor ließ sich nicht öffnen oder nicht speichern“, nicht
  „… lässt sich nicht öffnen“. Ein Fensterfehler allein (kein Wayland usw.) bleibt nach einer Aufnahme eine Warnung mit
  Exit 0, weil die Aufnahme abgelegt ist — ein Fehler-Code ließe ein Skript die Aufnahme wiederholen (wie bei F-180).
- **Fehler beim Zerstören:** Scheitert das Speichern einer Schließen-Anfrage und wird das Fenster zerstört, bevor ein
  Bild den Fehler gezeigt hat, geht der Fehler in die Rückgabe des Editors (F-300). Hat ein Bild ihn gezeigt, verwirft
  die nächste Anfrage wie bisher.

## Nachtrag 9 (21.11.2026): Ein arbeitendes Speichern wird nie aufgegeben (F-298)

**Status:** Akzeptiert (Andi, 21.11.2026: Variante b). Umgesetzt 21.11.2026 (Karte #119, Claude Opus 5.5), siehe
„Umsetzung“ am Ende.

- **Entscheidung:** Die Frist aus Nachtrag 7 unterscheidet künftig ein hängendes von einem langen Speichern. Solange
  das Speichern Fortschritt meldet (etwa nach Rendern, Kodieren, Schreiben), gilt es nicht als hängend; ein zweites
  Signal oder eine weitere Schließen-Anfrage bricht es nicht ab. Das bestätigt F-152 und Nachtrag 2 (ein zweites Signal
  bricht ein laufendes Speichern nicht ab) und schränkt Nachtrag 7 auf ein Speichern ohne Fortschritt ein.
- **Verworfen:** Abbrechen nach festen 15 Sekunden auch bei Fortschritt (Variante a). Ein 4K-Bild braucht im
  Probe-Build (`-race`) 8–24 s, im Normalbau 0,4–1,5 s (F-297, `knowledge/messungen/f-297.md`), ein Dokument am Budget
  35–45 s (Nachtrag 3); der Abbruch träfe gesunde Speichervorgänge.
- **Offen für die Umsetzung:** Wie fein die Fortschrittsmarken sind, wie lange ohne Marke als hängend gilt, und dass
  ein nach dem Aufgeben doch gelungenes Speichern `ErrSpeichernHaengt` nicht stehen lässt (Nebenbefund in F-298).
- **Zerstören ohne Anfrage** (Wayland weg, Nachtrag 7, `zerstoerenFrist` 2 Minuten) — entschieden 21.11.2026 (Andi,
  folgt dem Vorschlag): Die 2-Minuten-Grenze bleibt als äußerste Grenze, auch bei Fortschritt. Ohne Compositor gibt es
  keinen anderen Ausweg, und ein Dokument am Budget braucht höchstens 45 s. „Nie aufgegeben“ meint das zweite Signal
  und die weitere Schließen-Anfrage.
- **Umsetzung (21.11.2026, Karte #119):**
  - **Fortschrittsquelle** ist `internal/speicherschritt` (*seit F-312, Sprint 10: `internal/saveprogress`*): `Marke` nach jedem benannten Schritt (gerendert,
    alte-datei-gelesen, bild-kodiert, original-kodiert, datei-geschrieben, ersetzt, index, vorschau, budget-geprueft;
    die Schnitte der Messung F-297) und `Puls` aus langen Schleifen — je Füllung und beim Füllen alle 64 Zeilen (rund 3 % Mehrkosten im Füll-Benchmark `AufwandZweiKaesten8000`, 1,87 statt 1,82 s; je Pixel waren es 7 %), je
    Textzeichen, je geschriebenem Stück der PNG-Kodierung (`Schreiber`), beim Lesen und Dekodieren der alten Datei
    (`Leser`), je MiB beim Schreiben der Datei, je Zeile der Vorschau. `Zuletzt` liefert den Zeitpunkt des letzten
    Fortschritts mit monotonem Anteil der Uhr (ein Sprung der Wanduhr, etwa nach dem Ruhezustand, täuscht keinen
    Stillstand vor).
  - **Hängend** heißt: Ein Speichern läuft und meldet seit `speichernFrist` (15 s) keinen Fortschritt, gerechnet ab dem
    letzten Fortschritt, frühestens ab seinem Beginn. Nur dann sagt die Statusleiste es, und nur dann gibt die nächste
    Schließen-Anfrage oder ein weiteres Signal es auf. Kommt danach wieder Fortschritt, nimmt die Statusleiste die
    Meldung zurück („Saving…“). Die Prüfung weckt die Schleife zur Frist und danach alle `speichernFrist/4`, solange das
    Speichern läuft. Beim Speichern eines 4K-Bildschirmfotos liegt der längste Abstand ohne Fortschritt im Normalbau bei
    rund 80 ms (Verify-Lauf, Dekodieren der alten Datei, vor dessen Puls); für größere zulässige Dokumente ist er nicht
    gemessen.
  - **Doch noch gelungen:** Kommt die Antwort eines aufgegebenen Speicherns vor der Rückkehr des Editors (noch in der
    Schleife oder beim Zerstören, dort ohne zu warten), zählt sie: Gelang es, gilt der Stand als gesichert und
    `ErrSpeichernHaengt` entfällt; scheiterte es, steht sein Fehler in der Rückgabe. War ein Signal vorgemerkt und
    scheiterte ein Strg+S oder Schließen, läuft wie sonst der Neuversuch nach dem Signal (Nachtrag 2) und wird mit den
    üblichen Fristen abgewartet. Ein Fehler, der schon vor dem Aufgeben in der Rückgabe stand, bleibt stehen.
  - **Grenzen:** Der Zeitpunkt gilt prozessweit; ein Kopieren oder Exportieren, das nebenher rendert, meldet ebenfalls
    Fortschritt und verzögert das Erkennen eines hängenden Speicherns höchstens um seine Dauer. `zerstoerenFrist`
    (2 Minuten ab Beginn des Speicherns) bleibt unabhängig vom Fortschritt die äußerste Grenze. `fsync` und Umbenennen
    melden keinen Fortschritt: Ein Medium, das dort 15 s steht, gilt als hängend — gewollt, auch wenn ein sehr
    langsames, aber arbeitendes Medium (Netzordner) so aussehen kann. *Seit Nachtrag 10 synchronisiert das Schreiben
    je 1-MiB-Stück und pulst danach; ohne Fortschritt bleiben nur der Schluss-Sync und das Umbenennen.*
  - Tests: `internal/ui/speichern_fortschritt_test.go` (Fortschritt nie aufgegeben, auch im Warten beim Zerstören; Meldung
    zurückgenommen; Schließen nur bei gemeldetem Stillstand; doch gelungen oder gescheitert; früherer Fehler bleibt;
    Neuversuch nach Signal; 2-Minuten-Grenze trotz Fortschritt; wiederholte Prüfung), `internal/render/fortschritt_test.go`,
    `internal/fileformat/schreiben_fortschritt_test.go`, `internal/library/speichern_fortschritt_test.go`,
    `internal/speicherschritt/schritt_test.go` (heute `internal/saveprogress`). Verify-Lauf Fable (21.11.2026): 8 von 20 Mutationen blieben zunächst
    grün. Nach der Nacharbeit (Puls je Füllung, monotone Zeit, Schreiben in Stücken, Leser beim Lesen und Dekodieren,
    Neuversuch nach später Fehlantwort, neue Tests) werden 25 von 26 Mutationen rot; die übrige (`zerstoerenFrist` ab
    dem letzten Fortschritt statt ab Beginn) galt als gleichwertig. Das stimmt nur, wenn das Fenster gleich nach dem
    Start des Speicherns zerstört wird; bei spätem Zerstören und bei jedem weiteren Signal ohne Stillstand verschöbe sie
    die 2-Minuten-Grenze. Der Kritiker-Lauf zum Abschluss von Sprint 9 hat das gezeigt (F-314); seither deckt
    `TestZerstoerenFristAbBeginnBeiSpaetemZerstoeren` die Mutation ab.

  Seit Sprint 10 heißt das Paket `internal/saveprogress` (E-23, F-312).

## Nachtrag 10 (21.11.2026): Puls nach abgeflossenen Daten, nicht nach dem Seitencache (F-313, Karte #127)

**Status:** Akzeptiert (Claude, Andi informiert).

- `fileformat.SchreibeArbeitsdatei` schreibt in 1-MiB-Stücken. Kann der Schreiber synchronisieren (die temporäre Datei
  der Bibliothek, unverpackt durchgereicht), folgt auf jedes Stück ein `Sync` und danach der Puls. Ohne `Sync` (Puffer,
  Attrappe) pulst jedes Stück beim Schreiben wie bisher.
- Auf einem Medium mit 1 MB/s zerfällt die Wartezeit eines 25-MB-Speicherns so in Abschnitte von rund 1 s, jeder unter
  der Frist von 15 s: Das Speichern gilt nicht als hängend, solange Daten abfließen. Ein `Sync`, der nicht zurückkehrt,
  meldet nichts und gilt weiter als hängend; einen Puls aus einem Zeitgeber gibt es nicht.
- Kosten (Normalbau, NVMe, Rauschbild 4K rund 25 MB): Schreiben und Ersetzen 47 → 73 ms, rund 1,7 % von 1,5 s; die
  Widerlegungsgrenze von 10 % ist nicht erreicht.
- Nicht abgedeckt: Medien unter etwa 70 KB/s, auf denen ein einzelnes Stück länger als 15 s braucht. Schluss-`Sync` und
  Umbenennen melden weiter nichts.

## Nachtrag 11 (21.11.2026): Bild und Original nebenläufig kodieren (F-310, Karte #125)

**Status:** Akzeptiert (Claude, Andi informiert).

- `fileformat.SchreibeArbeitsdatei` kodiert sichtbares Bild und Original in zwei Goroutinen, je mit eigenem Puffer.
  Sie kehrt erst zurück, wenn beide fertig sind, auch im Fehlerfall; ein Fehler des Bilds geht vor. `Puls` und
  `Marke` sind nebenläufig sicher. Die Ausgabe ist bitgleich zur Fassung nacheinander (Test).
- `library.Speichern` verkleinert die Vorschau nebenläufig und schreibt sie wie bisher erst nach dem Ersetzen.
- Die alte Datei wird weiter mit `LeseArbeitsdatei` voll gelesen: Das Urteil „beschädigt“ erfasst auch kaputte Pixel,
  und `Speichern` überschreibt eine solche Datei nicht. `LeseMetadaten` prüft die Pixel nicht und genügt dafür nicht.
- Messung 4K im Normalbau (`-benchtime 3x -count 3`): glatt 0,38 → 0,20 s, Rauschen 1,54 → 0,83 s. Die Reihenfolge der
  Marken ist nicht mehr festgelegt; ihre Teilzeiten sind Abstände zur vorigen Marke, belastbar ist die Gesamtzeit.
