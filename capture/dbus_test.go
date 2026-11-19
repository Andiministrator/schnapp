// SPDX-License-Identifier: GPL-3.0-or-later

package capture

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// memfdQuelle liest aus einer gewöhnlichen Datei genauso wie aus einem memfd (ReadAt/Stat).
func TestMemfdQuelle(t *testing.T) {
	pfad := filepath.Join(t.TempDir(), "roh")
	if err := os.WriteFile(pfad, []byte("0123456789"), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(pfad)
	if err != nil {
		t.Fatal(err)
	}
	m := &memfdQuelle{f: f}
	defer m.Close()
	if d, err := m.lies(4); err != nil || string(d) != "0123" {
		t.Errorf("lies(4) = %q, %v", d, err)
	}
	// Die Größenprüfung meldet sich selbst, bevor gelesen wird (F-56: sonst prüfte der Test nur ReadFull).
	if _, err := m.lies(11); err == nil || !strings.Contains(err.Error(), "memfd hat 10 Bytes") {
		t.Errorf("größer als die Datei: %v", err)
	}
}

func TestPipeQuelle(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	p := &pipeQuelle{r: r, fertig: make(chan gelesen, 1), ctx: context.Background()}
	go func() {
		d, err := liesBegrenzt(r, 100)
		p.fertig <- gelesen{d, err}
	}()
	w.Write([]byte("abcdef"))
	w.Close()
	d, err := p.lies(4)
	if err != nil || string(d) != "abcd" {
		t.Errorf("lies(4) = %q, %v", d, err)
	}
	p.Close()

	// Abbruch durch den Context, wenn die Gegenstelle nie schließt.
	r2, w2, _ := os.Pipe()
	defer w2.Close()
	ctx, abbruch := context.WithCancel(context.Background())
	abbruch()
	q := &pipeQuelle{r: r2, fertig: make(chan gelesen, 1), ctx: ctx}
	if _, err := q.lies(1); err == nil {
		t.Error("abgebrochener Context ohne Fehler")
	}
	q.Close()
}
