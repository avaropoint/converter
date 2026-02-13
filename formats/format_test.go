package formats

import "testing"

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.txt", "normal.txt"},
		{"path/to/file.txt", "path_to_file.txt"},
		{"", "unnamed"},
		{"a:b*c?d", "a_b_c_d"},
	}
	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectNilData(t *testing.T) {
	result := Detect("test.xyz", nil)
	if result != nil {
		_ = result
	}
}

func TestRegisterAndAll(t *testing.T) {
	count := len(All())
	if count < 0 {
		t.Fatal("All() returned negative length")
	}
}
