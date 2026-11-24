// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

// Schnapp-Fernbedienung: spielt ein Drehbuch (eine Zeile je Befehl) gegen ein Programm im virtuellen KWin ab —
// Zeiger- und Tasteneingaben über org_kde_kwin_fake_input, Bildschirmfotos über die Aufnahmeschicht. Nur für
// Probeläufe (scripts/probe.sh --drehbuch), nie ausliefern: KWin winkt die Anmeldung der Schnittstelle ohne Prüfung
// durch, jedes Programm der Sitzung könnte also Eingaben einspeisen.
//
// Befehle: warte MS | zeiger X Y | klick [X Y] | doppelklick X Y | rechtsklick [X Y] | mittelklick [X Y] |
// ziehe X1 Y1 X2 Y2 [SCHRITTE] | ziehe-mitte … | taste KOMBI | druecke[-rechts|-mitte] [X Y] | bewege X Y [SCHRITTE] |
// loslassen | taste-halten KOMBI | taste-loslassen KOMBI (Halten, F-599; am Ende des Drehbuchs wird alles Gehaltene
// gelöst, auch nach einem Fehler; klick/ziehe/taste verweigern sich bei gehaltener eigener Taste, F-614) | schreibe TEXT | rad X Y SCHRITTE | foto NAME [BILDSCHIRM]
// Fotos landen als PNG in $FOTOS (Ordner wird angelegt). Exit: 0 durch, 1 Laufzeitfehler, 2 Fehler im Drehbuch.
package main

import (
	"context"
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Andiministrator/schnapp/internal/capture"
	"github.com/Andiministrator/schnapp/internal/probeguard"
)

// nurVirtuell lässt die Fernbedienung nur im virtuellen KWin von scripts/probe.sh zu (Prüfung gemeinsam mit dem
// Probe-Build von schnapp, internal/probeguard; Verify N-5, F-229). Das Werkzeug bleibt nach einem Lauf in tmp/probe
// liegen; von Hand in der echten Sitzung gestartet, würde es sich sonst mit dem echten KWin verbinden (KWin winkt die
// Anmeldung durch) und Eingaben einspeisen. Geprüft wird vor jedem Verbindungsversuch (Wayland und D-Bus für foto).
func nurVirtuell(getenv func(string) string) error { return probeguard.NurVirtuell(getenv) }

func main() { os.Exit(lauf(os.Args[1:], os.Stdout, os.Stderr)) }

func lauf(args []string, aus, fehler io.Writer) int {
	fs := flag.NewFlagSet("schnapp-fernbedienung", flag.ContinueOnError)
	fs.SetOutput(fehler)
	pause := fs.Int("pause", 80, "Wartezeit nach jedem Befehl in Millisekunden")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(fehler, "Aufruf: schnapp-fernbedienung [--pause MS] DREHBUCH")
		return 2
	}
	text, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(fehler, err)
		return 2
	}
	befehle, err := ParseDrehbuch(string(text))
	if err != nil {
		fmt.Fprintln(fehler, "Drehbuch:", err)
		return 2
	}
	if err := nurVirtuell(os.Getenv); err != nil {
		fmt.Fprintln(fehler, "Abbruch:", err)
		return 1
	}
	e, err := VerbindeEingabe()
	if err != nil {
		fmt.Fprintln(fehler, "Verbindung:", err)
		return 1
	}
	defer e.Close()
	fmt.Fprintf(aus, "fake_input Version %d verbunden, %d Befehle\n", e.Version, len(befehle))
	a := &abspieler{e: e, fotos: os.Getenv("FOTOS"), aus: aus}
	return abspielen(a, befehle, time.Duration(*pause)*time.Millisecond, e.Abgleich, aus, fehler)
}

// abspielen führt die Befehle der Reihe nach aus und gleicht nach jedem mit dem Compositor ab (abgleich; nil = nie).
// Was am Ende noch gehalten wird, löst es immer, auch nach einem Fehler, sonst bliebe der Sitz des virtuellen KWin
// verklemmt (Taste „gedrückt“). Exit 0 nur, wenn alle Befehle gelangen; ein Befehl, der das Halten verletzt (klick bei
// gehaltener Maustaste, taste mit gehaltener Taste), ist ein Fehler, kein stilles Lösen (F-614).
func abspielen(a *abspieler, befehle []*Befehl, pause time.Duration, abgleich func() error, aus, fehler io.Writer) int {
	defer func() {
		if err := a.alleLoslassen(); err != nil {
			fmt.Fprintln(fehler, "Loslassen am Ende:", err)
		}
	}()
	for _, b := range befehle {
		fmt.Fprintf(aus, "%3d: %s\n", b.Zeile, b.Text)
		if err := a.fuehreAus(b); err != nil {
			fmt.Fprintf(fehler, "Zeile %d (%s): %v\n", b.Zeile, b.Text, err)
			return 1
		}
		if abgleich != nil {
			if err := abgleich(); err != nil {
				fmt.Fprintf(fehler, "Zeile %d (%s): Compositor: %v\n", b.Zeile, b.Text, err)
				return 1
			}
		}
		time.Sleep(pause)
	}
	return 0
}

// eingeber ist, was der Abspieler von der Eingabe braucht (Eingabe erfüllt es; Tests setzen eine Attrappe ein).
type eingeber interface {
	Zeiger(x, y int) error
	Maus(knopf, zustand uint32) error
	Rad(wert float64) error
	Tastatur(code, zustand uint32) error
	Keysym(sym, zustand uint32) error
	Fassung() uint32
}

type abspieler struct {
	e     eingeber
	fotos string
	aus   io.Writer

	// Zustand des Haltens (F-599).
	zx, zy        int // zuletzt gesetzte Zeigerlage
	zeigerBekannt bool
	knoepfe       []uint32 // gehaltene Maustasten in Reihenfolge des Drückens
	tasten        []uint32 // gehaltene Tasten in Reihenfolge des Drückens
}

// kurz ist die Wartezeit zwischen den einzelnen Ereignissen einer Geste (Drücken, Bewegen, Loslassen).
const kurz = 15 * time.Millisecond

func (a *abspieler) fuehreAus(b *Befehl) error {
	z := b.Zahl
	switch b.Art {
	case "warte":
		time.Sleep(time.Duration(z[0]) * time.Millisecond)
	case "zeiger":
		return a.setzeZeiger(z[0], z[1])
	case "klick":
		return a.klicke(btnLinks, 1, z)
	case "rechtsklick":
		return a.klicke(btnRechts, 1, z)
	case "mittelklick":
		return a.klicke(btnMitte, 1, z)
	case "doppelklick":
		return a.klicke(btnLinks, 2, z)
	case "ziehe":
		return a.ziehe(btnLinks, z)
	case "ziehe-mitte":
		return a.ziehe(btnMitte, z)
	case "taste":
		return a.taste(b.Kombi)
	case "schreibe":
		return a.schreibe(b.Rest)
	case "rad":
		return a.rad(z[0], z[1], z[2])
	case "druecke":
		return a.druecke(btnLinks, z)
	case "druecke-rechts":
		return a.druecke(btnRechts, z)
	case "druecke-mitte":
		return a.druecke(btnMitte, z)
	case "bewege":
		return a.bewege(z)
	case "loslassen":
		return a.loslassen()
	case "taste-halten":
		return a.tasteHalten(b.Kombi)
	case "taste-loslassen":
		return a.tasteLoslassen(b.Kombi)
	case "foto":
		return a.foto(b.Rest, b.Extra)
	default:
		return fmt.Errorf("Befehl %q nicht umgesetzt", b.Art)
	}
	return nil
}

// gehaltenFehler meldet, wenn knopf von druecke gehalten wird: klick und ziehe würden ihn drücken und lösen, der
// gehaltene Zug endete im Compositor still und die Buchführung stimmte nicht mehr (F-614).
func (a *abspieler) gehaltenFehler(knopf uint32) error {
	if a.haelt(a.knoepfe, knopf) {
		return fmt.Errorf("Maustaste %#x wird gehalten (druecke): erst loslassen", knopf)
	}
	return nil
}

func (a *abspieler) klicke(knopf uint32, mal int, z []int) error {
	if err := a.gehaltenFehler(knopf); err != nil {
		return err
	}
	if len(z) == 2 {
		if err := a.setzeZeiger(z[0], z[1]); err != nil {
			return err
		}
		time.Sleep(kurz)
	}
	for i := 0; i < mal; i++ {
		if err := a.e.Maus(knopf, gedrueckt); err != nil {
			return err
		}
		time.Sleep(kurz)
		if err := a.e.Maus(knopf, los); err != nil {
			return err
		}
		time.Sleep(kurz)
	}
	return nil
}

func (a *abspieler) ziehe(knopf uint32, z []int) error {
	if err := a.gehaltenFehler(knopf); err != nil {
		return err
	}
	schritte := 10
	if len(z) == 5 {
		schritte = z[4]
	}
	x1, y1, x2, y2 := z[0], z[1], z[2], z[3]
	if err := a.setzeZeiger(x1, y1); err != nil {
		return err
	}
	time.Sleep(3 * kurz)
	if err := a.e.Maus(knopf, gedrueckt); err != nil {
		return err
	}
	time.Sleep(3 * kurz)
	for i := 1; i <= schritte; i++ {
		x := x1 + (x2-x1)*i/schritte
		y := y1 + (y2-y1)*i/schritte
		if err := a.setzeZeiger(x, y); err != nil {
			return err
		}
		time.Sleep(kurz)
	}
	time.Sleep(3 * kurz)
	return a.e.Maus(knopf, los)
}

func (a *abspieler) taste(k Kombi) error {
	// Eine gehaltene Taste (taste-halten) würde die Kombination lösen; dann stimmte die Buchführung nicht mehr (F-614).
	for _, c := range append(append([]uint32{}, k.Mods...), k.Taste) {
		if a.haelt(a.tasten, c) {
			return fmt.Errorf("Taste %d wird gehalten (taste-halten): erst taste-loslassen", c)
		}
	}
	for _, m := range k.Mods {
		if err := a.e.Tastatur(m, gedrueckt); err != nil {
			return err
		}
	}
	time.Sleep(kurz)
	if err := a.e.Tastatur(k.Taste, gedrueckt); err != nil {
		return err
	}
	time.Sleep(kurz)
	if err := a.e.Tastatur(k.Taste, los); err != nil {
		return err
	}
	for i := len(k.Mods) - 1; i >= 0; i-- {
		if err := a.e.Tastatur(k.Mods[i], los); err != nil {
			return err
		}
	}
	return nil
}

func (a *abspieler) schreibe(text string) error {
	for _, r := range text {
		if a.e.Fassung() >= 6 {
			s, err := Keysym(r)
			if err != nil {
				return err
			}
			if err := a.e.Keysym(s, gedrueckt); err != nil {
				return err
			}
			if err := a.e.Keysym(s, los); err != nil {
				return err
			}
		} else {
			code, umschalt, ok := ZeichenZuTaste(r)
			if !ok {
				return fmt.Errorf("Zeichen %q ohne Keysym-Unterstützung (Version %d) nicht schreibbar", r, a.e.Fassung())
			}
			k := Kombi{Taste: code}
			if umschalt {
				k.Mods = []uint32{keyUmschalt}
			}
			if err := a.taste(k); err != nil {
				return err
			}
		}
		time.Sleep(kurz)
	}
	return nil
}

func (a *abspieler) rad(x, y, schritte int) error {
	if err := a.setzeZeiger(x, y); err != nil {
		return err
	}
	time.Sleep(kurz)
	richtung := 15.0
	n := schritte
	if n < 0 {
		richtung, n = -15, -n
	}
	for i := 0; i < n; i++ {
		if err := a.e.Rad(richtung); err != nil {
			return err
		}
		time.Sleep(kurz)
	}
	return nil
}

// foto nimmt den Bildschirm unter dem Zeiger auf (bildschirm leer) oder den benannten und legt ihn als PNG ab.
func (a *abspieler) foto(name, bildschirm string) error {
	if err := nurVirtuell(os.Getenv); err != nil {
		return err
	}
	if a.fotos == "" {
		return fmt.Errorf("Umgebungsvariable FOTOS (Ordner für Fotos) nicht gesetzt")
	}
	if err := os.MkdirAll(a.fotos, 0o755); err != nil {
		return err
	}
	q, err := capture.NeuKWin()
	if err != nil {
		return err
	}
	defer q.Close()
	ctx, abbruch := context.WithTimeout(context.Background(), 20*time.Second)
	defer abbruch()
	var b *capture.Bild
	if bildschirm == "" {
		b, err = q.AktiverBildschirm(ctx, capture.Optionen{})
	} else {
		b, err = q.Bildschirm(ctx, bildschirm, capture.Optionen{})
	}
	if err != nil {
		return err
	}
	pfad := filepath.Join(a.fotos, name+".png")
	f, err := os.OpenFile(pfad, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if err := png.Encode(f, b.Pixel); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	r := b.Pixel.Bounds()
	fmt.Fprintf(a.aus, "     foto %s (%dx%d, Skala %.2f, Bildschirm %s)\n", pfad, r.Dx(), r.Dy(), b.Skala, b.Bildschirm)
	return nil
}

// setzeZeiger setzt den Zeiger und merkt sich die Lage (für bewege).
func (a *abspieler) setzeZeiger(x, y int) error {
	if err := a.e.Zeiger(x, y); err != nil {
		return err
	}
	a.zx, a.zy, a.zeigerBekannt = x, y, true
	return nil
}

func (a *abspieler) haelt(liste []uint32, code uint32) bool {
	for _, c := range liste {
		if c == code {
			return true
		}
	}
	return false
}

// druecke drückt eine Maustaste und hält sie, bis loslassen (oder das Ende des Drehbuchs) sie löst; mit X Y wandert
// der Zeiger vorher dorthin.
func (a *abspieler) druecke(knopf uint32, z []int) error {
	if a.haelt(a.knoepfe, knopf) {
		return fmt.Errorf("Maustaste wird schon gehalten")
	}
	if len(z) == 2 {
		if err := a.setzeZeiger(z[0], z[1]); err != nil {
			return err
		}
		time.Sleep(3 * kurz)
	}
	if err := a.e.Maus(knopf, gedrueckt); err != nil {
		return err
	}
	a.knoepfe = append(a.knoepfe, knopf)
	time.Sleep(3 * kurz)
	return nil
}

// bewege führt den Zeiger in SCHRITTE (Voreinstellung 10) gleichen Schritten von der zuletzt gesetzten Lage nach X Y,
// bei gehaltener Maustaste also ein Zug. Ohne bekannte Lage springt der Zeiger.
func (a *abspieler) bewege(z []int) error {
	schritte := 10
	if len(z) == 3 {
		schritte = z[2]
	}
	if !a.zeigerBekannt {
		return a.setzeZeiger(z[0], z[1])
	}
	x1, y1 := a.zx, a.zy
	for i := 1; i <= schritte; i++ {
		if err := a.setzeZeiger(x1+(z[0]-x1)*i/schritte, y1+(z[1]-y1)*i/schritte); err != nil {
			return err
		}
		time.Sleep(kurz)
	}
	return nil
}

// loslassen löst alle gehaltenen Maustasten (die zuletzt gedrückte zuerst).
func (a *abspieler) loslassen() error {
	if len(a.knoepfe) == 0 {
		return fmt.Errorf("keine Maustaste gehalten")
	}
	time.Sleep(3 * kurz)
	return a.loesseKnoepfe()
}

func (a *abspieler) loesseKnoepfe() error {
	var erster error
	for len(a.knoepfe) > 0 {
		k := a.knoepfe[len(a.knoepfe)-1]
		a.knoepfe = a.knoepfe[:len(a.knoepfe)-1]
		if err := a.e.Maus(k, los); err != nil && erster == nil {
			erster = err
		}
	}
	return erster
}

// tasteHalten drückt erst die Modifikatoren, dann die Taste und lässt sie gedrückt.
func (a *abspieler) tasteHalten(k Kombi) error {
	for _, c := range append(append([]uint32{}, k.Mods...), k.Taste) {
		if a.haelt(a.tasten, c) {
			return fmt.Errorf("Taste %d wird schon gehalten", c)
		}
		if err := a.e.Tastatur(c, gedrueckt); err != nil {
			return err
		}
		a.tasten = append(a.tasten, c)
		time.Sleep(kurz)
	}
	return nil
}

// tasteLoslassen löst die Taste und dann die Modifikatoren der Kombination, jede nur, wenn sie gehalten wird.
func (a *abspieler) tasteLoslassen(k Kombi) error {
	alle := append(append([]uint32{}, k.Mods...), k.Taste)
	for i := len(alle) - 1; i >= 0; i-- {
		c := alle[i]
		if !a.haelt(a.tasten, c) {
			return fmt.Errorf("Taste %d wird nicht gehalten", c)
		}
		if err := a.e.Tastatur(c, los); err != nil {
			return err
		}
		for j, h := range a.tasten {
			if h == c {
				a.tasten = append(a.tasten[:j], a.tasten[j+1:]...)
				break
			}
		}
		time.Sleep(kurz)
	}
	return nil
}

// alleLoslassen löst alles Gehaltene: erst Maustasten, dann Tasten in umgekehrter Reihenfolge des Drückens.
func (a *abspieler) alleLoslassen() error {
	erster := a.loesseKnoepfe()
	for len(a.tasten) > 0 {
		c := a.tasten[len(a.tasten)-1]
		a.tasten = a.tasten[:len(a.tasten)-1]
		if err := a.e.Tastatur(c, los); err != nil && erster == nil {
			erster = err
		}
	}
	return erster
}
