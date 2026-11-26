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

**HÄNGT** (F-478) ist ein eigener Status und **zählt als rot**: Das Paket endete am Test-Timeout (`-timeout`, Vorgabe
10 Minuten, je Mutant über `timeout` kürzbar), und `go test` nannte laufende Tests („running tests:“) — mindestens ein
Test war gestartet und kam nicht zurück, der Mutant ist also bemerkt, auch wenn kein Test ein `--- FAIL` ausgibt. Fiel
vorher ein Test, ist der Status ROT. Ein Timeout ohne laufenden Test (Hänger in `init` oder `TestMain`, Bau) ist
dagegen FEHLER. Die Ausgabe nennt die hängenden Tests und zählt HÄNGT in der Summe getrennt. HÄNGT gilt nur, wenn der
Grundlauf (unveränderter Code, gleiches Paket, gleicher `timeout`) durchläuft (F-489): Der Grundlauf läuft einmal je
Kombination aus Paket und `timeout`; scheitert er am `timeout`, melden alle betroffenen Mutanten FEHLER mit
„TIMEOUT-ZU-KNAPP“ (Exit 1) — dann den `timeout` erhöhen.

Das Programm arbeitet **nie im Arbeitsbaum**: Es kopiert die Wurzel (ohne `tmp/`, `.git/`, `.claude/` der obersten
Ebene) nach `<wurzel>/tmp/mutanten/lauf-<Zeit>-<Zufall>/baum` (eindeutig auch bei zwei Läufen in derselben Sekunde, F-495), mutiert und testet dort mit `GOFLAGS=-trimpath -tags=novulkan,nox11`
sowie `TMPDIR`/`GOTMPDIR` unter dem Laufordner und stellt die Datei in der Kopie nach jedem Mutanten wieder her. Alte
Läufe bleiben (gitignoriert) liegen. Ein Lauf über alle Gruppen dauert einige Minuten.

**Build-Cache (F-833):** Gebaut wird in einen eigenen Cache `<wurzel>/tmp/mutanten/gocache`, nicht in den des Benutzers.
`-trimpath` hält die Cache-Schlüssel über Läufe gleich (ohne trimpath stünde der wechselnde Laufordner in jedem
Schlüssel, und jeder Lauf legte alles neu ab — am 26.11.2026 116 GB und eine volle Platte). Über `cacheGrenze` (8 GiB)
leert das Programm den Cache mit `go clean -cache`, vor dem Lauf und alle `cachePruefTakt` (25) Mutanten; nutzt ein
paralleler Lauf ihn gerade (geteilte flock-Sperre `gocache.lock`), wird nicht geleert. Von Hand leeren:
`GOCACHE=$PWD/tmp/mutanten/gocache go clean -cache` in der Wurzel. Der erste Lauf nach dem Leeren baut alles neu (Gio mit
cgo, einige Minuten).

`go test` in diesem Modul legt Testordner über `t.TempDir()` an; `TMPDIR` dafür auf einen Ordner unter `tmp/` setzen. Die eigenen Tests bauen dabei die Standardbibliothek in einen frischen Cache unter `t.TempDir()` (einige Sekunden
länger, gewollt).

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
rot zu machen (E2/E3 der Gruppe `rescue`, F-455): sonst läuft jeder bis zum Vorgabe-Timeout von 10 Minuten. Der Wert
muss eine positive Dauer sein (`time.ParseDuration`), und unbekannte Schlüssel in den JSON-Dateien sind Fehler: beides
scheitert schon beim Laden, auch mit `-trocken` (F-478).

Optional `"tags": "probe"` hängt Build-Tags (kommagetrennt) an `novulkan,nox11` an, damit auch Pakete mit Tag gemutet
werden können (Gruppe `fernbedienung`: `./cmd/schnapp-fernbedienung` baut nur mit `probe`, F-614). Der Grundlauf läuft
einmal je Paket, `timeout` und `tags`; ungültige Namen scheitern beim Laden.

Optional `"modul": "ocr"` lässt `go test` im Unterordner mit eigenem `go.mod` laufen (Programm-Modul `ocr/`,
schnapp-ocr, F-643); `datei` bleibt relativ zur Wurzel, `paket` gilt im Modul (meist `.`). Ein `modul` außerhalb des
Baums scheitert beim Laden; der Grundlauf unterscheidet auch nach Modul.

## Äquivalente Mutanten

`"aequivalent": true` heißt: Die Änderung verändert das beobachtbare Verhalten nicht (etwa eine doppelte Bedingung
oder ein Zweig, den kein Aufrufer erreicht). Der Mutant darf überleben; `grund` nennt, warum. Wird er doch rot,
meldet das Programm es, die Markierung ist dann zu prüfen. Ein Mutant, der aus Bequemlichkeit als äquivalent
markiert wird, entwertet die Probe: im Zweifel einen Test schreiben.

## Neue Mutanten

Nach einem Befund mit Test: eine Stelle der Behebung rückgängig machen oder abschwächen, als Eintrag aufnehmen,
`-trocken` und danach `-nur` laufen lassen. Der Name trägt die F-Nummer.
