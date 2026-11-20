// SPDX-License-Identifier: GPL-3.0-or-later

// Package export bereitet den Export aus dem Editor auf (ADR-0025): Format aus Endung oder Filter, Inhalt aus
// demselben Renderer wie Arbeitsdatei und Zwischenablage, Schreiben über eine temporäre Datei im Zielordner.
// Ohne Gio und ohne D-Bus, damit alles ohne Fenster prüfbar ist.
package export

import (
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
// gewählte Filter und seine Endung wird angehängt.
func Waehle(pfad string, filter Format) (Format, string) {
	if f, ok := ausEndung(pfad); ok {
		return f, pfad
	}
	return filter, pfad + filter.Endung()
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

// SchreibeAtomar legt die Datei über eine temporäre Datei im Zielordner an und benennt sie um (ADR-0025 Punkt 5): Am
// Ziel steht nie eine halbe Datei, bei einem Fehler bleibt eine vorhandene Datei unberührt. Rechte 0644.
func SchreibeAtomar(pfad string, schreibe func(io.Writer) error) (fehler error) {
	tmp, err := os.CreateTemp(filepath.Dir(pfad), ".schnapp-export-*")
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
	if err := schreibe(tmp); err != nil {
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("schließen: %w", err)
	}
	return os.Rename(name, pfad)
}
