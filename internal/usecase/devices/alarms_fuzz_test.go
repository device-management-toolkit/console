package devices_test

import (
	"strings"
	"testing"

	devices "github.com/device-management-toolkit/console/internal/usecase/devices"
)

func FuzzParseInterval(f *testing.F) {
	seedInputs := []string{
		"",
		"P",
		"PT",
		"P2D",
		"PT5H",
		"PT30M",
		"P1DT6H30M",
		"P1DT6H30M45S",
		"P0DT0H0M0S",
		"PT-1H",
		"P-1D",
		"PX",
		"P1X",
		"P1DT",
		"PT1Q",
		"P999999999D",
		"P999999999999999999999999999999D",
		"PT999999999999999999999999999999H",
		"P1,D",
		"P1D/PT2H",
		"P1D/../T2H",
		"P1D\tT2H",
		"P1D\nT2H",
		"P1D\x00T2H",
		"P日本T2H",
		"PT🙂H",
		strings.Repeat("P", 128),
		strings.Repeat("9", 128),
		strings.Repeat("P", 4096),
		strings.Repeat("9", 4096) + "D",
	}

	for _, input := range seedInputs {
		f.Add(input)
	}

	f.Fuzz(func(t *testing.T, input string) {
		firstResult, firstErr := devices.ParseInterval(input)
		secondResult, secondErr := devices.ParseInterval(input)

		if (firstErr == nil) != (secondErr == nil) {
			t.Fatalf("ParseInterval error mismatch for %q: first=%v second=%v", input, firstErr, secondErr)
		}

		if firstErr != nil {
			if firstErr.Error() != secondErr.Error() {
				t.Fatalf("ParseInterval error text mismatch for %q: first=%q second=%q", input, firstErr.Error(), secondErr.Error())
			}

			return
		}

		if firstResult != secondResult {
			t.Fatalf("ParseInterval result mismatch for %q: first=%d second=%d", input, firstResult, secondResult)
		}
	})
}
