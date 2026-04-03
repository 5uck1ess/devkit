package lib

// Similarity computes the ratio of matching characters between two strings
// using a simple bigram overlap approach. Returns a value between 0.0 and 1.0.
// Used to detect "Groundhog Day" patterns where consecutive metric outputs
// are nearly identical, indicating the agent is stuck in a loop.
func Similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	// Short strings can't produce bigrams; fall back to exact match (handled above)
	if len(a) < 2 || len(b) < 2 {
		return 0.0
	}

	bigramsA := bigrams(a)
	bigramsB := bigrams(b)

	var matches int
	for bg, countA := range bigramsA {
		if countB, ok := bigramsB[bg]; ok {
			if countA < countB {
				matches += countA
			} else {
				matches += countB
			}
		}
	}

	total := len(a) - 1 + len(b) - 1
	if total == 0 {
		return 0.0
	}
	return 2.0 * float64(matches) / float64(total)
}

func bigrams(s string) map[string]int {
	m := make(map[string]int, len(s)-1)
	for i := 0; i < len(s)-1; i++ {
		m[s[i:i+2]]++
	}
	return m
}
