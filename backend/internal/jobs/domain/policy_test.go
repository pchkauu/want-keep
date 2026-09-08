package domain

import (
	"testing"
	"time"
)

func TestRetryPolicyBounds(t *testing.T) {
	p := DefaultRetryPolicy()
	previous := time.Duration(0)
	for i := 1; i <= 100; i++ {
		d := p.Delay(i, 1)
		if d < previous || d > 5*time.Minute {
			t.Fatal("unbounded backoff")
		}
		if p.Delay(i, 0) < d*8/10 {
			t.Fatal("unbounded jitter")
		}
		previous = d
	}
}
