// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"slices"

	"gioui.org/layout"
	"gioui.org/op"

	"github.com/Andiministrator/schnapp/internal/document"
	"github.com/Andiministrator/schnapp/internal/render"
)

// Freihand-Striche auf der Leinwand (F-1038, ADR-0049 Punkt 2): Die Zerlegung (render.ZerlegeStrich) und der daraus
// aufgezeichnete Pfad werden je Objekt gemerkt, solange das Objekt dem Wert nach gleich bleibt (document.Gleich: Punkte,
// Strich, Ebene); die Bereiche zwischen den Nähten (document.StreckenStuecke) zusätzlich, solange die Ansicht gleich
// bleibt. Ohne Merker zerlegte der Editor jeden Strich in jedem Bild neu (20 Striche zu 4096 Punkten: 56 ms je Bild).
// Vorlage ist der Merker der Schatten (schatten.go): Speichergrenze, Verdrängen der am längsten nicht gezeichneten,
// nie eines aus dem laufenden Bild, Leeren beim Wechsel der Aufnahme (uebernimm).

// freihandSpeicher begrenzt die geschätzten Bytes aller gemerkten Freihand-Striche. Variable nur für Tests.
var freihandSpeicher = 64 << 20

// freihandBytesJePunkt schätzt Zerlegung und aufgezeichneten Pfad je Punkt der Teile (16 Byte je Punkt, rund 30 Byte
// Pfadbefehl in Gio).
const freihandBytesJePunkt = 48

// freihandBild ist ein gemerkter Freihand-Strich. Ein geänderter Strich bekommt ein neues freihandBild (nie ops.Reset):
// Ein Aufruf des alten kann im laufenden Bild schon aufgezeichnet sein.
type freihandBild struct {
	o      document.Freihand
	sz     render.StrichZerlegung
	ops    op.Ops
	aufruf op.CallOp // die Teile im Original (Abbildung ohne Umrechnung), aufgezeichnet in ops
	// stuecke gelten für ansicht (Dokument, Zuschnitt, Streifen, Kante).
	stuecke      []document.Stueck
	ansicht      schattenAnsicht
	stueckeDa    bool
	bytes, runde int
}

// freihandFuer liefert den gemerkten Strich von v und zerlegt ihn neu, wenn er sich geändert hat.
func (e *editor) freihandFuer(gtx layout.Context, v document.Freihand) *freihandBild {
	if e.freihand == nil {
		e.freihand = map[int]*freihandBild{}
	}
	if b, ok := e.freihand[v.ID]; ok && document.Gleich(b.o, v) {
		b.runde = e.freihandRunde
		return b
	} else if ok {
		e.freihandBytes -= b.bytes
		delete(e.freihand, v.ID)
	}
	e.freihandNeu++
	b := &freihandBild{o: v, sz: render.ZerlegeStrich(v.Punkte(), v.Strich), runde: e.freihandRunde}
	m := op.Record(&b.ops)
	g := gtx
	g.Ops = &b.ops
	fuelleTeile(g, b.sz.Teile, v.Strich.Farbe, punktF32)
	b.aufruf = m.Stop()
	for _, t := range b.sz.Teile {
		b.bytes += len(t) * freihandBytesJePunkt
	}
	e.freihandVerdraengen(b.bytes)
	e.freihand[v.ID] = b
	e.freihandBytes += b.bytes
	return b
}

// stueckeFuer liefert die Bereiche, in denen der Strich zu sehen ist, gemerkt für die laufende Ansicht.
func (e *editor) stueckeFuer(b *freihandBild) []document.Stueck {
	d := e.s.D
	ansicht := schattenAnsicht{d: d, zuschnitt: d.Zuschnitt(), streifen: d.Streifen(), kante: d.Kante()}
	if b.stueckeDa && b.ansicht.gleich(ansicht) {
		return b.stuecke
	}
	b.stuecke, b.ansicht, b.stueckeDa = nil, ansicht, true
	if len(b.sz.Zug) > 0 {
		for _, st := range d.StreckenStuecke(b.sz.Zug, b.sz.Zu, b.sz.Suche) {
			b.stuecke = append(b.stuecke, st.Stueck)
		}
	}
	return b.stuecke
}

// freihandVerdraengen macht Platz für neu Bytes: die am längsten nicht gezeichneten Striche fallen weg, keiner aus dem
// laufenden Bild.
func (e *editor) freihandVerdraengen(neu int) {
	if e.freihandBytes+neu <= freihandSpeicher {
		return
	}
	alt := make([]int, 0, len(e.freihand))
	for id, b := range e.freihand {
		if b.runde < e.freihandRunde {
			alt = append(alt, id)
		}
	}
	slices.SortFunc(alt, func(a, b int) int { return e.freihand[a].runde - e.freihand[b].runde })
	for _, id := range alt {
		if e.freihandBytes+neu <= freihandSpeicher {
			break
		}
		e.freihandBytes -= e.freihand[id].bytes
		delete(e.freihand, id)
	}
}

// freihandAufraeumen verwirft nach jedem Bild die Striche von Objekten, die es nicht mehr gibt, und beginnt die nächste
// Runde.
func (e *editor) freihandAufraeumen() {
	e.freihandRunde++
	for id, b := range e.freihand {
		if _, ok := e.s.D.Objekt(id); !ok {
			e.freihandBytes -= b.bytes
			delete(e.freihand, id)
		}
	}
}
