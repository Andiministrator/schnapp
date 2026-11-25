// SPDX-License-Identifier: GPL-3.0-or-later

package mcp

import (
	"encoding/json"
	"testing"
)

// F-758: Kennungen über 2^53 bleiben verschieden; ein Abbruch für 9007199254740992 streicht nicht die wartende
// Anfrage 9007199254740993.
func TestStreicheTrifftNichtAndereGrosseKennung(t *testing.T) {
	var q schlange
	q.stelle(zeile{daten: []byte(`{"jsonrpc":"2.0","id":9007199254740993,"method":"tools/call","params":{"name":"b"}}`)})
	if q.streiche(json.RawMessage(`9007199254740992`)) {
		t.Errorf("Abbruch für 9007199254740992 hat die wartende Anfrage 9007199254740993 gestrichen (Länge jetzt %d)", q.laenge())
	}
	if !q.streiche(json.RawMessage(`9007199254740993`)) || q.laenge() != 0 {
		t.Errorf("die genaue Kennung wurde nicht gestrichen (Länge %d)", q.laenge())
	}
}

// F-758: Ein wartendes initialize streicht ein Abbruch nie.
func TestStreicheLaesstInitializeStehen(t *testing.T) {
	var q schlange
	q.stelle(zeile{daten: []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)})
	if q.streiche(json.RawMessage(`1`)) || q.laenge() != 1 {
		t.Errorf("initialize wurde gestrichen (Länge %d)", q.laenge())
	}
}

// F-758: normID gleicht gleiche Zahlen in anderer Schreibweise an, trennt aber Zeichenketten von Zahlen und
// riesige Exponenten brechen nichts.
func TestNormID(t *testing.T) {
	gleich := [][2]string{{`1`, `1.0`}, {`2`, `2e0`}, {`100`, `1e2`}, {` 7 `, `7`}, {`"a"`, `"a"`}}
	for _, p := range gleich {
		if string(normID(json.RawMessage(p[0]))) != string(normID(json.RawMessage(p[1]))) {
			t.Errorf("%s und %s sollen gleich sein", p[0], p[1])
		}
	}
	verschieden := [][2]string{{`1`, `"1"`}, {`9007199254740992`, `9007199254740993`}, {`1`, `2`}, {`1e999999999`, `1`}, {`1e999999999`, `1e999999998`}}
	for _, p := range verschieden {
		if string(normID(json.RawMessage(p[0]))) == string(normID(json.RawMessage(p[1]))) {
			t.Errorf("%s und %s sollen verschieden sein", p[0], p[1])
		}
	}
}
