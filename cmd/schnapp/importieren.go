// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
)

// importBeimOeffnen schaltet die Übernahme einer Datei von außen (ADR-0016 Nachträge 10 und 11): das Wiederfinden einer
// früher übernommenen Aufnahme beim Öffnen, das Speichern bei der ersten Änderung und das Merken in imports.jsonl. Tests,
// die den Rückfall „erstes Speichern legt ab“ prüfen, und der Drehbuch-Lauf schalten sie ab.
var importBeimOeffnen = true

// vorOeffnen bereitet das Öffnen einer Datei vor (ADR-0016 Nachtrag 11): Liegt sie außerhalb der Ablage und wurde
// derselbe Inhalt schon einmal übernommen, liefert es den Pfad dieser Aufnahme (sie öffnet sich samt Beschriftungen).
// Sonst bleibt pfad, wie er ist; aussen sagt dann, dass die Datei von außen kommt: Der Editor speichert sie bei der
// ersten Änderung, und dieses Speichern legt sie als neue Aufnahme ab und merkt die Summe (merkeUebernahme). Ohne
// Änderung kommt nichts in die Bibliothek. Die Datei von außen bleibt immer unberührt.
func vorOeffnen(b *library.Bibliothek, pfad string) (ziel string, aussen bool) {
	if !importBeimOeffnen || b == nil {
		return pfad, false
	}
	abs, err := filepath.Abs(pfad)
	if err != nil {
		return pfad, false
	}
	if _, ok := b.Relativ(abs); ok {
		return pfad, false
	}
	// Erst prüfen wie das Öffnen (reguläre Datei, höchstens fileformat.MaxDatei): Eine FIFO, ein Gerät oder eine
	// riesige Datei würde sonst beim Hashen blockieren (Kritik K-3); das Öffnen meldet den Fehler.
	fi, err := os.Stat(abs)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > fileformat.MaxDatei {
		return pfad, false
	}
	summe, err := dateiSumme(abs)
	if err != nil {
		summen.Store(abs, []byte(nil)) // übernommen wird trotzdem, nur später nicht wiedergefunden
		return pfad, true
	}
	if ziel, ok := uebernommeneAufnahme(b, summe); ok {
		diagnoseImport("schon übernommen: %s", ziel)
		merkeHerkunft(ziel, abs)
		return ziel, false
	}
	summen.Store(abs, summe)
	return pfad, true
}

// uebernommeneAufnahme ist die Aufnahme (absolut), die einen Inhalt mit der Summe summe schon übernommen hat.
func uebernommeneAufnahme(b *library.Bibliothek, summe []byte) (string, bool) {
	rel, ok := b.Importiert(summe)
	if !ok {
		return "", false
	}
	return filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel)), true
}

// zuordnerFuer bildet eine in Datei → Öffnen… gewählte Datei auf die Aufnahme ab, die sie öffnen wird (F-828): eine
// früher übernommene Datei von außen auf ihre Aufnahme, alles andere auf sich selbst (absolut). Ohne Nebenwirkung,
// anders als vorOeffnen; so erkennt der Editor eine Wahl der offenen Aufnahme, bevor er speichert und neu lädt.
func zuordnerFuer(b *library.Bibliothek) func(pfad string) string {
	return func(pfad string) string {
		abs, err := filepath.Abs(pfad)
		if err != nil || !importBeimOeffnen || b == nil {
			return pfad
		}
		if _, ok := b.Relativ(abs); ok {
			return abs
		}
		fi, err := os.Stat(abs)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > fileformat.MaxDatei {
			return abs
		}
		summe, err := dateiSumme(abs)
		if err != nil {
			return abs
		}
		if ziel, ok := uebernommeneAufnahme(b, summe); ok {
			return ziel
		}
		return abs
	}
}

// summen merkt je geöffneter Datei von außen (absoluter Pfad) die SHA-256-Summe ihres Inhalts beim Öffnen, bis das
// erste Speichern sie als Aufnahme ablegt (merkeUebernahme).
var summen sync.Map

// merkeUebernahme: Das Speichern hat die Datei von außen aussen als neue Aufnahme ziel (absolut, rel relativ zur Ablage)
// abgelegt. Die Bibliothek merkt sich den Inhalt beim Öffnen (imports.jsonl, Nachtrag 10 Punkt 2), und die Aufnahme
// merkt ihre Herkunft für die Warnung beim Schwärzen (Punkt 6).
func merkeUebernahme(b *library.Bibliothek, aussen, rel, ziel string) {
	v, ok := summen.LoadAndDelete(aussen)
	if !ok {
		return
	}
	if summe := v.([]byte); summe != nil {
		if err := b.MerkeImport(summe, rel); err != nil {
			diagnoseImport("Liste der Übernahmen: %v", err)
		}
	}
	merkeHerkunft(ziel, aussen)
}

// herkunft merkt je übernommener Aufnahme (absoluter Pfad) den Namen der Datei von außen ohne Ordner (F-155), damit das
// Speichern beim Schwärzen vor ihr warnt (ADR-0016 Nachtrag 10 Punkt 6, Kritik K-1).
var herkunft sync.Map

func merkeHerkunft(aufnahme, aussen string) { herkunft.Store(aufnahme, filepath.Base(aussen)) }

// herkunftVon liefert den Namen der Datei von außen, aus der die Aufnahme abs übernommen wurde.
func herkunftVon(abs string) (string, bool) {
	v, ok := herkunft.Load(abs)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// dateiSumme ist die SHA-256-Summe des Inhalts der Datei pfad.
func dateiSumme(pfad string) ([]byte, error) {
	f, err := os.Open(pfad)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// diagnoseImport schreibt nur mit SCHNAPP_DEBUG (Pfade gehören nicht in die normale Ausgabe).
func diagnoseImport(format string, args ...any) { diagnose(os.Stderr, "Übernahme: "+format, args...) }
