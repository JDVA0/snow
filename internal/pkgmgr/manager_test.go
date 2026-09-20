package pkgmgr

import "testing"

func TestSatisfiesVersion(t *testing.T) {
	cases := []struct {
		actual, constraint string
		want               bool
	}{
		{"1.4.2", "1.4.2", true},
		{"1.4.2", ">=1.2.0", true},
		{"1.4.2", "~1.4.0", true},
		{"1.5.0", "~1.4.0", false},
		{"1.8.0", "^1.2.0", true},
		{"2.0.0", "^1.2.0", false},
	}
	for _, test := range cases {
		if got := satisfiesVersion(test.actual, test.constraint); got != test.want {
			t.Errorf("satisfiesVersion(%q, %q) = %v, want %v", test.actual, test.constraint, got, test.want)
		}
	}
}
