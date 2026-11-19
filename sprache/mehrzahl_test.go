// SPDX-License-Identifier: GPL-3.0-or-later

package sprache

import (
	"strings"
	"testing"
)

func TestMehrzahlregel(t *testing.T) {
	faelle := []struct {
		kopf string
		want map[int]int // n → Index
	}{
		// Deutsch, Englisch
		{"nplurals=2; plural=(n != 1);", map[int]int{0: 1, 1: 0, 2: 1, 21: 1}},
		// Französisch: 0 und 1 Einzahl
		{"nplurals=2; plural=(n > 1);", map[int]int{0: 0, 1: 0, 2: 1}},
		// Japanisch: eine Form
		{"nplurals=1; plural=0;", map[int]int{0: 0, 1: 0, 5: 0}},
		// Polnisch: Vorrang von && vor || und verschachteltes ?:
		{"nplurals=3; plural=(n==1 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
			map[int]int{1: 0, 2: 1, 4: 1, 5: 2, 12: 2, 22: 1, 25: 2, 112: 2, 0: 2}},
		// Russisch
		{"nplurals=3; plural=(n%10==1 && n%100!=11 ? 0 : n%10>=2 && n%10<=4 && (n%100<10 || n%100>=20) ? 1 : 2);",
			map[int]int{1: 0, 11: 2, 21: 0, 3: 1, 13: 2, 5: 2}},
		// Arabisch: sechs Formen
		{"nplurals=6; plural=(n==0 ? 0 : n==1 ? 1 : n==2 ? 2 : n%100>=3 && n%100<=10 ? 3 : n%100>=11 ? 4 : 5);",
			map[int]int{0: 0, 1: 1, 2: 2, 3: 3, 10: 3, 11: 4, 99: 4, 100: 5, 102: 5}},
		// Negation, Rechenoperatoren
		{"nplurals=2; plural=!(n*2-2 == 0);", map[int]int{1: 0, 2: 1}},
		// Index außerhalb der Formen fällt auf 0
		{"nplurals=2; plural=n;", map[int]int{0: 0, 1: 1, 7: 0}},
		// Division durch 0 stürzt nicht ab
		{"nplurals=2; plural=n/0 + n%0;", map[int]int{3: 0}},
	}
	for _, f := range faelle {
		r, err := LeseMehrzahlregel(f.kopf)
		if err != nil {
			t.Errorf("%q: %v", f.kopf, err)
			continue
		}
		for n, want := range f.want {
			if got := r.Index(n); got != want {
				t.Errorf("%q: Index(%d) = %d, want %d", f.kopf, n, got, want)
			}
		}
	}
}

func TestMehrzahlregelFehler(t *testing.T) {
	for _, kopf := range []string{
		"",
		"plural=(n != 1);",
		"nplurals=2;",
		"nplurals=0; plural=0;",
		"nplurals=x; plural=0;",
		"nplurals=2; plural=(n != 1;",
		"nplurals=2; plural=n ? 1;",
		"nplurals=2; plural=n x 1;",
		"nplurals=2; plural=k;",
		"nplurals=2; plural=99999999999999999999;",
	} {
		if _, err := LeseMehrzahlregel(kopf); err == nil {
			t.Errorf("%q: kein Fehler", kopf)
		}
	}
}

// Sehr tiefe Klammerung bricht mit Fehler ab statt mit Stapelüberlauf (F-36); der Nullwert stürzt nicht ab.
func TestMehrzahlregelGrenzen(t *testing.T) {
	tief := "nplurals=2; plural=" + strings.Repeat("(", 1_000_000) + "n" + strings.Repeat(")", 1_000_000) + ";"
	if _, err := LeseMehrzahlregel(tief); err == nil {
		t.Error("tiefe Klammerung ohne Fehler")
	}
	grenze := "nplurals=2; plural=" + strings.Repeat("(", 500) + "n" + strings.Repeat(")", 500) + ";"
	if _, err := LeseMehrzahlregel(grenze); err != nil {
		t.Errorf("1001 Zeichen: %v", err)
	}
	var r Mehrzahlregel
	if r.Index(1) != 0 || r.Index(2) != 1 {
		t.Errorf("Nullwert: Index(1)=%d Index(2)=%d", r.Index(1), r.Index(2))
	}
}
