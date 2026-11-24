// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"os"
	"sync"
	"syscall"

	"github.com/Andiministrator/schnapp/internal/library"
)

// SpeicherGrenze ist die größte Menge dekodierter Vorschaubilder im Speicher (ADR-0036 Punkt 6): 64 MiB, die zuletzt
// am längsten nicht gezeigten fallen heraus.
const SpeicherGrenze = 64 << 20

// Grenzen für eine Datei im Vorschau-Cache: Er gehört dem Benutzer, aber eine Vorschau ist höchstens
// library.VorschauSeite groß; was deutlich größer ist, wird nicht dekodiert.
const (
	maxVorschauDatei = 4 << 20
	maxVorschauSeite = 4 * library.VorschauSeite
)

// Zustand eines Vorschaubilds im Speicher.
type Zustand int

const (
	Laedt  Zustand = iota // angefordert, noch nicht da: die Oberfläche zeigt ein graues Feld
	Da                    // dekodiert
	Kaputt                // fehlt oder unlesbar: graues Feld, kein neuer Versuch bis Vergiss
)

// ErrZuGross: Die Datei im Vorschau-Cache ist größer, als ein Vorschaubild sein kann.
var ErrZuGross = errors.New("libview: Vorschaubild zu groß")

// stueck ist ein Vorschaubild im Speicher.
type stueck struct {
	bild    *image.NRGBA
	zustand Zustand
	zuletzt uint64 // Zähler der letzten Abfrage (Verdrängen nach Alter)
	runde   uint64 // Runde der letzten Abfrage (Verwerfen weggerollter Aufträge, F-669)
	groesse int64  // belegte Bytes
}

// Vorschauen hält dekodierte Vorschaubilder und lädt fehlende in einer Goroutine nach (ADR-0036 Punkt 6). Die
// Oberfläche fragt in jedem Bild nur nach den sichtbaren (Bild) und wartet nie auf die Platte; ist ein Bild fertig,
// weckt die Goroutine das Fenster (wecke). Nebenläufig sicher.
type Vorschauen struct {
	mu      sync.Mutex
	lade    func(rel string) (*image.NRGBA, error)
	wecke   func()
	grenze  int64
	belegt  int64
	takt    uint64
	runde   uint64 // zählt Runde hoch: je gezeichnetem Bild einmal
	stuecke map[string]*stueck
	warte   []string // Aufträge in der Reihenfolge der Anforderung; die jüngste zuerst geladen (gerade sichtbar)
	laeuft  bool
}

// NeueVorschauen legt den Speicher an: lade liest ein Vorschaubild (in der Goroutine), wecke weckt die Oberfläche (darf
// nil sein), grenze ist die Speichergrenze in Bytes.
func NeueVorschauen(lade func(rel string) (*image.NRGBA, error), wecke func(), grenze int64) *Vorschauen {
	return &Vorschauen{lade: lade, wecke: wecke, grenze: grenze, stuecke: map[string]*stueck{}}
}

// Bild liefert das Vorschaubild der Aufnahme rel, wenn es da ist. Fehlt es, wird es angefordert und kommt mit einem
// späteren Bild (Zustand Laedt); jede Abfrage zählt als Gebrauch für das Verdrängen.
func (v *Vorschauen) Bild(rel string) (*image.NRGBA, Zustand) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.takt++
	s := v.stuecke[rel]
	if s == nil {
		s = &stueck{zustand: Laedt}
		v.stuecke[rel] = s
		v.warte = append(v.warte, rel)
		if !v.laeuft {
			v.laeuft = true
			go v.arbeite()
		}
	}
	s.zuletzt, s.runde = v.takt, v.runde
	return s.bild, s.zustand
}

// Runde meldet ein neues Bild der Oberfläche: Sie fragt danach die gerade sichtbaren Vorschauen ab (Bild). Ein Auftrag,
// den seit zwei Runden niemand mehr abgefragt hat (aus dem Bild gerollt), wird nicht mehr geladen; fragt die
// Oberfläche später wieder, wird er neu angefordert (F-669). Ohne Neuzeichnen verfällt nichts.
func (v *Vorschauen) Runde() {
	v.mu.Lock()
	v.runde++
	v.mu.Unlock()
}

// Vergiss wirft das Bild von rel weg, etwa nach dem eigenen Speichern (die Vorschau ist neu geschrieben); die nächste
// Abfrage lädt es neu. Ein gerade laufendes Laden wird verworfen.
func (v *Vorschauen) Vergiss(rel string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if s := v.stuecke[rel]; s != nil {
		v.belegt -= s.groesse
		delete(v.stuecke, rel)
	}
}

// Belegt ist die Zahl der Bytes dekodierter Bilder im Speicher.
func (v *Vorschauen) Belegt() int64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.belegt
}

// veraltetNach ist die Zahl der Runden ohne Abfrage, nach der ein wartender Auftrag verfällt; eine Runde Spiel für ein
// Bild, das gerade zwischen Runde und Abfrage steht.
const veraltetNach = 1

// arbeite lädt die Aufträge nacheinander, bis keiner mehr wartet.
func (v *Vorschauen) arbeite() {
	for {
		v.mu.Lock()
		if len(v.warte) == 0 {
			v.laeuft = false
			v.mu.Unlock()
			return
		}
		rel := v.warte[len(v.warte)-1]
		v.warte = v.warte[:len(v.warte)-1]
		s := v.stuecke[rel]
		if s != nil && s.zustand == Laedt && s.runde+veraltetNach < v.runde {
			delete(v.stuecke, rel) // weggerollt: nicht laden, eine neue Abfrage fordert neu an (F-669)
			s = nil
		}
		v.mu.Unlock()
		if s == nil || s.zustand != Laedt {
			continue // inzwischen vergessen, verdrängt oder weggerollt
		}
		bild, err := v.lade(rel)
		v.mu.Lock()
		if v.stuecke[rel] == s { // nicht inzwischen vergessen
			if err != nil || bild == nil {
				s.zustand = Kaputt
			} else {
				s.bild, s.zustand = bild, Da
				s.groesse = int64(len(bild.Pix))
				v.belegt += s.groesse
				v.verdraenge(s)
			}
		}
		wecke := v.wecke
		v.mu.Unlock()
		if wecke != nil {
			wecke()
		}
	}
}

// verdraenge wirft die am längsten nicht abgefragten Bilder weg, bis die Grenze hält; neu (das gerade geladene) bleibt.
// Läuft unter v.mu.
func (v *Vorschauen) verdraenge(neu *stueck) {
	for v.belegt > v.grenze {
		var altRel string
		var alt *stueck
		for rel, s := range v.stuecke {
			if s == neu || s.zustand != Da {
				continue
			}
			if alt == nil || s.zuletzt < alt.zuletzt {
				altRel, alt = rel, s
			}
		}
		if alt == nil {
			return
		}
		v.belegt -= alt.groesse
		delete(v.stuecke, altRel)
	}
}

// LadeDatei liest ein Vorschaubild aus dem Cache: nur eine reguläre Datei (kein FIFO, kein Verweis), höchstens
// maxVorschauDatei groß und höchstens maxVorschauSeite Pixel je Seite, sonst ErrZuGross.
func LadeDatei(pfad string) (*image.NRGBA, error) {
	f, err := os.OpenFile(pfad, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fs.ErrInvalid
	}
	if fi.Size() > maxVorschauDatei {
		return nil, ErrZuGross
	}
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		return nil, err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > maxVorschauSeite || cfg.Height > maxVorschauSeite {
		return nil, ErrZuGross
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	bild, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	if n, ok := bild.(*image.NRGBA); ok && n.Rect.Min == (image.Point{}) {
		return n, nil
	}
	n := image.NewNRGBA(image.Rect(0, 0, bild.Bounds().Dx(), bild.Bounds().Dy()))
	draw.Draw(n, n.Rect, bild, bild.Bounds().Min, draw.Src)
	return n, nil
}

// LaderFuer ist die Ladefunktion für die Bibliothek b: das Vorschaubild aus dem Cache (library.VorschauPfad); fehlt
// es, schreibt die Bibliothek es aus der Aufnahme neu (library.ErzeugeVorschau, wie beim Ablegen), dann ein zweiter
// Versuch.
func LaderFuer(b *library.Bibliothek) func(rel string) (*image.NRGBA, error) {
	return func(rel string) (*image.NRGBA, error) {
		bild, err := LadeDatei(b.VorschauPfad(rel))
		if errors.Is(err, fs.ErrNotExist) {
			b.ErzeugeVorschau(rel)
			bild, err = LadeDatei(b.VorschauPfad(rel))
		}
		return bild, err
	}
}
