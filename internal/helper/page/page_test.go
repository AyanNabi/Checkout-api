package page

import "testing"

func TestCursorEncoding(t *testing.T) {
	encoded, err := EncodeCursor(16)
	if err != nil {
		t.Fatalf("EncodeCursor() error = %v", err)
	}
	if encoded != "eyJpZCI6MTZ9" {
		t.Fatalf("encoded cursor = %q, want eyJpZCI6MTZ9", encoded)
	}

	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if decoded.ID != 16 {
		t.Fatalf("decoded ID = %d, want 16", decoded.ID)
	}
}

func TestCreatePage(t *testing.T) {
	data := []int{11, 12, 13, 14, 15, 16, 17}

	first := CreatePage(data, 6, func(item int) int { return item })
	if len(first.Data) != 6 {
		t.Fatalf("first page length = %d, want 6", len(first.Data))
	}
	if !first.HasMore {
		t.Fatal("first page has_more = false, want true")
	}
	if first.NextCursor == nil || *first.NextCursor != "eyJpZCI6MTZ9" {
		t.Fatalf("first page next_cursor = %v, want encoded ID 16", first.NextCursor)
	}

	second := CreatePage([]int{17, 18}, 6, func(item int) int { return item })
	if second.HasMore {
		t.Fatal("second page has_more = true, want false")
	}
	if second.NextCursor != nil {
		t.Fatalf("second page next_cursor = %v, want nil", second.NextCursor)
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	if _, err := DecodeCursor("not-a-cursor"); err == nil {
		t.Fatal("DecodeCursor() error = nil, want invalid cursor error")
	}
}
