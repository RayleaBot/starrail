package panelapi

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The pinned signing script runs in the engine and gives upstream's header:
// the time in seconds, then a hex digest.
func TestSignRunsThePinnedScript(t *testing.T) {
	header, err := sign(context.Background(), "100000001")
	if err != nil {
		t.Fatal(err)
	}
	stamp, digest, _ := strings.Cut(header, ",")
	seconds, err := strconv.ParseInt(stamp, 10, 64)
	if err != nil || time.Since(time.Unix(seconds, 0)).Abs() > time.Minute || !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(digest) {
		t.Fatal("the header is not the time and a digest")
	}
}
