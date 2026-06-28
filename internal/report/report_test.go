package report

import "testing"

func TestDisplayWidthAndTruncate(t *testing.T) {
	if got := DisplayWidth("a糖b"); got != 4 {
		t.Fatalf("DisplayWidth = %d", got)
	}
	got := Truncate("最少需要取出21个", 8)
	if DisplayWidth(got) > 8 {
		t.Fatalf("Truncate width = %d, value %q", DisplayWidth(got), got)
	}
	if got := Truncate("abcdef", 2); got != ".." {
		t.Fatalf("Truncate tiny width = %q", got)
	}
	if got := Truncate("abcdef", 0); got != "" {
		t.Fatalf("Truncate zero width = %q", got)
	}
}
