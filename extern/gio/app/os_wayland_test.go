// SPDX-License-Identifier: Unlicense OR MIT

//go:build ((linux && !android) || freebsd) && !nowayland

package app

import (
	"reflect"
	"testing"
)

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
}
