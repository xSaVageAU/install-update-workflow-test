//go:build windows

package uninstall

import "testing"

func TestStripPathEntry(t *testing.T) {
	cases := []struct {
		name        string
		pathVar     string
		dir         string
		wantUpdated string
		wantChanged bool
	}{
		{
			name:        "removes matching entry",
			pathVar:     `C:\a;C:\Users\me\Programs\iuw;C:\b`,
			dir:         `C:\Users\me\Programs\iuw`,
			wantUpdated: `C:\a;C:\b`,
			wantChanged: true,
		},
		{
			name:        "case insensitive",
			pathVar:     `C:\a;c:\users\me\programs\iuw;C:\b`,
			dir:         `C:\Users\me\Programs\iuw`,
			wantUpdated: `C:\a;C:\b`,
			wantChanged: true,
		},
		{
			name:        "ignores trailing backslash",
			pathVar:     `C:\a;C:\Users\me\Programs\iuw\;C:\b`,
			dir:         `C:\Users\me\Programs\iuw`,
			wantUpdated: `C:\a;C:\b`,
			wantChanged: true,
		},
		{
			name:        "no match leaves PATH untouched",
			pathVar:     `C:\a;C:\b`,
			dir:         `C:\Users\me\Programs\iuw`,
			wantUpdated: `C:\a;C:\b`,
			wantChanged: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotUpdated, gotChanged := stripPathEntry(c.pathVar, c.dir)
			if gotUpdated != c.wantUpdated || gotChanged != c.wantChanged {
				t.Errorf("stripPathEntry(%q, %q) = (%q, %v), want (%q, %v)",
					c.pathVar, c.dir, gotUpdated, gotChanged, c.wantUpdated, c.wantChanged)
			}
		})
	}
}
