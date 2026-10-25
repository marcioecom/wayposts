package id

import (
	"errors"
	"sync"
	"time"
)

// Layout de 53 bits: 41 de timestamp em ms, 6 de nó e 6 de sequência. O ID
// precisa caber em float64, limite 2^53, porque é usado como score de ZSET.
const (
	// EpochMillis é 2026-01-01T00:00:00Z.
	EpochMillis int64 = 1767225600000

	nodeBits     = 6
	sequenceBits = 6

	MaxNode int64 = (1 << nodeBits) - 1

	maxSequence = (1 << sequenceBits) - 1
	timeShift   = nodeBits + sequenceBits
	nodeShift   = sequenceBits
)

type Generator struct {
	mu         sync.Mutex
	node       int64
	lastMillis int64
	sequence   int64
	now        func() time.Time
}

func New(node int64) (*Generator, error) {
	return NewWithClock(node, time.Now)
}

func NewWithClock(node int64, now func() time.Time) (*Generator, error) {
	if node < 0 || node > MaxNode {
		return nil, errors.New("id: node fora do intervalo 0..63")
	}
	if now == nil {
		return nil, errors.New("id: relógio nil")
	}
	return &Generator{node: node, now: now}, nil
}

func (g *Generator) Next() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	ms := g.now().UnixMilli()
	if ms < g.lastMillis {
		// Relógio andou para trás: mantém a monotonicidade esperando o tempo
		// registrado. Skew maior que alguns segundos deve ser tratado no runbook.
		ms = g.lastMillis
	}

	if ms == g.lastMillis {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			ms++
		}
	} else {
		g.sequence = 0
	}

	g.lastMillis = ms
	return ((ms - EpochMillis) << timeShift) | (g.node << nodeShift) | g.sequence
}

func Time(pid int64) time.Time {
	ms := (pid >> timeShift) + EpochMillis
	return time.UnixMilli(ms).UTC()
}

func Float(pid int64) float64 {
	return float64(pid)
}
