package lib

import (
	"strings"
	"testing"
)

func TestSimilarity_Identical(t *testing.T) {
	if got := Similarity("hello world", "hello world"); got != 1.0 {
		t.Errorf("identical strings: got %f, want 1.0", got)
	}
}

func TestSimilarity_CompletelyDifferent(t *testing.T) {
	got := Similarity("aaaaaa", "zzzzzz")
	if got > 0.01 {
		t.Errorf("completely different: got %f, want ~0.0", got)
	}
}

func TestSimilarity_Empty(t *testing.T) {
	if got := Similarity("", ""); got != 1.0 {
		t.Errorf("both empty: got %f, want 1.0", got)
	}
	if got := Similarity("hello", ""); got != 0.0 {
		t.Errorf("one empty: got %f, want 0.0", got)
	}
}

func TestSimilarity_HighOverlap(t *testing.T) {
	a := "FAIL: 3 errors found in parser.go"
	b := "FAIL: 3 errors found in parser.go "
	got := Similarity(a, b)
	if got < 0.90 {
		t.Errorf("high overlap: got %f, want >= 0.90", got)
	}
}

func TestSimilarity_ModerateOverlap(t *testing.T) {
	a := "FAIL: 3 errors found in parser.go"
	b := "FAIL: 5 errors found in handler.go"
	got := Similarity(a, b)
	if got < 0.4 || got > 0.9 {
		t.Errorf("moderate overlap: got %f, want between 0.4 and 0.9", got)
	}
}

func TestSimilarity_LongIdenticalOutputs(t *testing.T) {
	long := strings.Repeat("test output line\n", 100)
	if got := Similarity(long, long); got != 1.0 {
		t.Errorf("long identical: got %f, want 1.0", got)
	}
}

func TestSimilarity_Short(t *testing.T) {
	if got := Similarity("a", "a"); got != 1.0 {
		t.Errorf("single char identical: got %f, want 1.0", got)
	}
	if got := Similarity("a", "b"); got != 0.0 {
		t.Errorf("single char different: got %f, want 0.0", got)
	}
}
