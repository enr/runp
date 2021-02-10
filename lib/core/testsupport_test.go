package core

import (
	"testing"
)

func assertSliceEquals(actual []string, expected []string, desc string, t *testing.T) {
	if len(actual) != len(expected) {
		t.Errorf(`%s, expected size %d, got %d`, desc, len(expected), len(actual))
	}
	xMap := make(map[string]int)
	yMap := make(map[string]int)

	for _, xElem := range actual {
		xMap[xElem]++
	}
	for _, yElem := range expected {
		yMap[yElem]++
	}

	for xMapKey, xMapVal := range xMap {
		if yMap[xMapKey] != xMapVal {
			t.Errorf(`%s, unexpected mount point %s`, desc, xMapKey)
		}
	}
	for yMapKey, yMapVal := range yMap {
		if xMap[yMapKey] != yMapVal {
			t.Errorf(`%s, mount point %s not found`, desc, yMapKey)
		}
	}
}
