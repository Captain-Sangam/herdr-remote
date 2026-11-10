package main

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"time"
)

var hidIdleRe = regexp.MustCompile(`"HIDIdleTime"\s*=\s*([0-9]+)`)

// parseHIDIdle reads the nanoseconds since the last keyboard or mouse input
// from `ioreg -c IOHIDSystem -d 4`. With several entries it takes the smallest,
// so the guard fails toward "in use".
func parseHIDIdle(out []byte) (time.Duration, error) {
	ms := hidIdleRe.FindAllSubmatch(out, -1)
	if len(ms) == 0 {
		return 0, errors.New("ioreg: no HIDIdleTime in output")
	}
	best := time.Duration(-1)
	for _, m := range ms {
		n, err := strconv.ParseUint(string(m[1]), 10, 63)
		if err != nil {
			return 0, errors.New("ioreg: HIDIdleTime is not a number")
		}
		if d := time.Duration(n); best < 0 || d < best {
			best = d
		}
	}
	return best, nil
}

// MacIdle is how long the Mac has had no local keyboard or mouse input.
func (r *runner) MacIdle(ctx context.Context) (time.Duration, error) {
	out, err := r.ioregIdle(ctx)
	if err != nil {
		return 0, err
	}
	return parseHIDIdle(out)
}
