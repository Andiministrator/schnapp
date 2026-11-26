// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
	"github.com/Andiministrator/schnapp/internal/render"
)

// importBeimOeffnen schaltet die Übernahme beim Öffnen (ADR-0016 Nachtrag 10). Tests, die den Rückfall „erstes
// Speichern legt ab“ prüfen, schalten sie ab.
var importBeimOeffnen = true

// importiere übernimmt eine Datei außerhalb der Ablage vor dem Öffnen als Aufnahme der Bibliothek und liefert den Pfad,
// den der Editor öffnen soll: die neue oder eine früher aus demselben Inhalt übernommene Aufnahme. Eine Datei in der
// Ablage und jeder Fehler liefern pfad unverändert; dann öffnet der gewöhnliche Weg die Datei selbst und meldet einen
// Lade- oder Budgetfehler (Punkt 3). Die Datei von außen bleibt unberührt.
func importiere(b *library.Bibliothek, pfad string, ohneGrenze bool, kante document.Kante) string {
	if !importBeimOeffnen || b == nil {
		return pfad
	}
	abs, err := filepath.Abs(pfad)
	if err != nil {
		return pfad
	}
	if _, ok := b.Relativ(abs); ok {
		return pfad
	}
	// Erst prüfen wie das Öffnen (reguläre Datei, höchstens fileformat.MaxDatei): Eine FIFO, ein Gerät oder eine
	// riesige Datei würde sonst beim Hashen blockieren (Kritik K-3); der gewöhnliche Weg meldet den Fehler.
	fi, err := os.Stat(abs)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > fileformat.MaxDatei {
		return pfad
	}
	summe, err := dateiSumme(abs)
	if err != nil {
		return pfad
	}
	if rel, ok := b.Importiert(summe); ok {
		diagnoseImport("schon übernommen: %s", rel)
		ziel := filepath.Join(b.Orte.Ablage, filepath.FromSlash(rel))
		merkeHerkunft(ziel, abs)
		return ziel
	}
	_, g, _, err := ladeFuerEditor(abs, ohneGrenze)
	if err != nil {
		return pfad
	}
	if g.Fremd {
		if err := g.Dokument.VorgabeKante(kante); err != nil {
			return pfad
		}
	}
	sichtbar, err := render.Bild(g.Dokument)
	if err != nil {
		return pfad
	}
	m := g.Meta
	m.Erstellt = time.Time{} // Zeitpunkt des Öffnens: vorn in der Leiste
	e, err := b.Ablegen(g.Dokument, sichtbar, m)
	if e.Pfad == "" {
		diagnoseImport("Übernahme gescheitert: %v", err)
		return pfad
	}
	if err != nil { // abgelegt, aber Index oder Vorschau scheiterten: die Aufnahme gilt trotzdem
		diagnoseImport("übernommen mit Fehler: %v", err)
	}
	if err := b.MerkeImport(summe, e.Pfad); err != nil {
		diagnoseImport("Liste der Übernahmen: %v", err)
	}
	ziel := filepath.Join(b.Orte.Ablage, filepath.FromSlash(e.Pfad))
	merkeHerkunft(ziel, abs)
	return ziel
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
