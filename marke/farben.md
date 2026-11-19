# Farben Schnapp

Stand 19.11.2026, festgelegt mit dem Logo e6 (Grün dunkler und ruhiger, Orange heller und markanter — Andi; Rot für
Fehler ergänzt). Rollen in der Oberfläche: [`corporate-design.md`](corporate-design.md) Abschnitt 3. Eine Palette für Logo **und** Oberfläche: dunkelgrauer Grund, dezentes
Grün als ruhige Hauptfarbe, helleres Orange als markante Gegenfarbe. Gleiche Sättigungsfamilie, aber verschieden hell —
das hilft auch bei Rot-Grün-Schwäche.

## Palette

| Name | Hex | Einsatz |
|---|---|---|
| grau-950 | `#16181B` | tiefster Grund (Overlay-Abdunklung, Werkzeugleisten) |
| grau-900 | `#1E2124` | **Grund** (Fenster, Logo dunkel) |
| grau-800 | `#2A2E33` | Flächen darüber (Leisten, Karten) |
| grau-700 | `#3A3F45` | Linien, Ränder |
| grau-400 | `#8E949B` | Nebentext, deaktiviert |
| grau-100 | `#ECEAE5` | **Schrift** auf dunkel |
| gruen-300 | `#86C2A6` | Grün auf dunkel als Text/Hover |
| gruen-500 | `#4F9479` | **Hauptfarbe**: Auswahl, aktives Werkzeug, Hauptaktion („Speichern“) |
| gruen-700 | `#3B7A5F` | Grün auf hellem Grund (Logo hell) |
| orange-300 | `#F9C894` | Orange auf dunkel als Text/Hover |
| orange-500 | `#F5A65B` | **Akzent**: Logo-„s“, Tastaturfokus, Hinweise, Auswahlecken |
| orange-700 | `#A85F22` | Orange auf hellem Grund |
| rot-500 | `#E67873` | Fehler, Gefahr, unumkehrbare Aktion (5,62:1 auf grau-900, 4,75:1 auf grau-800) |
| rot-700 | `#B4433E` | Rot auf hellem Grund (5,51:1 auf Weiß) |

## Kontraste (WCAG, gerechnet)

| Vordergrund | Grund | Kontrast |
|---|---|---|
| grau-100 | grau-900 | 13,45 |
| grau-400 | grau-900 | 5,28 |
| gruen-500 | grau-900 | 4,50 |
| orange-500 | grau-900 | 8,08 |
| gruen-500 / orange-500 | grau-800 | 3,80 / 6,82 |
| grau-900 auf gruen-500 / orange-500 (Schrift auf Schaltfläche) | — | 4,50 / 8,08 |
| gruen-700 / orange-700 | weiß | 5,08 / 4,86 |

gruen-500 erreicht auf grau-900 genau AA (4,5:1); für **Text** in Grün auf dunklem Grund gruen-300 nehmen, gruen-500
für Flächen, Linien, Symbole (Grenze dort 3:1). Grün und Orange unterscheiden sich in der Helligkeit (≈ 1,8:1).

## Regeln

- Logo: höchstens Grün + Orange auf Grau (dunkel) bzw. gruen-700 + Orange auf Weiß (hell). Im hellen Logo sind Maul
  und Zähne `#E0893A` (nur Schmuck, kein Text); Text in Orange auf Weiß immer orange-700. Einfarbig muss es auch gehen.
- **Grün und Orange nie als einziges Unterscheidungsmerkmal** (für Rot-Grün-Schwäche trotzdem schwer zu trennen):
  immer Form, Symbol oder Text dazu.
- Beschriftungsfarben im Editor (Pfeile, Kästen) sind eine eigene Auswahl für Bilder, nicht diese Palette.
