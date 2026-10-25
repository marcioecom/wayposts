package id

import (
	"testing"
	"time"
)

func TestNewRejectsNodeOutOfRange(t *testing.T) {
	if _, err := New(-1); err == nil {
		t.Fatal("esperava erro para nó negativo")
	}
	if _, err := New(MaxNode + 1); err == nil {
		t.Fatal("esperava erro para nó acima do máximo")
	}
	if _, err := New(MaxNode); err != nil {
		t.Fatalf("nó máximo deveria ser válido: %v", err)
	}
}

func TestNextIsUniqueAndMonotonic(t *testing.T) {
	g, err := New(7)
	if err != nil {
		t.Fatal(err)
	}

	const n = 200_000
	seen := make(map[int64]struct{}, n)
	last := int64(0)
	for i := 0; i < n; i++ {
		pid := g.Next()
		if pid <= last {
			t.Fatalf("ID não cresceu: %d <= %d", pid, last)
		}
		if _, dup := seen[pid]; dup {
			t.Fatalf("ID duplicado: %d", pid)
		}
		seen[pid] = struct{}{}
		last = pid
	}
}

func TestTimeRoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 34, 56, 789_000_000, time.UTC)
	g, err := NewWithClock(3, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	pid := g.Next()
	if got := Time(pid); !got.Equal(now.Truncate(time.Millisecond)) {
		t.Fatalf("timestamp: got %s, want %s", got, now)
	}
}

func TestSequenceRolloverAdvancesMillisecond(t *testing.T) {
	now := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	g, err := NewWithClock(0, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	var ids []int64
	for i := 0; i < 70; i++ {
		ids = append(ids, g.Next())
	}

	if got, want := Time(ids[0]).UnixMilli(), now.UnixMilli(); got != want {
		t.Fatalf("primeiro ID: got ms %d, want %d", got, want)
	}
	if got, want := Time(ids[64]).UnixMilli(), now.UnixMilli()+1; got != want {
		t.Fatalf("ID 65: got ms %d, want %d", got, want)
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Fatalf("sequência não monotônica no índice %d", i)
		}
	}
}

func TestClockGoingBackwardsKeepsMonotonic(t *testing.T) {
	now := time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC)
	g, err := NewWithClock(1, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	first := g.Next()
	now = now.Add(-2 * time.Second)
	second := g.Next()

	if second <= first {
		t.Fatalf("ID deveria continuar crescendo com relógio para trás: %d <= %d", second, first)
	}
}
