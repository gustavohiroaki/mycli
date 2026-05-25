package compress

import "testing"

func TestCRFFromLevel(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{1, 18},
		{35, 23},
		{100, 35},
	}
	for _, test := range tests {
		if got := crfFromLevel(test.level); got != test.want {
			t.Fatalf("crfFromLevel(%d) = %d, want %d", test.level, got, test.want)
		}
	}
}

func TestValidateOptionsRejectsInvalidLevel(t *testing.T) {
	if err := ValidateOptions(Options{Level: 0}); err == nil {
		t.Fatal("expected error")
	}
	if err := ValidateOptions(Options{Level: 101}); err == nil {
		t.Fatal("expected error")
	}
}
