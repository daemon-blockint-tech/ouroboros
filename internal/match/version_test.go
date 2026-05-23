package match

import "testing"

func TestVersionMatches(t *testing.T) {
	tests := []struct {
		wants []string
		have  string
		ok    bool
	}{
		{[]string{"1.2.3"}, "1.2.3", true},
		{[]string{"1.2.3"}, "1.2.4", false},
		{[]string{">=1.2.0"}, "1.2.3", true},
		{[]string{">=1.2.0"}, "1.1.9", false},
		{[]string{"<2.0.0"}, "1.9.9", true},
		{[]string{"<2.0.0"}, "2.0.0", false},
	}
	for _, tc := range tests {
		_, ok := VersionMatches(tc.wants, tc.have)
		if ok != tc.ok {
			t.Fatalf("VersionMatches(%v, %q) ok=%v want %v", tc.wants, tc.have, ok, tc.ok)
		}
	}
}
