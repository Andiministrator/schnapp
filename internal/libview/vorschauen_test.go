// SPDX-License-Identifier: GPL-3.0-or-later

package libview

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/fileformat"
	"github.com/Andiministrator/schnapp/internal/library"
)

// warteBis fragt f, bis es wahr ist (höchstens 5 s).
func warteBis(t *testing.T, f func() bool) {
	t.Helper()
	ende := time.Now().Add(5 * time.Second)
	for !f() {
		if time.Now().After(ende) {
			t.Fatal("Zeit abgelaufen")
		}
		time.Sleep(time.Millisecond)
	}
}

// ADR-0036 Punkt 6: Ein fehlendes Bild kommt nicht sofort, sondern aus der Goroutine; die Oberfläche wird geweckt. Ein
// unlesbares bleibt grau (Kaputt), ohne neuen Versuch.
func TestVorschauenLaden(t *testing.T) {
	var mu sync.Mutex
	geladen := map[string]int{}
	tor := make(chan struct{})
	lade := func(rel string) (*image.NRGBA, error) {
		<-tor
		mu.Lock()
		geladen[rel]++
		mu.Unlock()
		if rel == "kaputt" {
			return nil, errors.New("kaputt")
		}
		return image.NewNRGBA(image.Rect(0, 0, 4, 2)), nil
	}
	var geweckt atomic.Int32
	v := NeueVorschauen(lade, func() { geweckt.Add(1) }, 1<<20)
	if b, z := v.Bild("a"); b != nil || z != Laedt {
		t.Fatalf("erste Abfrage: %v %v", b, z)
	}
	v.Bild("kaputt")
	close(tor)
	warteBis(t, func() bool { return geweckt.Load() == 2 })
	if b, z := v.Bild("a"); b == nil || z != Da || b.Rect.Dx() != 4 {
		t.Errorf("geladen: %v %v", b, z)
	}
	if b, z := v.Bild("kaputt"); b != nil || z != Kaputt {
		t.Errorf("kaputt: %v %v", b, z)
	}
	if v.Belegt() != 4*2*4 {
		t.Errorf("belegt %d", v.Belegt())
	}
	v.Bild("a")
	v.Bild("kaputt")
	mu.Lock()
	if geladen["a"] != 1 || geladen["kaputt"] != 1 {
		t.Errorf("mehrfach geladen: %v", geladen)
	}
	mu.Unlock()
	// Vergiss lädt neu (nach dem eigenen Speichern).
	v.Vergiss("a")
	if v.Belegt() != 0 {
		t.Errorf("belegt nach Vergiss %d", v.Belegt())
	}
	v.Bild("a")
	warteBis(t, func() bool { _, z := v.Bild("a"); return z == Da })
	mu.Lock()
	if geladen["a"] != 2 {
		t.Errorf("nach Vergiss nicht neu geladen: %v", geladen)
	}
	mu.Unlock()
}

// Über der Grenze fallen die am längsten nicht abgefragten heraus; das neue bleibt, auch wenn es allein die Grenze
// sprengt.
func TestVorschauenGrenze(t *testing.T) {
	lade := func(string) (*image.NRGBA, error) { return image.NewNRGBA(image.Rect(0, 0, 10, 10)), nil } // 400 Byte
	v := NeueVorschauen(lade, nil, 1000)
	hole := func(rel string) {
		t.Helper()
		v.Bild(rel)
		warteBis(t, func() bool { _, z := v.Bild(rel); return z == Da })
	}
	hole("a")
	hole("b")
	v.Bild("a") // a ist jünger abgefragt als b
	hole("c")   // 1200 > 1000: b fällt heraus
	if v.Belegt() != 800 {
		t.Errorf("belegt %d", v.Belegt())
	}
	v.mu.Lock()
	_, a := v.stuecke["a"]
	_, b := v.stuecke["b"]
	_, c := v.stuecke["c"]
	v.mu.Unlock()
	if !a || b || !c {
		t.Errorf("verdrängt: a %v b %v c %v", a, b, c)
	}
	klein := NeueVorschauen(lade, nil, 100)
	klein.Bild("x")
	warteBis(t, func() bool { _, z := klein.Bild("x"); return z == Da })
	if klein.Belegt() != 400 {
		t.Errorf("einzelnes großes Bild: %d", klein.Belegt())
	}
}

// Die jüngste Anforderung wird zuerst geladen: Das ist die gerade sichtbare Kachel, nicht eine, an der die Leiste
// längst vorbeigerollt ist.
func TestVorschauenJuengsteZuerst(t *testing.T) {
	var reihe []string
	var mu sync.Mutex
	tor := make(chan struct{})
	erst := true
	lade := func(rel string) (*image.NRGBA, error) {
		if erst {
			erst = false
			<-tor
		}
		mu.Lock()
		reihe = append(reihe, rel)
		mu.Unlock()
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	v := NeueVorschauen(lade, nil, 1<<20)
	v.Bild("erst")
	warteBis(t, func() bool { v.mu.Lock(); defer v.mu.Unlock(); return len(v.warte) == 0 })
	v.Bild("alt")
	v.Bild("neu")
	close(tor)
	warteBis(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(reihe) == 3 })
	if reihe[1] != "neu" || reihe[2] != "alt" {
		t.Errorf("Reihenfolge %v", reihe)
	}
}

// Ein Vergiss während des Ladens verwirft das Ergebnis (sonst zeigte die Leiste nach dem Speichern die alte Vorschau).
func TestVorschauenVergissWaehrendLaden(t *testing.T) {
	tor := make(chan struct{})
	fertig := make(chan struct{})
	lade := func(string) (*image.NRGBA, error) {
		<-tor
		return image.NewNRGBA(image.Rect(0, 0, 2, 2)), nil
	}
	v := NeueVorschauen(lade, func() { close(fertig) }, 1<<20)
	v.Bild("a")
	warteBis(t, func() bool { v.mu.Lock(); defer v.mu.Unlock(); return len(v.warte) == 0 })
	v.Vergiss("a")
	close(tor)
	<-fertig
	if v.Belegt() != 0 {
		t.Errorf("verworfenes Ergebnis belegt %d", v.Belegt())
	}
	v.mu.Lock()
	_, da := v.stuecke["a"]
	v.mu.Unlock()
	if da {
		t.Error("verworfenes Ergebnis steht im Speicher")
	}
}

func schreibePNG(t *testing.T, pfad string, w, h int) {
	t.Helper()
	f, err := os.Create(pfad)
	if err != nil {
		t.Fatal(err)
	}
	b := image.NewRGBA(image.Rect(0, 0, w, h))
	b.Set(0, 0, color.RGBA{R: 200, A: 255})
	if err := png.Encode(f, b); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// LadeDatei: ein gültiges Bild kommt als NRGBA mit Ursprung 0; zu große Bilder, Verweise, Ordner und Fehlendes nicht.
func TestLadeDatei(t *testing.T) {
	ordner := t.TempDir()
	gut := filepath.Join(ordner, "gut.png")
	schreibePNG(t, gut, 8, 4)
	b, err := LadeDatei(gut)
	if err != nil || b.Rect != image.Rect(0, 0, 8, 4) || b.NRGBAAt(0, 0).R != 200 {
		t.Fatalf("gut: %v %v", b, err)
	}
	gross := filepath.Join(ordner, "gross.png")
	schreibePNG(t, gross, maxVorschauSeite+1, 1)
	if _, err := LadeDatei(gross); !errors.Is(err, ErrZuGross) {
		t.Errorf("gross: %v", err)
	}
	verweis := filepath.Join(ordner, "verweis.png")
	if err := os.Symlink(gut, verweis); err != nil {
		t.Fatal(err)
	}
	if _, err := LadeDatei(verweis); err == nil {
		t.Error("Verweis gelesen")
	}
	if _, err := LadeDatei(ordner); err == nil {
		t.Error("Ordner gelesen")
	}
	if _, err := LadeDatei(filepath.Join(ordner, "fehlt.png")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("fehlt: %v", err)
	}
	riesig := filepath.Join(ordner, "riesig.png")
	if err := os.WriteFile(riesig, make([]byte, maxVorschauDatei+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LadeDatei(riesig); !errors.Is(err, ErrZuGross) {
		t.Errorf("riesig: %v", err)
	}
}

// LaderFuer liest aus dem Vorschau-Cache und erzeugt eine fehlende Vorschau aus der Aufnahme (ADR-0036 Punkt 6).
func TestLaderFuer(t *testing.T) {
	heim := t.TempDir()
	b := library.Neu(library.Orte{Ablage: filepath.Join(heim, "ablage"), Index: filepath.Join(heim, "daten", "index.jsonl"),
		Vorschau: filepath.Join(heim, "cache", "vorschau"), Papierkorb: filepath.Join(heim, "Trash")})
	bild := image.NewRGBA(image.Rect(0, 0, 600, 300))
	for i := range bild.Pix {
		bild.Pix[i] = 0xFF
	}
	d, err := document.Neu(bild)
	if err != nil {
		t.Fatal(err)
	}
	e, err := b.Ablegen(d, d.Original(), fileformat.Metadaten{Erstellt: time.Date(2026, 11, 24, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	lade := LaderFuer(b)
	v, err := lade(e.Pfad)
	if err != nil || v.Rect.Dx() != library.VorschauSeite {
		t.Fatalf("aus dem Cache: %v %v", v, err)
	}
	if err := os.Remove(b.VorschauPfad(e.Pfad)); err != nil {
		t.Fatal(err)
	}
	v, err = lade(e.Pfad)
	if err != nil || v.Rect.Dx() != library.VorschauSeite {
		t.Fatalf("neu erzeugt: %v %v", v, err)
	}
	if _, err := lade("2026-11/gibtsnicht.png"); err == nil {
		t.Error("fehlende Aufnahme ohne Fehler")
	}
}

// F-669: Aufträge, die seit zwei Runden niemand mehr abgefragt hat (weggerollt), lädt die Goroutine nicht mehr; was in
// der laufenden oder der vorigen Runde gefragt wurde, schon. Eine spätere Abfrage fordert neu an.
func TestVorschauenWeggerollteVerfallen(t *testing.T) {
	var reihe []string
	var mu sync.Mutex
	tor := make(chan struct{})
	erst := true
	lade := func(rel string) (*image.NRGBA, error) {
		if erst {
			erst = false
			<-tor
		}
		mu.Lock()
		reihe = append(reihe, rel)
		mu.Unlock()
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	v := NeueVorschauen(lade, nil, 1<<20)
	v.Runde()
	v.Bild("blockiert")
	warteBis(t, func() bool { v.mu.Lock(); defer v.mu.Unlock(); return len(v.warte) == 0 })
	v.Bild("weg1") // Runde 1, danach weggerollt
	v.Bild("weg2")
	v.Runde()
	v.Bild("vorige") // Runde 2: noch gültig in Runde 3
	v.Runde()
	v.Bild("sichtbar") // Runde 3
	close(tor)
	warteBis(t, func() bool { v.mu.Lock(); defer v.mu.Unlock(); return !v.laeuft })
	mu.Lock()
	got := append([]string(nil), reihe...)
	mu.Unlock()
	if len(got) != 3 || got[0] != "blockiert" || got[1] != "sichtbar" || got[2] != "vorige" {
		t.Fatalf("geladen %v, erwartet [blockiert sichtbar vorige]", got)
	}
	if _, z := v.Bild("weg1"); z != Laedt {
		t.Errorf("weggerollter Auftrag: Zustand %v nach neuer Abfrage", z)
	}
	warteBis(t, func() bool { _, z := v.Bild("weg1"); return z == Da })
}

// Ohne neue Runde (die Oberfläche zeichnet nicht) verfällt nichts.
func TestVorschauenOhneRundeVerfaelltNichts(t *testing.T) {
	var n atomic.Int32
	tor := make(chan struct{})
	lade := func(rel string) (*image.NRGBA, error) {
		if rel == "a" {
			<-tor
		}
		n.Add(1)
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	v := NeueVorschauen(lade, nil, 1<<20)
	v.Bild("a")
	warteBis(t, func() bool { v.mu.Lock(); defer v.mu.Unlock(); return len(v.warte) == 0 })
	v.Bild("b")
	v.Bild("c")
	close(tor)
	warteBis(t, func() bool { return n.Load() == 3 })
}

// F-677: Pruefe wirft eine geladene Vorschau weg, deren Datei sich geändert hat (fremdes Speichern, etwa eine Schwärzung
// in einem anderen Fenster); unveränderte bleiben. Die nächste Abfrage lädt neu.
func TestVorschauenPruefeErkenntFremdesSpeichern(t *testing.T) {
	var mu sync.Mutex
	staende := map[string]string{"a": "1", "b": "1"}
	var geladen atomic.Int32
	lade := func(rel string) (*image.NRGBA, error) {
		geladen.Add(1)
		return image.NewNRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	v := NeueVorschauen(lade, nil, 1<<20)
	v.SetzeStand(func(rel string) string { mu.Lock(); defer mu.Unlock(); return staende[rel] })
	v.Bild("a")
	v.Bild("b")
	warteBis(t, func() bool { _, za := v.Bild("a"); _, zb := v.Bild("b"); return za == Da && zb == Da })
	mu.Lock()
	staende["a"] = "2" // eine andere Instanz hat a gespeichert
	mu.Unlock()
	v.Pruefe()
	warteBis(t, func() bool { v.mu.Lock(); defer v.mu.Unlock(); return !v.pruefen })
	if _, z := v.Bild("a"); z != Laedt {
		t.Errorf("geänderte Vorschau bleibt: Zustand %v", z)
	}
	if _, z := v.Bild("b"); z != Da {
		t.Errorf("unveränderte Vorschau verworfen: Zustand %v", z)
	}
	warteBis(t, func() bool { _, z := v.Bild("a"); return z == Da })
	if n := geladen.Load(); n != 3 {
		t.Errorf("%d Ladevorgänge, erwartet 3 (a, b, a neu)", n)
	}
}
