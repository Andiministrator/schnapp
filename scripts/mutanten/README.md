# Mutationsprobe (F-371)

Prüft, ob die Tests eine gezielte Fehlerstelle im Produktcode bemerken: Je Mutant wird genau eine Stelle geändert, das
Paket getestet, und mindestens ein Test muss rot werden. Eigenes Go-Modul (nur Standardbibliothek), damit
`go build ./...` und `go test ./...` im Wurzelmodul es nicht einfangen.

## Aufruf

```bash
cd scripts/mutanten
go run . -wurzel ../..                   # alle Gruppen
go run . -wurzel ../.. -gruppe capture   # eine Gruppe (Dateiname ohne .json)
go run . -wurzel ../.. -nur "C1 NeuKWin ohne Präfix"
go run . -wurzel ../.. -trocken          # nur prüfen, dass jedes Muster genau einmal vorkommt
```

Exit 0 nur, wenn jeder nicht als äquivalent markierte Mutant ROT ist; ÜBERLEBT und FEHLER (Muster nicht genau einmal
gefunden, Bau schlägt fehl, kein Test benannt) geben Exit 1.

Das Programm arbeitet **nie im Arbeitsbaum**: Es kopiert die Wurzel (ohne `tmp/`, `.git/`, `.claude/` der obersten
Ebene) nach `<wurzel>/tmp/mutanten/lauf-<Zeit>/baum`, mutiert und testet dort mit `GOFLAGS=-tags=novulkan,nox11`
sowie `TMPDIR`/`GOTMPDIR` unter dem Laufordner und stellt die Datei in der Kopie nach jedem Mutanten wieder her. Alte
Läufe bleiben (gitignoriert) liegen. Ein Lauf über alle Gruppen dauert einige Minuten.

`go test` in diesem Modul legt Testordner über `t.TempDir()` an; `TMPDIR` dafür auf einen Ordner unter `tmp/` setzen.

## Format

Je Gruppe eine Datei `*.json` mit einer Liste von Einträgen:

```json
{"name": "C1 NeuKWin ohne Präfix", "datei": "internal/capture/kwin.go", "paket": "./internal/capture",
 "alt": "fmt.Errorf(\"capture: Session-Bus: %w\", err)", "neu": "fmt.Errorf(\"Session-Bus: %w\", err)",
 "aequivalent": false, "grund": ""}
```

- `datei` relativ zur Wurzel, `paket` wie für `go test`.
- `alt` muss **genau einmal** in der Datei stehen, sonst FEHLER „Muster N-mal gefunden“. Ändert sich der Produktcode,
  fällt das hier zuerst auf: Muster an die neue Stelle anpassen, wenn die Absicht dieselbe ist.
- `neu` ersetzt `alt` (leer = Stelle streichen). Der Mutant muss kompilieren.

Optional `"timeout": "60s"` reicht `-timeout` an `go test` durch. Für Mutanten, die einen Test hängen lassen statt ihn
rot zu machen (E2/E3 der Gruppe `rescue`, F-455): sonst läuft jeder bis zum Vorgabe-Timeout von 10 Minuten.

## Äquivalente Mutanten

`"aequivalent": true` heißt: Die Änderung verändert das beobachtbare Verhalten nicht (etwa eine doppelte Bedingung
oder ein Zweig, den kein Aufrufer erreicht). Der Mutant darf überleben; `grund` nennt, warum. Wird er doch rot,
meldet das Programm es, die Markierung ist dann zu prüfen. Ein Mutant, der aus Bequemlichkeit als äquivalent
markiert wird, entwertet die Probe: im Zweifel einen Test schreiben.

## Neue Mutanten

Nach einem Befund mit Test: eine Stelle der Behebung rückgängig machen oder abschwächen, als Eintrag aufnehmen,
`-trocken` und danach `-nur` laufen lassen. Der Name trägt die F-Nummer.
