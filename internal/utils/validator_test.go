package utils

import "testing"

func TestNormalizePhoneCanonicalizesGhanaFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "0541234567", want: "+233541234567"},
		{input: "233541234567", want: "+233541234567"},
		{input: "+233 54-123-4567", want: "+233541234567"},
		{input: "(054) 123 4567", want: "+233541234567"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()

			if got := NormalizePhone(tc.input); got != tc.want {
				t.Fatalf("NormalizePhone(%q) = %q, want %q", tc.input, got, tc.want)
			}
			if !ValidatePhone(tc.input) {
				t.Fatalf("ValidatePhone(%q) = false, want true", tc.input)
			}
		})
	}
}
