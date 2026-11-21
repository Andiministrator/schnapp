// SPDX-License-Identifier: GPL-3.0-or-later

package library

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"testing"

	"github.com/Andiministrator/schnapp/internal/fileformat"
)

// verderbeIDAT ersetzt die Daten aller IDAT-Blöcke durch Müll mit richtiger Prüfsumme (gleiche Länge).
func verderbeIDAT(t *testing.T, pfad string) {
	t.Helper()
	roh, err := os.ReadFile(pfad)
	if err != nil {
		t.Fatal(err)
	}
	for i := 8; i+12 <= len(roh); {
		n := int(binary.BigEndian.Uint32(roh[i:]))
		if string(roh[i+4:i+8]) == "IDAT" {
			for j := i + 8; j < i+8+n; j++ {
				roh[j] = 0xAB
			}
			binary.BigEndian.PutUint32(roh[i+8+n:], crc32.ChecksumIEEE(roh[i+4:i+8+n]))
		}
		i += 12 + n
	}
	if err := os.WriteFile(pfad, roh, 0o600); err != nil {
		t.Fatal(err)
	}
}

// Der Neuaufbau liest nur snAP und IHDR: Eine Aufnahme mit zerstörten Pixeln, aber gültigem snAP kommt in den Index
// (die Vorschau lässt sich dann nicht nachziehen, das ist hier nicht Teil der Aussage).
func TestNeuaufbauLiestNurMetadaten(t *testing.T) {
	b, _ := neueBibliothek(t)
	e := ablegen(t, b, fileformat.Metadaten{Erstellt: zeit("2026-11-19T12:35:12Z"), Schlagworte: []string{"Rechnung"}})
	verderbeIDAT(t, ablageDatei(b, e.Pfad))
	erg, err := b.Neuaufbau()
	if err != nil {
		t.Fatal(err)
	}
	if erg.Aufgenommen != 1 || len(erg.Uebersprungen) != 0 {
		t.Fatalf("Ergebnis %+v", erg)
	}
	g, _ := b.Laden()
	if len(g.Eintraege) != 1 || g.Eintraege[0].Breite != 40 || g.Eintraege[0].Hoehe != 30 ||
		len(g.Eintraege[0].Schlagworte) != 1 || g.Eintraege[0].Schlagworte[0] != "Rechnung" {
		t.Errorf("Index %+v", g.Eintraege)
	}
}
