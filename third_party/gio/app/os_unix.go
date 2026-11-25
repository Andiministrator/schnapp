// SPDX-License-Identifier: Unlicense OR MIT

//go:build (linux && !android) || freebsd || openbsd

package app

import (
	"errors"
	"unsafe"

	"gioui.org/io/pointer"
)

type X11ViewEvent struct {
	// Display is a pointer to the X11 Display created by XOpenDisplay.
	Display unsafe.Pointer
	// Window is the X11 window ID as returned by XCreateWindow.
	Window uintptr
}

func (X11ViewEvent) implementsViewEvent() {}
func (X11ViewEvent) ImplementsEvent()     {}
func (x X11ViewEvent) Valid() bool {
	return x != (X11ViewEvent{})
}

type WaylandViewEvent struct {
	// Display is the *wl_display returned by wl_display_connect.
	Display unsafe.Pointer
	// Surface is the *wl_surface returned by wl_compositor_create_surface.
	Surface unsafe.Pointer
	// Toplevel is the *xdg_toplevel returned by xdg_surface_get_toplevel.
	Toplevel unsafe.Pointer
}

func (WaylandViewEvent) implementsViewEvent() {}
func (WaylandViewEvent) ImplementsEvent()     {}
func (w WaylandViewEvent) Valid() bool {
	return w != (WaylandViewEvent{})
}

func osMain() {
	select {}
}

type windowDriver func(*callbacks, []Option) error

// Instead of creating files with build tags for each combination of wayland +/- x11
// let each driver initialize these variables with their own version of createWindow.
var wlDriver, x11Driver windowDriver

func newWindow(window *callbacks, options []Option) {
	var errFirst error
	for _, d := range []windowDriver{wlDriver, x11Driver} {
		if d == nil {
			continue
		}
		err := d(window, options)
		if err == nil {
			return
		}
		if errFirst == nil {
			errFirst = err
		}
	}
	if errFirst == nil {
		errFirst = errors.New("app: no window driver available")
	}
	window.ProcessEvent(DestroyEvent{Err: errFirst})
}

// xCursor contains mapping from pointer.Cursor to XCursor.
var xCursor = [...]string{
	pointer.CursorDefault:                  "left_ptr",
	pointer.CursorNone:                     "",
	pointer.CursorText:                     "xterm",
	pointer.CursorVerticalText:             "vertical-text",
	pointer.CursorPointer:                  "hand2",
	pointer.CursorCrosshair:                "crosshair",
	pointer.CursorAllScroll:                "fleur",
	pointer.CursorColResize:                "sb_h_double_arrow",
	pointer.CursorRowResize:                "sb_v_double_arrow",
	pointer.CursorGrab:                     "hand1",
	pointer.CursorGrabbing:                 "move",
	pointer.CursorNotAllowed:               "crossed_circle",
	pointer.CursorWait:                     "watch",
	pointer.CursorProgress:                 "left_ptr_watch",
	pointer.CursorNorthWestResize:          "top_left_corner",
	pointer.CursorNorthEastResize:          "top_right_corner",
	pointer.CursorSouthWestResize:          "bottom_left_corner",
	pointer.CursorSouthEastResize:          "bottom_right_corner",
	pointer.CursorNorthSouthResize:         "sb_v_double_arrow",
	pointer.CursorEastWestResize:           "sb_h_double_arrow",
	pointer.CursorWestResize:               "left_side",
	pointer.CursorEastResize:               "right_side",
	pointer.CursorNorthResize:              "top_side",
	pointer.CursorSouthResize:              "bottom_side",
	pointer.CursorNorthEastSouthWestResize: "fd_double_arrow",
	pointer.CursorNorthWestSouthEastResize: "bd_double_arrow",
}

// xCursorFallback lists further names per cursor, tried in order after the
// CSS name and before the X11 name of xCursor. Themes differ: Breeze, for
// example, has neither fd_double_arrow nor bd_double_arrow but nesw-resize and
// size_bdiag, so a single name falls back to the default arrow.
var xCursorFallback = map[pointer.Cursor][]string{
	pointer.CursorNorthWestSouthEastResize: {"nwse-resize", "size_fdiag"},
	pointer.CursorNorthEastSouthWestResize: {"nesw-resize", "size_bdiag"},
	pointer.CursorNorthSouthResize:         {"ns-resize", "size_ver"},
	pointer.CursorEastWestResize:           {"ew-resize", "size_hor"},
	pointer.CursorRowResize:                {"row-resize", "size_ver"},
	pointer.CursorColResize:                {"col-resize", "size_hor"},
	pointer.CursorAllScroll:                {"all-scroll", "size_all"},
	pointer.CursorGrab:                     {"grab", "openhand"},
	pointer.CursorGrabbing:                 {"grabbing", "closedhand"},
	pointer.CursorPointer:                  {"pointer"},
	pointer.CursorNotAllowed:               {"not-allowed", "forbidden"},
	pointer.CursorNorthWestResize:          {"nw-resize"},
	pointer.CursorNorthEastResize:          {"ne-resize"},
	pointer.CursorSouthWestResize:          {"sw-resize"},
	pointer.CursorSouthEastResize:          {"se-resize"},
	pointer.CursorNorthResize:              {"n-resize"},
	pointer.CursorSouthResize:              {"s-resize"},
	pointer.CursorWestResize:               {"w-resize"},
	pointer.CursorEastResize:               {"e-resize"},
}

// xCursorNames returns the theme names to try for c, most specific first.
func xCursorNames(c pointer.Cursor) []string {
	names := append([]string(nil), xCursorFallback[c]...)
	if x := xCursor[c]; x != "" {
		names = append(names, x)
	}
	return names
}
