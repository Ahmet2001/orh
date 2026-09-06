package cli

import (
	"strings"
	"testing"
)

func TestReadRunInputPreservesPipedLines(t *testing.T) {
	got, err := readRunInput(strings.NewReader("QUESTION: q\nEVIDENCE: e\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "QUESTION: q\nEVIDENCE: e" {
		t.Errorf("input = %q", got)
	}
}
