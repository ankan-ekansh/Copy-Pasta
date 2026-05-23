package store

import (
	"context"
	"time"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/metrics"
)

// InstrumentedStore wraps a Store and records Prometheus metrics for each operation.
type InstrumentedStore struct {
	inner Store
}

// NewInstrumented wraps the given store with Prometheus instrumentation.
func NewInstrumented(s Store) Store {
	if s == nil {
		return nil
	}
	return &InstrumentedStore{inner: s}
}

func (s *InstrumentedStore) Save(ctx context.Context, p *Pasta) error {
	start := time.Now()
	err := s.inner.Save(ctx, p)
	s.record("save", start, err)
	return err
}

func (s *InstrumentedStore) Get(ctx context.Context, id string) (*Pasta, error) {
	start := time.Now()
	p, err := s.inner.Get(ctx, id)
	s.record("get", start, err)
	return p, err
}

func (s *InstrumentedStore) ListBySession(ctx context.Context, sessionID string, limit, offset int) ([]Pasta, error) {
	start := time.Now()
	pastas, err := s.inner.ListBySession(ctx, sessionID, limit, offset)
	s.record("list", start, err)
	return pastas, err
}

func (s *InstrumentedStore) DeleteByOwner(ctx context.Context, id, sessionID string) error {
	start := time.Now()
	err := s.inner.DeleteByOwner(ctx, id, sessionID)
	s.record("delete", start, err)
	return err
}

func (s *InstrumentedStore) SetPublicByOwner(ctx context.Context, id, sessionID string, isPublic bool) error {
	start := time.Now()
	err := s.inner.SetPublicByOwner(ctx, id, sessionID, isPublic)
	s.record("set_public", start, err)
	return err
}

func (s *InstrumentedStore) Close() {
	s.inner.Close()
}

// Ping delegates to the inner store if it implements pinging.
func (s *InstrumentedStore) Ping(ctx context.Context) error {
	type pinger interface {
		Ping(ctx context.Context) error
	}
	if p, ok := s.inner.(pinger); ok {
		return p.Ping(ctx)
	}
	return nil
}

func (s *InstrumentedStore) record(operation string, start time.Time, err error) {
	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil && err != ErrNotFound {
		status = "error"
	}
	metrics.DBOperationsTotal.WithLabelValues(operation, status).Inc()
	metrics.DBOperationDuration.WithLabelValues(operation).Observe(duration)
}
