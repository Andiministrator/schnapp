// SPDX-License-Identifier: GPL-3.0-or-later

// Package export bereitet den Export aus dem Editor auf (ADR-0025): Format aus Endung oder Filter, Inhalt aus
// demselben Renderer wie Arbeitsdatei und Zwischenablage, Schreiben über eine temporäre Datei im Zielordner.
// Ohne Gio und ohne D-Bus, damit alles ohne Fenster prüfbar ist.
package export

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andiministrator/schnapp/document"
	"github.com/Andiministrator/schnapp/fileformat"
	"github.com/Andiministrator/schnapp/render"
)

// Format ist ein Exportformat. Die Reihenfolge ist die der Filter im Dialog (ADR-0025 Punkt 2).
type Format int

const (
	PNG Format = iota
	JPG
	SVG
)

// Formate in Reihenfolge der Filter.
var Formate = []Format{PNG, JPG, SVG}

// JPGQualitaet ist die feste Qualität beim JPG-Export (ADR-0025 Punkt 4).
const JPGQualitaet = 90

// Endung liefert die anzuhängende Endung mit Punkt.
func (f Format) Endung() string {
	switch f {
	case JPG:
		return ".jpg"
	case SVG:
		return ".svg"
	}
	return ".png"
}

// MIME liefert den MIME-Typ.
func (f Format) MIME() string {
	switch f {
	case JPG:
		return "image/jpeg"
	case SVG:
		return "image/svg+xml"
	}
	return "image/png"
}

// Glob liefert das Dateimuster für den Dialogfilter.
func (f Format) Glob() string { return "*" + f.Endung() }

// AusIndex liefert das Format zum Filterindex; ein unbekannter Index (der Dialog nannte keinen Filter) gilt als PNG.
func AusIndex(i int) Format {
	if i < 0 || i >= len(Formate) {
		return PNG
	}
	return Formate[i]
}

// aus Endung (ohne Beachtung der Groß-/Kleinschreibung).
func ausEndung(pfad string) (Format, bool) {
	switch strings.ToLower(filepath.Ext(pfad)) {
	case ".png":
		return PNG, true
	case ".jpg", ".jpeg":
		return JPG, true
	case ".svg":
		return SVG, true
	}
	return PNG, false
}

// Waehle bestimmt Format und endgültigen Pfad (ADR-0025 Punkt 3): Eine bekannte Endung entscheidet; sonst gilt der
// gewählte Filter und seine Endung wird angehängt. angehaengt meldet das: Der Dialog hat dann nur den Namen ohne die
// Endung bestätigt, ein schon vorhandenes Ziel darf nicht still ersetzt werden (ErsetzeNichtStill, ADR-0025 Nachtrag 1).
func Waehle(pfad string, filter Format) (f Format, ziel string, angehaengt bool) {
	if f, ok := ausEndung(pfad); ok {
		return f, pfad, false
	}
	return filter, pfad + filter.Endung(), true
}

// ErrZielVorhanden: Das Ziel mit angehängter Endung gibt es schon; der Dialog hat nur den Namen ohne Endung bestätigt.
var ErrZielVorhanden = errors.New("Ziel vorhanden")

// PruefeAngehaengt lehnt ein schon vorhandenes Ziel ab, wenn Waehle die Endung angehängt hat (A-1). Fehler beim
// Prüfen außer „nicht vorhanden“ gelten als vorhanden (im Zweifel nichts ersetzen).
func PruefeAngehaengt(pfad string, angehaengt bool) error {
	if !angehaengt {
		return nil
	}
	_, err := os.Lstat(pfad)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err // etwa keine Rechte: als solcher melden, nicht als „vorhanden“ (Verify N-4)
	}
	return ErrZielVorhanden
}

// Schreibe schreibt das Dokument im Format f nach w: PNG und JPG aus dem fertig gerenderten Bild (render.Bild, das
// auch die Arbeitsdatei und die Zwischenablage speist), SVG über fileformat.ExportiereSVG. Ohne snAP/snAO-Blöcke und
// ohne Metadaten (Kernprinzip 3). Schwärzen ist in den Pixeln endgültig (ADR-0008, Kernprinzip 2).
func Schreibe(d *document.Dokument, f Format, w io.Writer) error {
	if f == SVG {
		return fileformat.ExportiereSVG(w, d)
	}
	b, err := render.Bild(d)
	if err != nil {
		return err
	}
	if f == JPG {
		return fileformat.ExportiereJPG(w, b, JPGQualitaet)
	}
	return fileformat.ExportierePNG(w, b)
}

// SchreibeAtomar legt die Datei über eine temporäre Datei im Zielordner an und benennt sie um (ADR-0025 Punkt 5, Nachtrag
// 1 und 2): Am Ziel steht nie eine halbe Datei, bei einem Fehler bleibt eine vorhandene Datei unberührt. Die Rechte
// folgen der umask des Prozesses; ersetzt die Datei eine vorhandene reguläre Datei, übernimmt sie deren Rechte schon vor
// dem Schreiben (die temporäre Datei ist nie offener als das Ziel). Ist das Ziel ein symbolischer Link, wird dorthin
// geschrieben, wohin er zeigt (der Link bleibt).
func SchreibeAtomar(pfad string, schreibe func(io.Writer) error) error {
	return schreibeAtomar(pfad, false, schreibe)
}

// SchreibeOhneErsetzen ist SchreibeAtomar, legt die Datei aber nur an, wenn es das Ziel noch nicht gibt — geprüft beim
// Anlegen selbst (harter Link), nicht vorher: Entsteht das Ziel während des Renderns, gilt ErrZielVorhanden (Verify N-4).
// Für eine angehängte Endung, die der Dialog nicht bestätigt hat. Kann das Dateisystem keine harten Links, wird vorher
// geprüft und umbenannt (bestmöglich).
func SchreibeOhneErsetzen(pfad string, schreibe func(io.Writer) error) error {
	return schreibeAtomar(pfad, true, schreibe)
}

func schreibeAtomar(pfad string, ohneErsetzen bool, schreibe func(io.Writer) error) (fehler error) {
	if st, err := os.Lstat(pfad); err == nil && st.Mode()&os.ModeSymlink != 0 {
		echt, err := filepath.EvalSymlinks(pfad)
		if err != nil {
			return err // Link ins Leere: nichts anlegen, wohin der Name nicht zeigt
		}
		pfad = echt
	}
	tmp, err := tempDatei(filepath.Dir(pfad))
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() {
		if fehler != nil {
			tmp.Close()
			os.Remove(name)
		}
	}()
	if st, err := os.Lstat(pfad); err == nil && st.Mode().IsRegular() {
		// Vor dem Schreiben (Verify N-2). Ein Fehler (etwa exFAT ohne Rechte) bricht nicht ab.
		if err := tmp.Chmod(st.Mode().Perm()); err != nil {
			Diagnose("Rechte der ersetzten Datei nicht übernommen: %v", err)
		}
	}
	if err := schreibe(tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("schließen: %w", err)
	}
	if !ohneErsetzen {
		return os.Rename(name, pfad)
	}
	switch err := os.Link(name, pfad); {
	case err == nil:
		os.Remove(name)
		return nil
	case errors.Is(err, os.ErrExist):
		return ErrZielVorhanden
	}
	if _, err := os.Lstat(pfad); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return ErrZielVorhanden
		}
		return err
	}
	return os.Rename(name, pfad)
}

// Diagnose nimmt Meldungen, die nur im Debug-Log erscheinen sollen; die Oberfläche setzt sie, sonst bleibt sie leer.
var Diagnose = func(format string, a ...any) {}

// tempDatei legt die temporäre Datei exklusiv mit Modus 0666 an (der Kern wendet die umask an) und zufälligem Namen.
func tempDatei(ordner string) (*os.File, error) {
	for versuch := 0; ; versuch++ {
		var z [8]byte
		if _, err := rand.Read(z[:]); err != nil {
			return nil, err
		}
		name := filepath.Join(ordner, ".schnapp-export-"+hex.EncodeToString(z[:]))
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if errors.Is(err, os.ErrExist) && versuch < 10 {
			continue
		}
		return f, err
	}
}
