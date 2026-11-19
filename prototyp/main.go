// Wegwerf-Prototyp: Prüft, ob Gio für Schnapp trägt (ADR-0003, Kriterien in Nachtrag 1).
// Kein Produktcode; wird nach der Entscheidung gelöscht oder ersetzt.
//
// Aufrufe:
//
//	schnapp-prototyp                    Aufnahme über KWin → Overlay → Editor
//	schnapp-prototyp -bild datei.png    Bild laden → Overlay → Editor
//	schnapp-prototyp -testbild          synthetisches 4K-Bild → Editor
//	schnapp-prototyp -messen 10         synthetisches 4K-Bild, Pfeil biegt sich 10 s selbst, Bildrate ausgeben
//
// Die Aufnahme klappt nur, wenn die laufende Binärdatei in einer installierten .desktop-Datei mit
// X-KDE-DBUS-Restricted-Interfaces=org.kde.KWin.ScreenShot2 steht (siehe knowledge/topics/kwin-screenshot.md).
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"sync"
	"time"

	"gioui.org/app"
)

var programmStart = time.Now()

func main() {
	bildPfad := flag.String("bild", "", "PNG-Datei laden statt aufzunehmen")
	testbild := flag.Bool("testbild", false, "synthetisches 4K-Bild im Editor öffnen")
	messen := flag.Float64("messen", 0, "Sekunden automatischer Biege-Messung auf 4K-Testbild (0 = aus)")
	flag.Parse()
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	log.SetPrefix("[prototyp] ")
	log.Printf("Start (Unix-ns %d)", programmStart.UnixNano())

	type plan struct {
		bild    *image.RGBA
		ausgabe string // Wayland-Ausgabe für das Vollbildfenster; leer = KWin wählt
	}
	var (
		plaene []plan
		modus  = modusAuswahl
	)
	switch {
	case *messen > 0 || *testbild:
		plaene = []plan{{bild: erzeugeTestbild(3840, 2160)}}
		modus = modusEditor
	case *bildPfad != "":
		b, err := ladePNG(*bildPfad)
		if err != nil {
			log.Fatal(err)
		}
		plaene = []plan{{bild: b}}
	default:
		v, err := kwinVersion()
		if err != nil {
			log.Fatalf("ScreenShot2 nicht erreichbar: %v", err)
		}
		log.Printf("KWin ScreenShot2 Version %d", v)
		ausgaben, err := app.WaylandOutputs()
		if err != nil {
			log.Fatalf("Wayland-Ausgaben: %v", err)
		}
		log.Printf("%d Wayland-Ausgaben nach %v", len(ausgaben), time.Since(programmStart))
		// Je Bildschirm einzeln aufnehmen (native Auflösung): dann passt jedes Bild 1:1 zu seinem Overlay.
		for _, o := range ausgaben {
			a, err := aufnehmen("CaptureScreen", o.Name)
			if err != nil {
				log.Fatalf("Aufnahme %s: %v", o.Name, err)
			}
			log.Printf("Ausgabe %s: Modus %dx%d, Ganzzahl-Skala %d; Aufnahme %dx%d, scale %.2f, QImage-Format %d, %v",
				o.Name, o.Width, o.Height, o.Scale, a.bild.Rect.Dx(), a.bild.Rect.Dy(), a.skala, a.format, a.dauer)
			plaene = append(plaene, plan{bild: a.bild, ausgabe: o.Name})
		}
	}
	log.Printf("%d Bild(er) bereit nach %v", len(plaene), time.Since(programmStart))

	var gruppe []*ansicht
	for _, p := range plaene {
		a := neueAnsicht(p.bild, modus, time.Duration(*messen*float64(time.Second)))
		a.ausgabe = p.ausgabe
		gruppe = append(gruppe, a)
	}
	var fertig sync.WaitGroup
	for _, a := range gruppe {
		a.gruppe = gruppe
		fertig.Add(1)
		go func() {
			defer fertig.Done()
			if err := a.laufen(); err != nil {
				log.Print(err)
			}
		}()
	}
	go func() {
		fertig.Wait()
		os.Exit(0)
	}()
	app.Main()
}

// exitNachLog beendet das Programm; Protokollzeilen sind ungepuffert schon geschrieben.
func exitNachLog() { os.Exit(0) }

func ladePNG(pfad string) (*image.RGBA, error) {
	f, err := os.Open(pfad)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	if r, ok := b.(*image.RGBA); ok && r.Rect.Min == (image.Point{}) {
		return r, nil
	}
	r := image.NewRGBA(image.Rect(0, 0, b.Bounds().Dx(), b.Bounds().Dy()))
	for y := 0; y < r.Rect.Dy(); y++ {
		for x := 0; x < r.Rect.Dx(); x++ {
			r.Set(x, y, b.At(b.Bounds().Min.X+x, b.Bounds().Min.Y+y))
		}
	}
	return r, nil
}

// erzeugeTestbild malt ein synthetisches Bild mit Verläufen und Raster (keine echten Bildschirminhalte).
func erzeugeTestbild(breite, hoehe int) *image.RGBA {
	b := image.NewRGBA(image.Rect(0, 0, breite, hoehe))
	for y := 0; y < hoehe; y++ {
		for x := 0; x < breite; x++ {
			i := y*b.Stride + x*4
			r, g, bl := uint8(x*255/breite), uint8(y*255/hoehe), uint8(128)
			if x%120 < 2 || y%120 < 2 {
				r, g, bl = 40, 40, 40
			}
			b.Pix[i], b.Pix[i+1], b.Pix[i+2], b.Pix[i+3] = r, g, bl, 255
		}
	}
	return b
}

func init() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Schnapp-Prototyp (ADR-0003). Aufruf ohne Angaben: Aufnahme über KWin.")
		flag.PrintDefaults()
	}
}
