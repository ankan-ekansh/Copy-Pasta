package store

import "testing"

func TestGenerateID(t *testing.T) {
	id, err := GenerateID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(id) != idLength {
		t.Errorf("expected ID length %d, got %d", idLength, len(id))
	}

	// Ensure uniqueness (basic sanity check)
	seen := map[string]bool{id: true}
	for i := 0; i < 1000; i++ {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateID_CharacterSet(t *testing.T) {
	for i := 0; i < 100; i++ {
		id, err := GenerateID()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
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
