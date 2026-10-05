package helper

import (
	"net/http/httptest"
	"testing"
)

func TestDecodeRequestQueryPagination(t *testing.T) {
	req := httptest.NewRequest("GET", "/items?limit=6&cursor=eyJpZCI6MTZ9", nil)

	got, err := DecodeRequest(req)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if got.Pagination.Limit != 6 {
		t.Fatalf("limit = %d, want 6", got.Pagination.Limit)
	}
	if got.Pagination.Cursor != "eyJpZCI6MTZ9" {
		t.Fatalf("cursor = %q, want opaque cursor", got.Pagination.Cursor)
	}
}

func TestDecodeRequestInvalidLimit(t *testing.T) {
	req := httptest.NewRequest("GET", "/items?limit=invalid", nil)

	if _, err := DecodeRequest(req); err == nil {
		t.Fatal("DecodeRequest() error = nil, want invalid limit error")
	}
}
