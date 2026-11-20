// SPDX-License-Identifier: GPL-3.0-or-later

//go:build probe

// Schnapp-Fernbedienung: spielt ein Drehbuch (eine Zeile je Befehl) gegen ein Programm im virtuellen KWin ab —
// Zeiger- und Tasteneingaben über org_kde_kwin_fake_input, Bildschirmfotos über die Aufnahmeschicht. Nur für
// Probeläufe (scripts/probe.sh --drehbuch), nie ausliefern: KWin winkt die Anmeldung der Schnittstelle ohne Prüfung
// durch, jedes Programm der Sitzung könnte also Eingaben einspeisen.
//
// Befehle: warte MS | zeiger X Y | klick [X Y] | doppelklick X Y | rechtsklick [X Y] | mittelklick [X Y] |
// ziehe X1 Y1 X2 Y2 [SCHRITTE] | ziehe-mitte … | taste KOMBI | schreibe TEXT | rad X Y SCHRITTE | foto NAME [BILDSCHIRM]
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
	"strings"
	"time"

	"github.com/Andiministrator/schnapp/capture"
)

// nurVirtuell lässt die Fernbedienung nur im virtuellen KWin von scripts/probe.sh zu: Wayland-Socket „schnapp-probe“ und
// Laufzeitordner tmp/probe/run. Das Werkzeug bleibt nach einem Lauf in tmp/probe liegen; von Hand in der echten Sitzung
// gestartet, würde es sich sonst mit dem echten KWin verbinden (KWin winkt die Anmeldung durch) und Eingaben einspeisen.
// Geprüft wird vor jedem Verbindungsversuch (Wayland und D-Bus für foto). Für den D-Bus-Weg zusätzlich: probe.sh
// setzt PROBE_INNEN=1 und den privaten Bus (dbus-run-session); dessen Adresse muss gesetzt sein und sich von der des
// äußeren Aufrufs (PROBE_AUSSENBUS) unterscheiden. Ohne Busadresse fiele godbus auf den Bus der echten Sitzung zurück
// (Verify N-5).
func nurVirtuell(getenv func(string) string) error {
	if getenv("PROBE_INNEN") != "1" {
		return fmt.Errorf("PROBE_INNEN ist nicht 1 — nur im virtuellen KWin von scripts/probe.sh")
	}
	if b := getenv("DBUS_SESSION_BUS_ADDRESS"); b == "" || b == getenv("PROBE_AUSSENBUS") {
		return fmt.Errorf("kein privater D-Bus (DBUS_SESSION_BUS_ADDRESS leer oder gleich dem äußeren Bus) — nur im virtuellen KWin von scripts/probe.sh")
	}
	if d := getenv("WAYLAND_DISPLAY"); d != "schnapp-probe" {
		return fmt.Errorf("WAYLAND_DISPLAY ist %q, erwartet \"schnapp-probe\" — nur im virtuellen KWin von scripts/probe.sh", d)
	}
	if r := getenv("XDG_RUNTIME_DIR"); !strings.HasSuffix(r, "/tmp/probe/run") {
		return fmt.Errorf("XDG_RUNTIME_DIR endet nicht auf /tmp/probe/run — nur im virtuellen KWin von scripts/probe.sh")
	}
	return nil
}

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
	for _, b := range befehle {
		fmt.Fprintf(aus, "%3d: %s\n", b.Zeile, b.Text)
		if err := a.fuehreAus(b); err != nil {
			fmt.Fprintf(fehler, "Zeile %d (%s): %v\n", b.Zeile, b.Text, err)
			return 1
		}
		if err := e.Abgleich(); err != nil {
			fmt.Fprintf(fehler, "Zeile %d (%s): Compositor: %v\n", b.Zeile, b.Text, err)
			return 1
		}
		time.Sleep(time.Duration(*pause) * time.Millisecond)
	}
	return 0
}

type abspieler struct {
	e     *Eingabe
	fotos string
	aus   io.Writer
}

// kurz ist die Wartezeit zwischen den einzelnen Ereignissen einer Geste (Drücken, Bewegen, Loslassen).
const kurz = 15 * time.Millisecond

func (a *abspieler) fuehreAus(b *Befehl) error {
	z := b.Zahl
	switch b.Art {
	case "warte":
		time.Sleep(time.Duration(z[0]) * time.Millisecond)
	case "zeiger":
		return a.e.Zeiger(z[0], z[1])
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
	case "foto":
		return a.foto(b.Rest, b.Extra)
	default:
		return fmt.Errorf("Befehl %q nicht umgesetzt", b.Art)
	}
	return nil
}

func (a *abspieler) klicke(knopf uint32, mal int, z []int) error {
	if len(z) == 2 {
		if err := a.e.Zeiger(z[0], z[1]); err != nil {
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
	schritte := 10
	if len(z) == 5 {
		schritte = z[4]
	}
	x1, y1, x2, y2 := z[0], z[1], z[2], z[3]
	if err := a.e.Zeiger(x1, y1); err != nil {
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
		if err := a.e.Zeiger(x, y); err != nil {
			return err
		}
		time.Sleep(kurz)
	}
	time.Sleep(3 * kurz)
	return a.e.Maus(knopf, los)
}

func (a *abspieler) taste(k Kombi) error {
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
		if a.e.Version >= 6 {
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
				return fmt.Errorf("Zeichen %q ohne Keysym-Unterstützung (Version %d) nicht schreibbar", r, a.e.Version)
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
	if err := a.e.Zeiger(x, y); err != nil {
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
