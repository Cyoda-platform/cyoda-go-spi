package spi

import "testing"

func TestValidateChangeLevel(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want ChangeLevel
		err  bool
	}{
		{"ARRAY_LENGTH", ChangeLevelArrayLength, false},
		{"ARRAY_ELEMENTS", ChangeLevelArrayElements, false},
		{"TYPE", ChangeLevelType, false},
		{"STRUCTURAL", ChangeLevelStructural, false},
		{"", "", true},
		{"invalid", "", true},
	} {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ValidateChangeLevel(tc.in)
			if (err != nil) != tc.err {
				t.Fatalf("got err=%v, want err=%v", err, tc.err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
