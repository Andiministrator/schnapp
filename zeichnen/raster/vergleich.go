// SPDX-License-Identifier: GPL-3.0-or-later

package raster

import (
	"fmt"
	"image"
	"image/draw"
	"math"
)

// Toleranz: Abweichung je Kanal (0..255), die als gleich gilt (Rundung beim Mischen).
const Toleranz = 2

// MittelSchwelle: größte erlaubte Abweichung je Kanal nach 3x3-Mittelung. Gemessen 18.11.2026: echte
// Paare (eigener Rasterer gegen rsvg) höchstens 9, Fehler an dünnen Strichen (fehlend, andersfarbig, zu breit) ab 68,
// eine Kante um ein volles Pixel verschoben rund ein Drittel des Kontrasts (48 bei Kontrast 145). Ein halbes Pixel
// verschiebt das Mittel um ein Sechstel des Kontrasts; 32 lässt das bis Kontrast 190 zu, 16 schon ab 96 nicht mehr.
const MittelSchwelle = 32

// Vergleich: Ergebnis eines Bildvergleichs.
type Vergleich struct {
	Pixel      int // Pixel im Bild
	Anders     int // Pixel mit irgendeiner Abweichung
	UeberTol   int // Pixel mit Abweichung über Toleranz
	Ueber32    int // Pixel mit Abweichung über 32
	Groesste   int // größte Abweichung eines Kanals
	Verstoesse int // Pixel mit Abweichung über Toleranz, die nicht in beiden Bildern an einer Kante liegen
	Mittel     int // größte Abweichung nach 3x3-Mittelung
	MittelZu   int // Pixel, deren 3x3-Mittel um mehr als MittelSchwelle abweicht (zählen als Verstöße)
	Beispiele  []image.Point
}

func vormultipliziert(b image.Image) *image.RGBA {
	r := image.NewRGBA(b.Bounds())
	draw.Draw(r, r.Bounds(), b, b.Bounds().Min, draw.Src)
	return r
}

// Vergleichen misst zwei gleich große Bilder gegeneinander (vormultipliziertes RGBA).
//
// Maßstab "höchstens ein Mischpixel" , zwei Kriterien:
//  1. Weicht ein Pixel um mehr als die Toleranz ab, muss es in beiden Bildern an einer Kante liegen, also in beiden
//     Bildern eine nicht einfarbige 3x3-Nachbarschaft haben. Flächen müssen damit gleich sein, und eine Kante darf
//     nicht um zwei Pixel wandern.
//  2. Das 3x3-Mittel jedes Pixels darf um höchstens MittelSchwelle abweichen. Das fängt, was Kriterium 1 an Kanten
//     durchlässt: dünne Striche (nur aus Kantenpixeln), die fehlen, andersfarbig oder zu breit sind, und eine Kante,
//     die um ein volles Pixel wandert. Erlaubt bleibt, was Kantenglättung unterscheidet (Bruchteile eines Pixels).
func Vergleichen(a, b image.Image) (Vergleich, error) {
	if a.Bounds().Size() != b.Bounds().Size() {
		return Vergleich{}, fmt.Errorf("Größen verschieden: %v gegen %v", a.Bounds().Size(), b.Bounds().Size())
	}
	pa, pb := vormultipliziert(a), vormultipliziert(b)
	w, h := pa.Bounds().Dx(), pa.Bounds().Dy()
	v := Vergleich{Pixel: w * h}
	einfarbig := func(x, y int, q *image.RGBA) bool {
		for k := 0; k < 4; k++ {
			lo, hi := 255, 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					xx, yy := x+dx, y+dy
					if xx < 0 || yy < 0 || xx >= w || yy >= h {
						continue
					}
					c := int(q.Pix[(yy*w+xx)*4+k])
					lo, hi = min(lo, c), max(hi, c)
				}
			}
			if hi-lo > Toleranz {
				return false
			}
		}
		return true
	}
	mittel := func(q *image.RGBA, x, y, k int) float64 {
		s, n := 0, 0
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				xx, yy := x+dx, y+dy
				if xx < 0 || yy < 0 || xx >= w || yy >= h {
					continue
				}
				s += int(q.Pix[(yy*w+xx)*4+k])
				n++
			}
		}
		return float64(s) / float64(n)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			md := 0.0
			for k := 0; k < 4; k++ {
				md = math.Max(md, math.Abs(mittel(pa, x, y, k)-mittel(pb, x, y, k)))
			}
			if int(math.Round(md)) > v.Mittel {
				v.Mittel = int(math.Round(md))
			}
			verstoss := md > MittelSchwelle
			if verstoss {
				v.MittelZu++
			}
			d := 0
			for k := 0; k < 4; k++ {
				e := int(pa.Pix[i+k]) - int(pb.Pix[i+k])
				d = max(d, e, -e)
			}
			if d > 0 {
				v.Anders++
			}
			v.Groesste = max(v.Groesste, d)
			if d > 32 {
				v.Ueber32++
			}
			if d > Toleranz {
				v.UeberTol++
				verstoss = verstoss || einfarbig(x, y, pa) || einfarbig(x, y, pb)
			}
			if verstoss {
				v.Verstoesse++
				if len(v.Beispiele) < 5 {
					v.Beispiele = append(v.Beispiele, image.Point{X: x, Y: y})
				}
			}
		}
	}
	return v, nil
}
