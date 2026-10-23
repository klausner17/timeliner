package domain

import "testing"

func TestBucketsCoverEveryYearOnce(t *testing.T) {
	for _, c := range []struct{ from, span, n int }{{1900, 100, 7}, {-500, 2001, 200}, {0, 5, 5}, {1990, 1, 1}} {
		next := c.from
		for i := range c.n {
			lo, hi := BucketRange(i, c.from, c.span, c.n)
			if lo != next || hi < lo {
				t.Fatalf("%+v faixa %d = %d–%d, esperava começar em %d", c, i, lo, hi, next)
			}
			for y := lo; y <= hi; y++ {
				if b := BucketOf(y, c.from, c.span, c.n); b != i {
					t.Fatalf("%+v: ano %d na faixa %d, não %d", c, y, b, i)
				}
			}
			next = hi + 1
		}
		if next != c.from+c.span {
			t.Fatalf("%+v termina em %d", c, next-1)
		}
	}
}
