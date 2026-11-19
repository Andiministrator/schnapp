# Schnapp — Logo-Paket

Stand 19.11.2026. Gewählt von Andi: Entwurf **e6** aus Runde 6 (`../entwuerfe/runde-6/`). Regeln für den Einsatz
stehen im Corporate Design: [`../corporate-design.md`](../corporate-design.md) (PDF daneben).

> **Alle Dateien hier sind erzeugt.** Quelle ist `../werkzeug/paket.py`; nie eine Datei von Hand ändern, sondern
> das Skript anpassen und neu bauen:
>
> ```bash
> python3 brand/werkzeug/paket.py
> ```
>
> Braucht Python mit `fontTools`, die Schrift Open Sans SemiBold (Pfad über `SCHNAPP_SCHRIFT`, Vorgabe
> `/usr/share/fonts/TTF/OpenSans-SemiBold.ttf`), `rsvg-convert` und ImageMagick (`magick`).

## Das Logo

- **Schriftzug „schnapp“**, selbst gezeichnet: Monoline mit runden Enden, Bögen als abgerundete Rechtecke.
  Das **h** sitzt tiefer, die **pp** höher — beide ragen oben und unten gleich weit aus dem Band der kleinen
  Buchstaben.
- **Das „s“** ist orange und trägt ein Auge, dessen Pupille nach rechts auf „chnapp“ blickt.
- **Unterzeile** „SCREENSHOTS“ unter „sc“ und „FOR LINUX“ unter „na“, in Open Sans SemiBold, als Pfade gesetzt
  (das SVG braucht keine Schriftdatei).

## Fassungen

**dunkel** = für dunklen Grund (helle Schrift). **hell** = für hellen Grund (dunkle Schrift). Ohne Zusatz ist der
Grund durchsichtig; **mit-grund** hat die passende Fläche dahinter (Dunkelgrau `#1E2124` bzw. Weiß).

| Datei | Inhalt |
|---|---|
| `svg/schnapp-logo-{dunkel,hell}.svg` | Logo mit Unterzeile, Grund durchsichtig |
| `svg/schnapp-logo-{dunkel,hell}-mit-grund.svg` | dasselbe mit Fläche |
| `svg/schnapp-logo-ohne-unterzeile-{dunkel,hell}[-mit-grund].svg` | Logo ohne Unterzeile (klein, Fensterleisten, Kopfzeilen) |
| `svg/schnapp-logo[-ohne-unterzeile]-einfarbig-{schwarz,weiss}.svg` | einfarbig für Druck, Prägung, Stempel; das Auge bleibt als Ring mit Pupille |
| `svg/schnapp-symbol-{dunkel,hell}.svg` | App-Symbol: das „s“ auf abgerundeter Kachel (dunkelgrau bzw. hellgrau) |
| `svg/schnapp-s-{dunkel,hell}.svg` | nur das „s“ mit Auge, freigestellt |
| `png/schnapp-logo-…-{300,600,1200,2400}.png` | alle farbigen Logo-Fassungen als PNG, Breite in Pixeln (einfarbig nur 1200) |
| `png/schnapp-symbol-{dunkel,hell}-{16…1024}.png` | App-Symbol in 16, 22, 24, 32, 48, 64, 128, 180, 192, 256, 512, 1024 px |
| `png/schnapp-s-{dunkel,hell}-{64,256,512}.png` | freigestelltes „s“ |
| `favicon/favicon.ico` | 16, 32, 48 px in einer Datei |
| `favicon/favicon.svg`, `favicon-{16,32,48}.png` | Favicon als SVG und einzeln |
| `favicon/apple-touch-icon.png` | 180 px |
| `favicon/icon-{192,512}.png`, `site.webmanifest` | für Web-App-Manifest |
| `sozial/schnapp-sozial-1280x640.{svg,png}` | Vorschaubild für GitHub („Social preview“) und geteilte Links |

## Welche Datei wofür?

| Einsatz | Datei |
|---|---|
| README, Webseite (dunkles oder helles Thema) | `svg/schnapp-logo-dunkel.svg` bzw. `-hell.svg` |
| Dokumente, Folien | `png/schnapp-logo-hell-mit-grund-1200.png` oder SVG |
| App-Symbol unter Linux (hicolor) | `svg/schnapp-symbol-dunkel.svg` (scalable) und PNGs 16–512 — ohne Auswahlrahmen (Andi, 19.11.2026) |
| Webseite | `favicon/` komplett |
| GitHub → Settings → Social preview | `sozial/schnapp-sozial-1280x640.png` |
| klein (unter 320 px Breite) | `schnapp-logo-ohne-unterzeile-…`; unter 100 px nur das Symbol |

## Lizenz

Logo und Symbol stehen vorerst wie das ganze Repository unter GPL-3.0-or-later; dazu kommt eine kurze
Markenregel (E-12, entschieden: frei nutzbar, um auf Schnapp zu verweisen; Abwandlungen nicht als Schnapp ausgeben;
der Wortlaut folgt vor der ersten Paket-Veröffentlichung). Die Unterzeile verwendet Umrisse der Schrift Open Sans
(SIL Open Font License 1.1); als Grafik eingebettete Umrisse sind damit erlaubt.
