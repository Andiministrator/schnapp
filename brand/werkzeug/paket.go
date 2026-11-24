// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Logo-Paket nach brand/logo/ (früher paket.py). Rastern mit rsvg-convert, favicon.ico mit ImageMagick.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	versatz        = map[rune]int{'h': 30, 'p': -31}                 // h tiefer, pp höher: ragen oben und unten gleich weit heraus
	fuss, groesse  = 241.0, 24.0                                     // Grundlinie und Größe der Unterzeile
	sc             = [2]int{40 - 11, 40 + lauf + bw + 11}            // Außenkanten von s und c
	na             = [2]int{40 + 3*lauf - 11, 40 + 4*lauf + bw + 11} // Außenkanten von n und a
	symbolGroessen = []int{16, 22, 24, 32, 48, 64, 128, 180, 192, 256, 512, 1024}
	logoBreiten    = []int{300, 600, 1200, 2400}
	fassung        = map[string]farben{"dunkel": dunkel, "hell": hell}
	// Einfarbig (Stempel, Druck, Prägung): alles in einer Farbe; das Auge bleibt als Ring mit Pupille erhalten.
	einfarbig = map[string]farben{
		"schwarz": {"grund": "#FFFFFF", "schrift": "#000000", "akzent": "#000000", "pupille": "#000000", "weiss": "#FFFFFF"},
		"weiss":   {"grund": "#000000", "schrift": "#FFFFFF", "akzent": "#FFFFFF", "pupille": "#FFFFFF", "weiss": "none"},
	}
	unterzeileFarbe = map[string]string{"dunkel": neben["dunkel"], "hell": neben["hell"], "schwarz": "#000000", "weiss": "#FFFFFF"}
)

// rahmen ist der Ausschnitt des Logos: Inhalt x 29–823, y 58–242, 10 Rand.
const rahmen = "19 48 814 204"

func svgDatei(vb, inhalt string, breite, hoehe int) string {
	g := ""
	if breite > 0 {
		g = fmt.Sprintf(` width="%d" height="%d"`, breite, hoehe)
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s"%s>%s</svg>`+"\n", vb, g, inhalt)
}

type bauer struct {
	schrift *schrift
	marke   string // Ordner brand/
}

func (b *bauer) unterzeile(art string) (string, error) {
	gesperrt := func(text string, x0, x1 int) (string, error) {
		_, nat, err := b.schrift.textAlsPfad(text, 0, 0, groesse, "#000", 0)
		if err != nil {
			return "", err
		}
		sp := (float64(x1-x0) - nat) / float64(len([]rune(text))-1) / groesse
		p, _, err := b.schrift.textAlsPfad(text, float64(x0), fuss, groesse, unterzeileFarbe[art], sp)
		return p, err
	}
	a, err := gesperrt("SCREENSHOTS", sc[0], sc[1])
	if err != nil {
		return "", err
	}
	c, err := gesperrt("FOR LINUX", na[0], na[1])
	return a + c, err
}

func (b *bauer) logo(art string, mitUnterzeile, grund bool) (string, error) {
	f := fassung[art]
	if f == nil {
		f = einfarbig[art]
	}
	inhalt, _ := schriftzug(f, versatz)
	if mitUnterzeile {
		u, err := b.unterzeile(art)
		if err != nil {
			return "", err
		}
		inhalt += u
	}
	if grund {
		t := strings.Fields(rahmen)
		inhalt = fmt.Sprintf(`<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`, t[0], t[1], t[2], t[3], f["grund"]) + inhalt
	}
	return svgDatei(rahmen, inhalt, 0, 0), nil
}

// symbol: quadratisches Symbol, das s mit Auge; kachel=false nur das s, freigestellt.
func symbol(art string, kachel bool) string {
	f := fassung[art]
	s := sMitAuge(0, f, 1.0)
	if !kachel {
		return svgDatei("-16 84 108 132", s, 0, 0) // s mit Strich: x −11…87, y 89…211
	}
	grund := f["grund"]
	if art != "dunkel" {
		grund = "#ECEAE5"
	}
	// Kachel eckig seit Karte #217 (Variante B, Andi 24.11.2026; vorher rx 56).
	return svgDatei("0 0 256 256", fmt.Sprintf(`<rect x="8" y="8" width="240" height="240" fill="%s"/>`+
		`<g transform="translate(77,-75) scale(1.35)">%s</g>`, grund, s), 0, 0)
}

// sozial: Vorschaubild für GitHub/Links 1280×640, Logo dunkel mittig auf dunkelgrauem Grund.
func (b *bauer) sozial() (string, error) {
	zugText, _ := schriftzug(dunkel, versatz)
	u, err := b.unterzeile("dunkel")
	if err != nil {
		return "", err
	}
	return svgDatei("0 0 1280 640", fmt.Sprintf(`<rect width="1280" height="640" fill="%s"/>`+
		`<g transform="translate(150,197) scale(1.2)"><g transform="translate(-19,-48)">%s</g></g>`, dunkel["grund"], zugText+u),
		1280, 640), nil
}

func schreibe(pfad, text string) error {
	if err := os.MkdirAll(filepath.Dir(pfad), 0o755); err != nil {
		return err
	}
	return os.WriteFile(pfad, []byte(text), 0o644)
}

func fuehreAus(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func png(svgPfad, ziel string, breite int) error {
	if err := os.MkdirAll(filepath.Dir(ziel), 0o755); err != nil {
		return err
	}
	return fuehreAus("rsvg-convert", "-w", fmt.Sprint(breite), svgPfad, "-o", ziel)
}

func (b *bauer) paket() error {
	ziel := filepath.Join(b.marke, "logo")
	s, p := filepath.Join(ziel, "svg"), filepath.Join(ziel, "png")
	namen := []struct {
		mit  bool
		name string
	}{{true, "schnapp-logo"}, {false, "schnapp-logo-ohne-unterzeile"}}
	for _, art := range []string{"dunkel", "hell"} {
		for _, n := range namen {
			for _, grund := range []bool{false, true} {
				text, err := b.logo(art, n.mit, grund)
				if err != nil {
					return err
				}
				zusatz := ""
				if grund {
					zusatz = "-mit-grund"
				}
				if err := schreibe(filepath.Join(s, n.name+"-"+art+zusatz+".svg"), text); err != nil {
					return err
				}
			}
			for _, br := range logoBreiten {
				for _, zusatz := range []string{"", "-mit-grund"} {
					if err := png(filepath.Join(s, n.name+"-"+art+zusatz+".svg"),
						filepath.Join(p, fmt.Sprintf("%s-%s%s-%d.png", n.name, art, zusatz, br)), br); err != nil {
						return err
					}
				}
			}
		}
		if err := schreibe(filepath.Join(s, "schnapp-symbol-"+art+".svg"), symbol(art, true)); err != nil {
			return err
		}
		if err := schreibe(filepath.Join(s, "schnapp-s-"+art+".svg"), symbol(art, false)); err != nil {
			return err
		}
		for _, g := range symbolGroessen {
			if err := png(filepath.Join(s, "schnapp-symbol-"+art+".svg"), filepath.Join(p, fmt.Sprintf("schnapp-symbol-%s-%d.png", art, g)), g); err != nil {
				return err
			}
		}
		for _, g := range []int{64, 256, 512} {
			if err := png(filepath.Join(s, "schnapp-s-"+art+".svg"), filepath.Join(p, fmt.Sprintf("schnapp-s-%s-%d.png", art, g)), g); err != nil {
				return err
			}
		}
	}
	for _, art := range []string{"schwarz", "weiss"} {
		for _, n := range namen {
			text, err := b.logo(art, n.mit, false)
			if err != nil {
				return err
			}
			datei := filepath.Join(s, n.name+"-einfarbig-"+art+".svg")
			if err := schreibe(datei, text); err != nil {
				return err
			}
			if err := png(datei, filepath.Join(p, n.name+"-einfarbig-"+art+"-1200.png"), 1200); err != nil {
				return err
			}
		}
	}

	fav := filepath.Join(ziel, "favicon")
	favSVG := filepath.Join(fav, "favicon.svg")
	if err := schreibe(favSVG, symbol("dunkel", true)); err != nil {
		return err
	}
	var stufen []string
	for _, g := range []int{16, 32, 48} {
		t := filepath.Join(fav, fmt.Sprintf("tmp-%d.png", g))
		if err := png(favSVG, t, g); err != nil {
			return err
		}
		stufen = append(stufen, t)
	}
	if err := fuehreAus("magick", append(stufen, filepath.Join(fav, "favicon.ico"))...); err != nil {
		return err
	}
	for _, g := range []int{16, 32, 48} { // Zwischenstufen umbenennen statt löschen: in das ico gewandert, als PNG behalten
		if err := os.Rename(filepath.Join(fav, fmt.Sprintf("tmp-%d.png", g)), filepath.Join(fav, fmt.Sprintf("favicon-%d.png", g))); err != nil {
			return err
		}
	}
	for _, e := range []struct {
		name string
		g    int
	}{{"apple-touch-icon.png", 180}, {"icon-192.png", 192}, {"icon-512.png", 512}} {
		if err := png(favSVG, filepath.Join(fav, e.name), e.g); err != nil {
			return err
		}
	}
	manifest := "{\n  \"name\": \"Schnapp\",\n  \"short_name\": \"Schnapp\",\n  \"icons\": [\n" +
		"    {\"src\": \"icon-192.png\", \"sizes\": \"192x192\", \"type\": \"image/png\"},\n" +
		"    {\"src\": \"icon-512.png\", \"sizes\": \"512x512\", \"type\": \"image/png\"}\n  ],\n" +
		fmt.Sprintf("  \"theme_color\": \"%s\",\n  \"background_color\": \"%s\",\n", dunkel["grund"], dunkel["grund"]) +
		"  \"display\": \"standalone\"\n}\n"
	if err := schreibe(filepath.Join(fav, "site.webmanifest"), manifest); err != nil {
		return err
	}
	soz, err := b.sozial()
	if err != nil {
		return err
	}
	sozSVG := filepath.Join(ziel, "sozial", "schnapp-sozial-1280x640.svg")
	if err := schreibe(sozSVG, soz); err != nil {
		return err
	}
	return png(sozSVG, filepath.Join(ziel, "sozial", "schnapp-sozial-1280x640.png"), 1280)
}
