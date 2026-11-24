// SPDX-License-Identifier: GPL-3.0-or-later

// Package export bereitet den Export aus dem Editor auf (ADR-0025): Format aus Endung oder Filter, Inhalt aus
// demselben Renderer wie Arbeitsdatei und Zwischenablage, Schreiben über eine temporäre Datei im Zielordner.
// Ohne Gio und ohne D-Bus, damit alles ohne Fenster prüfbar ist.
package export

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/errprefix"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/render"
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

// AusEndung liefert das Format nach der Endung von pfad (.png, .jpg/.jpeg, .svg, ohne Beachtung der
// Groß-/Kleinschreibung) und ob die Endung bekannt ist (Kommandozeile, ADR-0030).
func AusEndung(pfad string) (Format, bool) { return ausEndung(pfad) }

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
var ErrZielVorhanden = errors.New("export: Ziel vorhanden")

// PruefeAngehaengt lehnt ein schon vorhandenes Ziel ab, wenn Waehle die Endung angehängt hat (A-1). Fehler beim
// Prüfen außer „nicht vorhanden“ gelten als vorhanden (im Zweifel nichts ersetzen).
func PruefeAngehaengt(pfad string, angehaengt bool) (fehler error) {
	defer errprefix.Defer("export", &fehler)
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
// Ein SVG, dessen Lupen mehr Pixel einbetteten, als fileformat.PruefeSVGExport zulässt, lehnt es vor dem Kodieren ab
// (fileformat.ErrLupenZuGross, ADR-0033 Nachtrag 5).
func Schreibe(d *document.Dokument, f Format, w io.Writer) (fehler error) {
	defer errprefix.Defer("export", &fehler)
	return schreibe(d, f, w, false)
}

// SchreibeOhneGrenze ist Schreibe ohne die Grenze der Lupenpixel im SVG (--ignore-limit, ADR-0033 Nachtrag 5).
func SchreibeOhneGrenze(d *document.Dokument, f Format, w io.Writer) (fehler error) {
	defer errprefix.Defer("export", &fehler)
	return schreibe(d, f, w, true)
}

func schreibe(d *document.Dokument, f Format, w io.Writer, ohneGrenze bool) error {
	if f == SVG {
		if ohneGrenze {
			return fileformat.ExportiereSVGOhneGrenze(w, d)
		}
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
// 1 und 2): Am Ziel steht nie eine halbe Datei, bei einem Fehler bleibt eine vorhandene Datei unberührt. Einzige
// Ausnahme ist der Rückfall (schreibeDirekt): darf neben dem Ziel keine Datei entstehen, wird das vorhandene Ziel
// direkt beschrieben; scheitert das, wird der alte Inhalt zurückgeschrieben, und nur wenn auch das scheitert, meldet
// ErrZielBeschaedigt eine unvollständige Datei (F-322). Die Rechte
// folgen der umask des Prozesses; ersetzt die Datei eine vorhandene reguläre Datei, übernimmt sie deren Rechte schon vor
// dem Schreiben (die temporäre Datei ist nie offener als das Ziel). Ist das Ziel ein symbolischer Link, wird dorthin
// geschrieben, wohin er zeigt (der Link bleibt).
func SchreibeAtomar(pfad string, schreibe func(io.Writer) error) (fehler error) {
	defer errprefix.Defer("export", &fehler)
	return schreibeAtomar(pfad, false, 0, schreibe)
}

// SchreibeAtomarPrivat ist SchreibeAtomar, legt eine neue Datei aber mit den Rechten 0600 an statt nach der umask
// (Kommandozeile --output, ADR-0030; Bildschirminhalt wie in der Bibliothek, ADR-0016). Die Rechte werden gesetzt,
// bevor ein Byte geschrieben wird. Eine vorhandene Zieldatei behält ihre Rechte wie bei SchreibeAtomar. Der Export
// aus dem Editor und --export bleiben bei SchreibeAtomar (KA-4).
func SchreibeAtomarPrivat(pfad string, schreibe func(io.Writer) error) (fehler error) {
	defer errprefix.Defer("export", &fehler)
	return schreibeAtomar(pfad, false, 0o600, schreibe)
}

// SchreibeOhneErsetzen ist SchreibeAtomar, legt die Datei aber nur an, wenn es das Ziel noch nicht gibt — geprüft beim
// Anlegen selbst (harter Link), nicht vorher: Entsteht das Ziel während des Renderns, gilt ErrZielVorhanden (Verify N-4).
// Für eine angehängte Endung, die der Dialog nicht bestätigt hat. Kann das Dateisystem keine harten Links, benennt es
// mit RENAME_NOREPLACE um; kann es auch das nicht, wird vorher geprüft und umbenannt (bestmöglich, F-494).
func SchreibeOhneErsetzen(pfad string, schreibe func(io.Writer) error) (fehler error) {
	defer errprefix.Defer("export", &fehler)
	return schreibeAtomar(pfad, true, 0, schreibe)
}

// SchreibeOhneErsetzenPrivat ist SchreibeOhneErsetzen mit den Rechten 0600 für die neue Datei (MCP-Werkzeug
// `capture_screen` mit `output`, ADR-0030 Nachtrag 4 Punkt 2).
func SchreibeOhneErsetzenPrivat(pfad string, schreibe func(io.Writer) error) (fehler error) {
	defer errprefix.Defer("export", &fehler)
	return schreibeAtomar(pfad, true, 0o600, schreibe)
}

// schreibeAtomar: neuRechte != 0 setzt die Rechte einer neu entstehenden Datei (sonst gilt die umask).
func schreibeAtomar(pfad string, ohneErsetzen bool, neuRechte os.FileMode, schreibe func(io.Writer) error) (fehler error) {
	if st, err := os.Lstat(pfad); err == nil && st.Mode()&os.ModeSymlink != 0 {
		echt, err := filepath.EvalSymlinks(pfad)
		if err != nil {
			return err // Link ins Leere: nichts anlegen, wohin der Name nicht zeigt
		}
		pfad = echt
	}
	tmp, err := oeffneTemp(filepath.Dir(pfad))
	if err != nil {
		if !ohneErsetzen && keinSchreibrechtImOrdner(err) {
			return schreibeDirekt(pfad, schreibe, err)
		}
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
	} else if neuRechte != 0 {
		// Neue Datei mit festen Rechten, vor dem Schreiben. Ein Dateisystem ohne Unix-Rechte (etwa exFAT) lehnt das
		// womöglich ab; dann gelten dessen feste Rechte, wie beim Ersetzen oben.
		if err := tmp.Chmod(neuRechte); err != nil {
			Diagnose("Rechte der neuen Datei nicht gesetzt: %v", err)
		}
	}
	if err := schreibe(tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("export: schließen: %w", err)
	}
	if !ohneErsetzen {
		return os.Rename(name, pfad)
	}
	switch err := linkNaht(name, pfad); {
	case err == nil:
		os.Remove(name)
		return nil
	case errors.Is(err, os.ErrExist):
		return ErrZielVorhanden
	}
	// Ohne harte Links: Umbenennen, das ein vorhandenes Ziel nicht ersetzt — geprüft vom Kernel beim Umbenennen selbst,
	// ohne Fenster zwischen Prüfen und Umbenennen (F-494).
	switch err := umbenennenOhneErsetzen(name, pfad); {
	case err == nil:
		return nil
	case errors.Is(err, os.ErrExist):
		return ErrZielVorhanden
	case !errors.Is(err, unix.EINVAL) && !errors.Is(err, unix.ENOSYS):
		return err
	}
	// Auch das kann das Dateisystem nicht (etwa manche FUSE- oder Netz-Dateisysteme): bestmöglich prüfen, dann
	// umbenennen. Hier bleibt ein kurzes Fenster.
	if _, err := os.Lstat(pfad); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return ErrZielVorhanden
		}
		return err
	}
	return os.Rename(name, pfad)
}

// Nähte für Tests: harter Link und Umbenennen ohne Ersetzen (Dateisystem ohne diese Fähigkeiten nachbauen).
var (
	linkNaht               = os.Link
	umbenennenOhneErsetzen = func(alt, neu string) error {
		err := unix.Renameat2(unix.AT_FDCWD, alt, unix.AT_FDCWD, neu, unix.RENAME_NOREPLACE)
		if err != nil {
			return &os.LinkError{Op: "renameat2", Old: alt, New: neu, Err: err}
		}
		return nil
	}
)

// keinSchreibrechtImOrdner: Der Ordner erlaubt keine neue Datei (Rechte, schreibgeschützt).
func keinSchreibrechtImOrdner(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.EROFS)
}

// MaxRueckfallAlt ist die größte vorhandene Datei, die der Rückfall vorher in den Speicher liest, um sie bei einem
// Schreibfehler wiederherzustellen (F-322). Ist die Datei größer, wird sie gar nicht direkt beschrieben.
const MaxRueckfallAlt = 256 << 20

// ErrZielBeschaedigt: Der Rückfall hat das Ziel beim Schreiben gekürzt und den alten Inhalt nicht wiederherstellen
// können; die Zieldatei ist unvollständig (F-322).
var ErrZielBeschaedigt = errors.New("export: Zieldatei beschädigt")

// zielDatei ist, was der Rückfall von der geöffneten Zieldatei braucht (Naht für Tests: Schreibfehler nachbauen).
type zielDatei interface {
	io.Reader
	io.Writer
	io.Seeker
	Truncate(size int64) error
	Sync() error
	Close() error
}

// errZielGewechselt: Unter dem Namen liegt nicht mehr die Datei, die Lstat sah (F-330).
var errZielGewechselt = errors.New("export: Ziel hat gewechselt")

// Wartezeit, wenn das Öffnen an einer Lease anderer Programme scheitert (EAGAIN mit O_NONBLOCK, F-338): kurz wiederholen,
// dann aufgeben (es gilt der Fehler der temporären Datei, die Datei bleibt unverändert).
var (
	leaseVersuche = 20
	leasePause    = 50 * time.Millisecond
)

// oeffneDirekt öffnet das Ziel des Rückfalls einmal zum Lesen und Schreiben, ohne einem Link zu folgen (O_NOFOLLOW) und
// ohne zu kürzen; Sichern, Kürzen, Schreiben und Wiederherstellen laufen danach alle über diesen einen Deskriptor
// (F-338). Die geöffnete Datei muss dieselbe reguläre Datei sein, die Lstat sah (st), sonst bleibt sie unverändert
// (F-330). O_NONBLOCK lässt ein untergeschobenes FIFO nicht blockieren; bei einer regulären Datei wirkt es nur beim
// Öffnen: Hält ein anderes Programm eine Lease, scheitert das Öffnen sofort mit EAGAIN statt zu warten — dann wird
// kurz wiederholt. Nur Tests biegen es um.
var oeffneDirekt = func(pfad string, st os.FileInfo) (zielDatei, error) {
	var f *os.File
	var err error
	for versuch := 0; ; versuch++ {
		f, err = os.OpenFile(pfad, os.O_RDWR|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
		if !errors.Is(err, syscall.EAGAIN) || versuch >= leaseVersuche {
			break
		}
		time.Sleep(leasePause)
	}
	if err != nil {
		return nil, err
	}
	fst, err := f.Stat()
	if err != nil || !fst.Mode().IsRegular() || !os.SameFile(st, fst) {
		f.Close()
		return nil, errZielGewechselt
	}
	return f, nil
}

// schreibeDirekt ist der Rückfall, wenn neben dem Ziel keine temporäre Datei entstehen darf (F-196): etwa unter
// Flatpak, wo das Dokumenten-Portal nur die gewählte Datei freigibt, nicht ihren Ordner. Nur für ein vorhandenes,
// reguläres, beschreibbares Ziel bis MaxRueckfallAlt; sonst gilt der ursprüngliche Fehler. Das Dokument wird erst
// vollständig in den Speicher gerendert; dann wird das Ziel einmal geöffnet, sein alter Inhalt über denselben
// Deskriptor gelesen, die Datei gekürzt und neu geschrieben. Scheitert das Schreiben, wird der alte Inhalt über
// denselben Deskriptor zurückgeschrieben — in genau die Datei, die gesichert wurde, auch wenn der Name inzwischen
// woanders hinzeigt (F-338). Nur wenn auch das misslingt, ist das Ziel beschädigt und der Fehler wickelt
// ErrZielBeschaedigt ein (nicht atomar, bewusst nur im Rückfall, F-322).
func schreibeDirekt(pfad string, schreibe func(io.Writer) error, ursache error) error {
	st, err := os.Lstat(pfad)
	if err != nil || !st.Mode().IsRegular() || st.Size() > MaxRueckfallAlt {
		return ursache
	}
	var puffer bytes.Buffer
	if err := schreibe(&puffer); err != nil {
		return err
	}
	f, err := oeffneDirekt(pfad, st)
	if err != nil {
		return ursache // Ziel nicht beschreibbar oder getauscht: der Fehler der temporären Datei erklärt es besser
	}
	alt, err := leseAlt(f, st)
	if err != nil {
		f.Close()
		return ursache // ohne Sicherung nicht direkt schreiben
	}
	Diagnose("Rückfall: direkt in %s geschrieben, keine temporäre Datei möglich: %v", filepath.Base(pfad), ursache)
	veraendert, werr := ersetzeInhalt(f, puffer.Bytes())
	if werr == nil {
		if err := f.Close(); err != nil {
			// Nach dem Sync steht der neue Inhalt vollständig: ein späterer Close-Fehler ändert daran nichts. Die
			// Oberfläche soll keinen Fehlschlag melden, obwohl die Datei stimmt; der Fehler bleibt als Diagnose (F-344).
			Diagnose("Rückfall: Schließen von %s nach erfolgreichem Schreiben gescheitert: %v", filepath.Base(pfad), err)
		}
		return nil
	}
	if !veraendert {
		// Schon das Kürzen scheiterte: die Datei ist unberührt, es gibt nichts wiederherzustellen und nichts zu beklagen.
		f.Close()
		return werr
	}
	// Wiederherstellen: der alte Inhalt, vollständig, in dieselbe Datei.
	_, rerr := ersetzeInhalt(f, alt)
	cerr := f.Close()
	if rerr == nil && cerr != nil {
		Diagnose("Rückfall: Schließen von %s nach der Wiederherstellung gescheitert: %v", filepath.Base(pfad), cerr)
	}
	if rerr != nil {
		return fmt.Errorf("%w: %v; Wiederherstellung gescheitert: %v", ErrZielBeschaedigt, werr, rerr)
	}
	return werr
}

// ersetzeInhalt kürzt die geöffnete Datei und schreibt inhalt vom Anfang an, mit Sync. veraendert ist false, solange
// noch nichts an der Datei geändert wurde (das Kürzen selbst scheiterte).
func ersetzeInhalt(f zielDatei, inhalt []byte) (veraendert bool, err error) {
	if err := f.Truncate(0); err != nil {
		return false, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return true, err
	}
	if _, err := f.Write(inhalt); err != nil {
		return true, err
	}
	return true, f.Sync()
}

// leseAlt liest den alten Inhalt über den geöffneten Deskriptor; er muss genau so lang sein, wie Lstat sah. Hat sich die
// Datei dazwischen geändert (etwa über MaxRueckfallAlt hinaus gewachsen), wäre die Sicherung gekürzt oder veraltet und
// taugt nicht zum Wiederherstellen.
func leseAlt(f zielDatei, st os.FileInfo) ([]byte, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	alt, err := io.ReadAll(io.LimitReader(f, MaxRueckfallAlt+1))
	if err != nil {
		return nil, err
	}
	if int64(len(alt)) != st.Size() {
		return nil, errors.New("export: Ziel hat sich beim Lesen geändert")
	}
	return alt, nil
}

// Diagnose nimmt Meldungen, die nur im Debug-Log erscheinen sollen; die Oberfläche setzt sie, sonst bleibt sie leer.
var Diagnose = func(format string, a ...any) {}

// tempSchreiber ist, was schreibeAtomar von der temporären Datei braucht (Naht für Tests: Close-Fehler nachbauen, F-340).
type tempSchreiber interface {
	io.Writer
	Name() string
	Chmod(mode os.FileMode) error
	Sync() error
	Close() error
}

// oeffneTemp legt die temporäre Datei an; nur Tests biegen es um (Hülle um die echte Datei).
var oeffneTemp = func(ordner string) (tempSchreiber, error) {
	f, err := tempDatei(ordner)
	if err != nil {
		return nil, err
	}
	return f, nil
}

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
