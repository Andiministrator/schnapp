// Arbeitsordner (gitignoriert bis auf diese Datei). Das eigene Modul trennt tmp/ vom Wurzelmodul, damit
// `go build ./...` und `go test ./...` keine hier abgelegten .go-Dateien mitbauen. Nichts hier wird gebaut.
module schnapp-tmp

go 1.27.4
