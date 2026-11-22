// SPDX-License-Identifier: GPL-3.0-or-later

package main

// Bilder des Corporate Designs und das PDF aus brand/corporate-design.md (früher cd.py).
//   → brand/cd/*.svg|png (Farbfelder, Schutzraum, Oberflächen-Beispiel)
//   → brand/corporate-design.pdf (über HTML, gedruckt mit Chromium im Headless-Modus, eigenes Profil unter tmp/;
//     Metadaten danach mit qpdf entfernt)
// Der Markdown-Umwandler kann genau das, was corporate-design.md benutzt: Überschriften, Absätze, Listen, Tabellen,
// Bilder, Zitate, Code, fett, Links.

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var cdFarben = [][3]string{ // Name, Hex, Schrift darauf
	{"grau-950", "#16181B", "#ECEAE5"}, {"grau-900", "#1E2124", "#ECEAE5"}, {"grau-800", "#2A2E33", "#ECEAE5"},
	{"grau-700", "#3A3F45", "#ECEAE5"}, {"grau-400", "#92989F", "#1E2124"}, {"grau-100", "#ECEAE5", "#1E2124"},
	{"gruen-300", "#86C2A6", "#1E2124"}, {"gruen-500", "#4F9479", "#1E2124"}, {"gruen-700", "#3B7A5F", "#FFFFFF"},
	{"orange-300", "#F9C894", "#1E2124"}, {"orange-500", "#F5A65B", "#1E2124"}, {"orange-700", "#A85F22", "#FFFFFF"},
	{"rot-500", "#E67873", "#1E2124"}, {"rot-700", "#B4433E", "#FFFFFF"},
}

func (b *bauer) farbfelder() (string, error) {
	const feld, abstand, jeZeile = 150, 12, 7
	var teile []string
	for i, f := range cdFarben {
		x, y := (i%jeZeile)*(feld+abstand), (i/jeZeile)*(feld+abstand)
		teile = append(teile, fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" rx="10" fill="%s" stroke="#3A3F45" stroke-width="1"/>`,
			x, y, feld, feld, f[1]))
		for _, z := range []struct {
			text string
			dy   int
		}{{f[0], 36}, {f[1], 14}} {
			p, _, err := b.schrift.textAlsPfad(z.text, float64(x+12), float64(y+feld-z.dy), 17, f[2], 0.04)
			if err != nil {
				return "", err
			}
			teile = append(teile, p)
		}
	}
	w := jeZeile*(feld+abstand) - abstand
	h := ((len(cdFarben)-1)/jeZeile+1)*(feld+abstand) - abstand
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">%s</svg>`+"\n",
		w, h, w, h, strings.Join(teile, "")), nil
}

var svgHuelle = regexp.MustCompile(`^<svg[^>]*>|</svg>\s*$`)

// schutzraum: Logo dunkel mit gestricheltem Schutzraum, ringsum eine halbe x-Höhe (50 Einheiten).
func (b *bauer) schutzraum() (string, error) {
	var r [4]float64
	for i, t := range strings.Fields(rahmen) {
		v, err := strconv.ParseFloat(t, 64)
		if err != nil {
			return "", err
		}
		r[i] = v
	}
	innen := [4]float64{r[0] + 10, r[1] + 10, r[2] - 20, r[3] - 20}
	const rand = 50
	aussen := [4]float64{innen[0] - rand, innen[1] - rand, innen[2] + 2*rand, innen[3] + 2*rand}
	l, err := b.logo("dunkel", true, false)
	if err != nil {
		return "", err
	}
	inhalt := svgHuelle.ReplaceAllString(l, "")
	z := pyZahl
	vb := z(aussen[0]-20) + " " + z(aussen[1]-20) + " " + z(aussen[2]+40) + " " + z(aussen[3]+40)
	return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="` + vb + `">` +
		`<rect x="` + z(aussen[0]-20) + `" y="` + z(aussen[1]-20) + `" width="` + z(aussen[2]+40) + `" height="` + z(aussen[3]+40) + `" fill="#1E2124"/>` +
		`<rect x="` + z(aussen[0]) + `" y="` + z(aussen[1]) + `" width="` + z(aussen[2]) + `" height="` + z(aussen[3]) + `" fill="none" ` +
		`stroke="#F5A65B" stroke-width="3" stroke-dasharray="12 8"/>` +
		`<rect x="` + z(innen[0]) + `" y="` + z(innen[1]) + `" width="` + z(innen[2]) + `" height="` + z(innen[3]) + `" fill="none" ` +
		`stroke="#92989F" stroke-width="1.5" stroke-dasharray="4 6"/>` +
		inhalt + "</svg>\n", nil
}

const uiHTML = `<!doctype html><html><head><meta charset="utf-8"><style>
body{margin:0;background:#1E2124;color:#ECEAE5;font:15px/1.5 'Open Sans',sans-serif;padding:20px}
.ui{background:#2A2E33;border:1px solid #3A3F45;border-radius:10px;padding:16px;width:600px}
.l{display:flex;gap:8px;margin-bottom:14px;align-items:center}
.k{border:0;border-radius:6px;padding:7px 14px;font:inherit}
.haupt{background:#4F9479;color:#1E2124;font-weight:600}.zweit{background:none;color:#ECEAE5;border:1px solid #3A3F45}
.wz{background:#1E2124;color:#ECEAE5;border:1px solid #3A3F45}.wz.a{border-color:#4F9479;color:#86C2A6}
.fokus{outline:2px solid #F5A65B;outline-offset:2px}
.gefahr{background:none;color:#E67873;border:1px solid #E67873}
.h{border-left:4px solid #F5A65B;background:#1E2124;padding:8px 12px;border-radius:4px}.h b{color:#F9C894}
.aw{position:relative;height:100px;margin:14px 0;background:#16181B;border-radius:6px}
.aw div{position:absolute;left:30%;top:20px;width:40%;height:58px;border:2px solid #4F9479}
.aw i{position:absolute;width:12px;height:12px;border:3px solid #F5A65B}
.n{color:#92989F;margin-left:auto}
</style></head><body><div class="ui">
<div class="l"><button class="k wz a">Pfeil</button><button class="k wz">Kasten</button><button class="k wz fokus">Text</button>
<button class="k gefahr">Schwärzen</button><span class="n">Fokus: oranger Ring</span></div>
<div class="aw"><div></div>
<i style="left:calc(30% - 6px);top:14px;border-right:0;border-bottom:0"></i>
<i style="left:calc(70% - 8px);top:14px;border-left:0;border-bottom:0"></i>
<i style="left:calc(30% - 6px);top:74px;border-right:0;border-top:0"></i>
<i style="left:calc(70% - 8px);top:74px;border-left:0;border-top:0"></i></div>
<div class="h"><b>Hinweis:</b> Geschwärzte Bereiche lassen sich nach dem Speichern nicht wiederherstellen.</div>
<div class="l" style="margin:14px 0 0"><button class="k haupt">Speichern</button><button class="k zweit">Abbrechen</button>
<span class="n">1920 × 1080 · 100 %</span></div></div></body></html>`

func (b *bauer) chromium(args ...string) error {
	profil := filepath.Join(b.wurzel(), "tmp", "chromium-profil") // eigenes, leeres Profil — nie das des Nutzers
	if err := os.MkdirAll(profil, 0o755); err != nil {
		return err
	}
	c := exec.Command("chromium", append([]string{"--headless", "--disable-gpu", "--no-first-run",
		"--no-default-browser-check", "--user-data-dir=" + profil}, args...)...)
	var fehler bytes.Buffer
	c.Stderr = &fehler
	if err := c.Run(); err != nil {
		// Ohne Fehlerausgabe wäre die Ursache (z. B. zu langer Socket-Pfad) nicht zu sehen.
		zeilen := strings.Split(strings.TrimRight(fehler.String(), "\n"), "\n")
		if len(zeilen) > 20 {
			zeilen = zeilen[len(zeilen)-20:]
		}
		auszug := strings.Join(zeilen, "\n")
		if strings.TrimSpace(auszug) == "" {
			auszug = "(keine Fehlerausgabe)"
		}
		return fmt.Errorf("Chromium ist gescheitert (%v). Letzte Ausgabe:\n%s", err, auszug)
	}
	return nil
}

func (b *bauer) wurzel() string { return filepath.Dir(b.marke) }

func dateiURI(pfad string) string { return (&url.URL{Scheme: "file", Path: pfad}).String() }

// ---- Markdown → HTML (nur der benutzte Teil, wie bisher cd.py) ----

// escape wie Pythons html.escape (quote=True: auch " und ').
func escape(t string, anf bool) string {
	t = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(t)
	if anf {
		t = strings.NewReplacer(`"`, "&quot;", "'", "&#x27;").Replace(t)
	}
	return t
}

var (
	reCode   = regexp.MustCompile("`([^`]+)`")
	reFett   = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reBild   = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	reLink   = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	reTitel  = regexp.MustCompile(`^(#{1,4}) (.*)`)
	rePunkt  = regexp.MustCompile(`^(- |\d+\. )`)
	reZahl   = regexp.MustCompile(`^\d+\. `)
	reAbsatz = regexp.MustCompile("^(#|\\||- |\\d+\\. |>|```)")
)

func inline(t string) string {
	t = escape(t, false)
	t = reCode.ReplaceAllString(t, "<code>${1}</code>")
	t = reFett.ReplaceAllString(t, "<strong>${1}</strong>")
	t = reBild.ReplaceAllString(t, `<img src="${2}" alt="${1}">`)
	t = reLink.ReplaceAllString(t, `<a href="${2}">${1}</a>`)
	return t
}

func zellen(z string) []string {
	teile := strings.Split(strings.Trim(z, "|"), "|")
	for i := range teile {
		teile[i] = strings.TrimSpace(teile[i])
	}
	return teile
}

func mdZuHTML(md string) (string, error) {
	zeilen := strings.Split(md, "\n")
	var aus []string
	for i := 0; i < len(zeilen); {
		z := zeilen[i]
		switch {
		case strings.HasPrefix(z, "```"):
			j := i + 1
			for j < len(zeilen) && !strings.HasPrefix(zeilen[j], "```") {
				j++
			}
			if j == len(zeilen) {
				return "", fmt.Errorf("Code-Zaun ab Zeile %d wird nicht geschlossen", i+1)
			}
			aus = append(aus, "<pre><code>"+escape(strings.Join(zeilen[i+1:j], "\n"), true)+"</code></pre>")
			i = j + 1
		case reTitel.MatchString(z):
			m := reTitel.FindStringSubmatch(z)
			n := len(m[1])
			aus = append(aus, fmt.Sprintf("<h%d>%s</h%d>", n, inline(m[2]), n))
			i++
		case strings.HasPrefix(z, "|"):
			kopf := zellen(z)
			i += 2
			var b strings.Builder
			b.WriteString("<table><thead><tr>")
			for _, c := range kopf {
				b.WriteString("<th>" + inline(c) + "</th>")
			}
			b.WriteString("</tr></thead><tbody>")
			for i < len(zeilen) && strings.HasPrefix(zeilen[i], "|") {
				b.WriteString("<tr>")
				for _, c := range zellen(zeilen[i]) {
					b.WriteString("<td>" + inline(c) + "</td>")
				}
				b.WriteString("</tr>")
				i++
			}
			b.WriteString("</tbody></table>")
			aus = append(aus, b.String())
		case rePunkt.MatchString(z):
			tag := "ul"
			if reZahl.MatchString(z) {
				tag = "ol"
			}
			var punkte []string
			for i < len(zeilen) && (rePunkt.MatchString(zeilen[i]) || strings.HasPrefix(zeilen[i], "  ")) {
				if strings.HasPrefix(zeilen[i], "  ") {
					punkte[len(punkte)-1] += " " + strings.TrimSpace(zeilen[i])
				} else {
					punkte = append(punkte, rePunkt.ReplaceAllString(zeilen[i], ""))
				}
				i++
			}
			var b strings.Builder
			b.WriteString("<" + tag + ">")
			for _, p := range punkte {
				b.WriteString("<li>" + inline(p) + "</li>")
			}
			b.WriteString("</" + tag + ">")
			aus = append(aus, b.String())
		case strings.HasPrefix(z, ">"):
			var teil []string
			for i < len(zeilen) && strings.HasPrefix(zeilen[i], ">") {
				teil = append(teil, strings.TrimLeft(zeilen[i], "> "))
				i++
			}
			aus = append(aus, "<blockquote>"+inline(strings.Join(teil, " "))+"</blockquote>")
		case strings.TrimSpace(z) == "---":
			aus = append(aus, "<hr>")
			i++
		case strings.TrimSpace(z) != "":
			var teil []string
			for i < len(zeilen) && strings.TrimSpace(zeilen[i]) != "" && !reAbsatz.MatchString(zeilen[i]) {
				teil = append(teil, zeilen[i])
				i++
			}
			aus = append(aus, "<p>"+inline(strings.Join(teil, " "))+"</p>")
		default:
			i++
		}
	}
	return strings.Join(aus, "\n"), nil
}

const cdCSS = `
@page { size: A4; margin: 18mm 16mm; }
body { font: 10.5pt/1.5 'Open Sans', sans-serif; color: #1E2124; }
h1 { font-size: 22pt; margin: 0 0 4mm; } h2 { font-size: 15pt; margin: 8mm 0 3mm; border-bottom: 2px solid #4F9479;
padding-bottom: 1mm; break-after: avoid; } h3 { font-size: 12pt; margin: 5mm 0 2mm; break-after: avoid; }
table { border-collapse: collapse; width: 100%; margin: 2mm 0 4mm; font-size: 9.5pt; break-inside: avoid; }
th, td { border: 1px solid #C9CCD0; padding: 1.5mm 2mm; text-align: left; vertical-align: top; }
th { background: #ECEAE5; } code { font: 9pt monospace; background: #F1F0EC; padding: 0 1mm; border-radius: 2px; }
pre { background: #F1F0EC; padding: 3mm; border-radius: 3px; } pre code { background: none; padding: 0; }
img { max-width: 100%; display: block; margin: 3mm 0; break-inside: avoid; }
blockquote { border-left: 4px solid #F5A65B; margin: 3mm 0; padding: 1mm 4mm; background: #FBF6EF; }
a { color: #3B7A5F; } hr { border: 0; border-top: 1px solid #C9CCD0; margin: 6mm 0; }
`

// pdf druckt corporate-design.md über Chromium und entfernt mit qpdf die Metadaten (Kernprinzip 3). Gelöscht wird
// nichts: Zwischenstände werden umbenannt oder beim nächsten Lauf überschrieben.
func (b *bauer) pdf() error {
	md, err := os.ReadFile(filepath.Join(b.marke, "corporate-design.md"))
	if err != nil {
		return err
	}
	rumpf, err := mdZuHTML(string(md))
	if err != nil {
		return err
	}
	seite := `<!doctype html><html lang="de"><head><meta charset="utf-8"><title>Schnapp Corporate Design</title>` +
		`<style>` + cdCSS + `</style></head><body>` + rumpf + `</body></html>`
	zwischen := filepath.Join(b.marke, "corporate-design.tmp.html") // neben der md, damit relative Bildpfade stimmen
	if err := os.WriteFile(zwischen, []byte(seite), 0o644); err != nil {
		return err
	}
	ablage := filepath.Join(b.wurzel(), "tmp", "brand")
	if err := os.MkdirAll(ablage, 0o755); err != nil {
		return err
	}
	// Zwischenstand aufheben statt löschen, auch bei Fehler.
	defer func() { _ = os.Rename(zwischen, filepath.Join(ablage, "corporate-design.html")) }()
	roh, sauber := filepath.Join(ablage, "corporate-design.roh.pdf"), filepath.Join(ablage, "corporate-design.pdf")
	// Ein altes Roh-PDF aus einem Vorlauf darf nicht übernommen werden (F-29): vor dem Chromium-Aufruf beiseitelegen
	// (fester Name) und danach nur auf Existenz prüfen. Bei Fehlschlag kommt es zurück.
	alt := filepath.Join(ablage, "corporate-design.roh.alt")
	vorher := false
	if st, err := os.Stat(roh); err == nil && st.Mode().IsRegular() {
		if err := os.Rename(roh, alt); err != nil {
			return err
		}
		vorher = true
	}
	erfolg := false
	defer func() {
		if vorher && !erfolg {
			_ = os.Rename(alt, roh)
		}
	}()
	if err := b.chromium("--no-pdf-header-footer", "--print-to-pdf="+roh, dateiURI(zwischen)); err != nil {
		return err
	}
	if _, err := os.Stat(roh); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Chromium hat in diesem Lauf kein PDF nach %s geschrieben; brand/corporate-design.pdf bleibt unverändert", roh)
	}
	// Keine Spuren (Kernprinzip 3): Chromium schreibt Programm- und Systemnamen in Creator/Producer — entfernen.
	if err := fuehreAus("qpdf", "--remove-info", "--remove-metadata", roh, sauber); err != nil {
		return err
	}
	if err := os.Rename(sauber, filepath.Join(b.marke, "corporate-design.pdf")); err != nil { // erst jetzt ersetzen
		return err
	}
	erfolg = true
	return nil
}

func (b *bauer) cd() error {
	cd := filepath.Join(b.marke, "cd")
	if err := os.MkdirAll(cd, 0o755); err != nil {
		return err
	}
	ff, err := b.farbfelder()
	if err != nil {
		return err
	}
	sr, err := b.schutzraum()
	if err != nil {
		return err
	}
	for _, e := range []struct{ name, inhalt string }{{"farben", ff}, {"schutzraum", sr}} {
		p := filepath.Join(cd, e.name+".svg")
		if err := schreibe(p, e.inhalt); err != nil {
			return err
		}
		if err := png(p, filepath.Join(cd, e.name+".png"), 1100); err != nil {
			return err
		}
	}
	ui := filepath.Join(b.wurzel(), "tmp", "brand", "ui-beispiel.html")
	if err := schreibe(ui, uiHTML); err != nil {
		return err
	}
	if err := b.chromium("--hide-scrollbars", "--window-size=680,330", "--screenshot="+filepath.Join(cd, "ui-beispiel.png"), dateiURI(ui)); err != nil {
		return err
	}
	return b.pdf()
}
