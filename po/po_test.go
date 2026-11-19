// SPDX-License-Identifier: GPL-3.0-or-later

package po_test

import (
	"reflect"
	"testing"

	"github.com/Andiministrator/schnapp/i18n"
	"github.com/Andiministrator/schnapp/po"
)

// Die eingebetteten Dateien sind lesbar und tragen die erwarteten Sprachen und Mehrzahlregeln.
func TestEingebettet(t *testing.T) {
	sprachen, err := i18n.Sprachen(po.Dateien)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"de"}; !reflect.DeepEqual(sprachen, want) {
		t.Fatalf("Sprachen = %v, want %v", sprachen, want)
	}
	for _, s := range sprachen {
		if _, err := i18n.Lade(po.Dateien, s); err != nil {
			t.Errorf("%s.po: %v", s, err)
		}
	}
}
