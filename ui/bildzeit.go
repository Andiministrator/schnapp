// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import "time"

// zugZeiten sammelt die Bildzeiten eines Zeigerzugs (Drücken bis Loslassen) für die Diagnose (F-250). Bildzeit ist die
// Dauer vom Eintreffen des FrameEvent bis zum Abschicken des Bildes (nach app.FrameEvent.Frame), also die Arbeit der
// Ereignisschleife je Bild — nicht der Abstand zwischen zwei Bildern. Die Einheit kennt kein Gio und ist nur dann im
// Spiel, wenn SCHNAPP_DEBUG gesetzt ist; sie hält nur drei Zahlen und braucht keinen Speicher je Bild.
type zugZeiten struct {
	aktiv  bool          // ein Zug läuft: nur dann werden Bilder gezählt
	n      int           // Zahl der Bilder im Zug
	summe  time.Duration // Summe der Bildzeiten
	laengs time.Duration // längste Bildzeit
}

// ms rechnet eine Dauer in Millisekunden für die Diagnosezeile.
func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// beginne startet einen neuen Zug und verwirft den Stand des vorigen.
func (z *zugZeiten) beginne() { *z = zugZeiten{aktiv: true} }

// bild trägt die Zeit eines Bildes ein; außerhalb eines Zugs geschieht nichts.
func (z *zugZeiten) bild(d time.Duration) {
	if !z.aktiv {
		return
	}
	z.n++
	z.summe += d
	if d > z.laengs {
		z.laengs = d
	}
}

// mittel liefert die mittlere Bildzeit, ohne Bilder 0.
func (z *zugZeiten) mittel() time.Duration {
	if z.n == 0 {
		return 0
	}
	return z.summe / time.Duration(z.n)
}

// beende beendet den Zug und liefert Zahl, Mittel und Maximum; ohne laufenden Zug ok = false.
func (z *zugZeiten) beende() (n int, mittel, laengste time.Duration, ok bool) {
	if !z.aktiv {
		return 0, 0, 0, false
	}
	n, mittel, laengste = z.n, z.mittel(), z.laengs
	*z = zugZeiten{}
	return n, mittel, laengste, true
}
