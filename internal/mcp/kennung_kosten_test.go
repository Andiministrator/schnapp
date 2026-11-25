// SPDX-License-Identifier: GPL-3.0-or-later

package mcp

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// F-772: Exponenten werden nicht ausmultipliziert; Ergebnis klein und schnell, auch unter -race.
func TestNormIDGrosserExponent(t *testing.T) {
	for _, id := range []string{`1e1000000`, `1e-1000000`, `1e999999`, `1.5e-999999`, `1e100000`, `1e99999999999999999999`} {
		t0 := time.Now()
		n := normID(json.RawMessage(id))
		if d := time.Since(t0); d > 50*time.Millisecond {
			t.Errorf("normID(%s) braucht %v", id, d)
		}
		if len(n) > 64 {
			t.Errorf("normID(%s) liefert %d Bytes", id, len(n))
		}
	}
}

// F-772: gleiche Zahl in anderer Schreibweise gleich, verschiedene verschieden, Zeichenkette keine Zahl.
func TestNormIDKanonisch(t *testing.T) {
	gleich := [][2]string{{`1`, `1.0`}, {`1`, `10e-1`}, {`100`, `1e2`}, {`0`, `-0`}, {`0`, `0.000`}, {`0.5`, `5e-1`}, {`007`, `7`}, {`1E+2`, `100`}, {`1e1000000`, `10e999999`}}
	for _, p := range gleich {
		if string(normID(json.RawMessage(p[0]))) != string(normID(json.RawMessage(p[1]))) {
			t.Errorf("%s und %s sollen gleich sein: %s / %s", p[0], p[1], normID(json.RawMessage(p[0])), normID(json.RawMessage(p[1])))
		}
	}
	verschieden := [][2]string{{`1`, `"1"`}, {`1`, `-1`}, {`0.5`, `5`}, {`1e1000000`, `1e999999`}, {`9007199254740992`, `9007199254740993`}, {`12`, `1.2`}, {`1e2`, `1e-2`}}
	for _, p := range verschieden {
		if string(normID(json.RawMessage(p[0]))) == string(normID(json.RawMessage(p[1]))) {
			t.Errorf("%s und %s sollen verschieden sein", p[0], p[1])
		}
	}
}

// F-772: 64 wartende Anfragen mit riesigen Exponenten, ein Abbruch: schnell.
func TestStreicheMitGrossenExponenten(t *testing.T) {
	var q schlange
	for i := range 64 {
		z := zeile{daten: []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1e%d,"method":"tools/call","params":{"name":"b"}}`, 999000+i))}
		if !q.stelle(z) {
			t.Fatalf("Zeile %d passt nicht in die Schlange", i)
		}
	}
	t0 := time.Now()
	ok := q.streiche(json.RawMessage(`1e1000000`))
	if d := time.Since(t0); d > 500*time.Millisecond {
		t.Errorf("ein Abbruch hält die Leseschleife %v auf", d)
	}
	if ok || q.laenge() != 64 {
		t.Errorf("falsche Anfrage gestrichen (ok=%v, Länge %d)", ok, q.laenge())
	}
	if !q.streiche(json.RawMessage(`1e999010`)) || q.laenge() != 63 {
		t.Errorf("genaue Kennung nicht gestrichen (Länge %d)", q.laenge())
	}
}
