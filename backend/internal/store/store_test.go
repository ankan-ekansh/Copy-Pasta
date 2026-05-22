package store

import "testing"

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if len(id) != idLength {
		t.Errorf("expected ID length %d, got %d", idLength, len(id))
	}

	// Ensure uniqueness (basic sanity check)
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateID()
		if seen[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateID_CharacterSet(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := GenerateID()
		for _, c := range id {
			found := false
			for _, a := range idAlphabet {
				if c == a {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ID contains invalid character: %c", c)
			}
		}
	}
}
