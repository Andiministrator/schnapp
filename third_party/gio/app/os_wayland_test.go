// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd) && !nowayland

package app

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	syscall "golang.org/x/sys/unix"

	"gioui.org/io/pointer"
)

// Schnapp patch (F-176): writeAllFD writes several MB through a non-blocking
// pipe whose reader is slow, ends with EPIPE when the reader is gone and gives
// up when the reader never reads.
func TestWriteAllFD(t *testing.T) {
	pipe := func(t *testing.T) (r *os.File, w int) {
		var p [2]int
		if err := syscall.Pipe2(p[:], syscall.O_NONBLOCK|syscall.O_CLOEXEC); err != nil {
			t.Fatal(err)
		}
		return os.NewFile(uintptr(p[0]), "r"), p[1]
	}
	data := make([]byte, 5<<20)
	for i := range data {
		data[i] = byte(i * 7)
	}

	t.Run("blocking pipe without reader gives up", func(t *testing.T) {
		var p [2]int
		if err := syscall.Pipe2(p[:], syscall.O_CLOEXEC); err != nil { // blocking, as most receivers create it
			t.Fatal(err)
		}
		defer syscall.Close(p[0])
		defer syscall.Close(p[1])
		done := make(chan error, 1)
		go func() { done <- writeAllFD(p[1], data, 200*time.Millisecond) }()
		select {
		case err := <-done:
			if !errors.Is(err, errWriteStalled) {
				t.Errorf("err = %v, want errWriteStalled", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("writeAllFD still blocked after 5 s")
		}
	})

	t.Run("complete", func(t *testing.T) {
		r, w := pipe(t)
		defer r.Close()
		got := make(chan []byte)
		go func() {
			var buf bytes.Buffer
			chunk := make([]byte, 4096)
			for {
				n, err := r.Read(chunk)
				buf.Write(chunk[:n])
				if err != nil {
					got <- buf.Bytes()
					return
				}
				if buf.Len()%(256<<10) < 4096 {
					time.Sleep(time.Millisecond) // slow reader: the writer meets EAGAIN
				}
			}
		}()
		err := writeAllFD(w, data, 5*time.Second)
		syscall.Close(w)
		if err != nil {
			t.Fatal(err)
		}
		if b := <-got; !bytes.Equal(b, data) {
			t.Errorf("read %d bytes, want %d identical bytes", len(b), len(data))
		}
	})

	t.Run("reader closed", func(t *testing.T) {
		r, w := pipe(t)
		defer syscall.Close(w)
		r.Close()
		if err := writeAllFD(w, data, time.Second); !errors.Is(err, syscall.EPIPE) {
			t.Errorf("err %v, want EPIPE", err)
		}
	})

	t.Run("reader stalls", func(t *testing.T) {
		r, w := pipe(t)
		defer r.Close()
		defer syscall.Close(w)
		start := time.Now()
		if err := writeAllFD(w, data, 50*time.Millisecond); !errors.Is(err, errWriteStalled) {
			t.Errorf("err %v, want errWriteStalled", err)
		}
		if d := time.Since(start); d > 2*time.Second {
			t.Errorf("gave up after %v", d)
		}
	})
}

func TestSortedOutputs(t *testing.T) {
	in := []WaylandOutput{
		{Name: "HDMI-A-1", Width: 1920, Height: 1080, Scale: 1},
		{Name: "", Width: 800, Height: 600, Scale: 1},
		{Name: "DP-2", Width: 3840, Height: 2160, Scale: 2},
		{Name: "DP-1", Width: 2560, Height: 1440, Scale: 1},
	}
	want := []WaylandOutput{
		{Name: "DP-1", Width: 2560, Height: 1440, Scale: 1},
		{Name: "DP-2", Width: 3840, Height: 2160, Scale: 2},
		{Name: "HDMI-A-1", Width: 1920, Height: 1080, Scale: 1},
	}
	if got := sortedOutputs(in); !reflect.DeepEqual(got, want) {
		t.Errorf("sortedOutputs = %v, want %v", got, want)
	}
	if got := sortedOutputs([]WaylandOutput{{}}); len(got) != 0 {
		t.Errorf("unnamed output kept: %v", got)
	}
	// Runs of digits compare by value (schnapp F-42).
	got := sortedOutputs([]WaylandOutput{{Name: "DP-10"}, {Name: "DP-2"}, {Name: "DP-1"}, {Name: "HDMI-A-1"}})
	var names []string
	for _, o := range got {
		names = append(names, o.Name)
	}
	if want := []string{"DP-1", "DP-2", "DP-10", "HDMI-A-1"}; !reflect.DeepEqual(names, want) {
		t.Errorf("sortedOutputs order = %v, want %v", names, want)
	}
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"DP-2", "DP-10", true},
		{"DP-10", "DP-2", false},
		{"DP-1", "DP-1", false},
		{"DP-01", "DP-1", true}, // same value, plain order decides
		{"DP-1", "DP-01", false},
		{"DP", "DP-1", true},
		{"eDP-1", "DP-1", false},
		{"HDMI-A-2", "HDMI-A-10", true},
		{"a9b", "a10a", true},
		{"", "a", true},
	}
	for _, tc := range tests {
		if got := naturalLess(tc.a, tc.b); got != tc.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		// Strict weak order: never both a<b and b<a.
		if naturalLess(tc.a, tc.b) && naturalLess(tc.b, tc.a) {
			t.Errorf("naturalLess(%q, %q) not antisymmetric", tc.a, tc.b)
		}
	}
}

// Every resize, move and grab cursor has more than one theme name, so a theme
// without the X11 name (Breeze lacks fd_double_arrow and bd_double_arrow)
// still finds one before falling back to the default arrow.
func TestXCursorNamesHaveFallbacks(t *testing.T) {
	for _, c := range []pointer.Cursor{
		pointer.CursorNorthWestSouthEastResize, pointer.CursorNorthEastSouthWestResize,
		pointer.CursorNorthSouthResize, pointer.CursorEastWestResize, pointer.CursorAllScroll,
		pointer.CursorGrab, pointer.CursorGrabbing,
	} {
		names := xCursorNames(c)
		if len(names) < 3 {
			t.Errorf("%v: names %v, want at least three", c, names)
		}
		if names[len(names)-1] != xCursor[c] {
			t.Errorf("%v: X11 name %q not last in %v", c, xCursor[c], names)
		}
	}
	if got := xCursorNames(pointer.CursorNorthWestSouthEastResize); got[0] != "nwse-resize" || got[1] != "size_fdiag" {
		t.Errorf("NWSE: %v", got)
	}
	if got := xCursorNames(pointer.CursorNone); len(got) != 0 {
		t.Errorf("None: %v", got)
	}
}
