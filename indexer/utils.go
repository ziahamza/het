package indexer

import (
	"math"
	"search/stemmer"
	"strings"
)

func GetVector(body string) (map[string]int, float64) {
	vector := make(map[string]int)
	for _, token := range strings.Fields(body) {
		word := stemmer.StemWord(token)
		if len(word) > 0 {
			vector[word]++
		}
	}

	length := 0
	for _, val := range vector {
		length += val * val
	}

	return vector, math.Sqrt(float64(length))

}
