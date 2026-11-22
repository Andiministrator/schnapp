// SPDX-License-Identifier: GPL-3.0-or-later

package errprefix

import (
	"errors"
	"io/fs"
	"testing"
)

func TestWrap(t *testing.T) {
	if Wrap("p", nil) != nil {
		t.Error("nil bleibt nil")
	}
	fremd := fs.ErrNotExist
	w := Wrap("p", fremd)
	if w.Error() != "p: "+fremd.Error() || !errors.Is(w, fremd) {
		t.Errorf("fremder Fehler: %q", w)
	}
	if again := Wrap("p", w); again != w {
		t.Errorf("Präfix doppelt gesetzt: %q", again)
	}
	// Ein Join bleibt ein Join, jeder Zweig bekommt sein Präfix.
	a, b := errors.New("p: erstens"), errors.New("zweitens")
	j := Wrap("p", errors.Join(a, b))
	if j.Error() != "p: erstens\np: zweitens" || !errors.Is(j, a) || !errors.Is(j, b) {
		t.Errorf("Join: %q", j)
	}
	if _, ok := j.(interface{ Unwrap() []error }); !ok {
		t.Error("Join verlor seine Zweige")
	}
}

func TestDefer(t *testing.T) {
	f := func() (fehler error) {
		defer Defer("p", &fehler)
		return fs.ErrPermission
	}
	if err := f(); err.Error() != "p: "+fs.ErrPermission.Error() || !errors.Is(err, fs.ErrPermission) {
		t.Errorf("Defer: %v", err)
	}
	g := func() (fehler error) {
		defer Defer("p", &fehler)
		return nil
	}
	if g() != nil {
		t.Error("Defer macht aus nil einen Fehler")
	}
}
