package server

import "testing"

func TestNewRegistersTools(t *testing.T) {
	t.Parallel()
	if New() == nil {
		t.Fatal("expected server")
	}
}
