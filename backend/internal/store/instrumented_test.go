package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/metrics"
	dto "github.com/prometheus/client_model/go"
)

// mockStore implements Store for testing InstrumentedStore.
type mockStore struct {
	saveErr      error
	getResult    *Pasta
	getErr       error
	listResult   []Pasta
	listErr      error
	deleteErr    error
	setPublicErr error
	closed       bool
}

func (m *mockStore) Save(_ context.Context, _ *Pasta) error              { return m.saveErr }
func (m *mockStore) Get(_ context.Context, _ string) (*Pasta, error)     { return m.getResult, m.getErr }
func (m *mockStore) ListBySession(_ context.Context, _ string, _, _ int) ([]Pasta, error) {
	return m.listResult, m.listErr
}
func (m *mockStore) DeleteByOwner(_ context.Context, _, _ string) error  { return m.deleteErr }
func (m *mockStore) SetPublicByOwner(_ context.Context, _, _ string, _ bool) error {
	return m.setPublicErr
}
func (m *mockStore) Close() { m.closed = true }

// mockPingStore adds Ping support.
type mockPingStore struct {
	mockStore
	pingErr error
}

func (m *mockPingStore) Ping(_ context.Context) error { return m.pingErr }

func TestInstrumentedStore_Save_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()
	metrics.DBOperationDuration.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)

	err := s.Save(context.Background(), &Pasta{ID: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counter, _ := metrics.DBOperationsTotal.GetMetricWithLabelValues("save", "success")
	m := &dto.Metric{}
	counter.Write(m)
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 save success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_Save_Error_RecordsErrorStatus(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{saveErr: errors.New("db down")}
	s := NewInstrumented(inner)

	err := s.Save(context.Background(), &Pasta{ID: "test"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	counter, _ := metrics.DBOperationsTotal.GetMetricWithLabelValues("save", "error")
	m := &dto.Metric{}
	counter.Write(m)
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 save error, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_Get_NotFound_IsSuccess(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{getErr: ErrNotFound}
	s := NewInstrumented(inner)

	_, err := s.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// ErrNotFound should be classified as "success" (expected outcome)
	counter, _ := metrics.DBOperationsTotal.GetMetricWithLabelValues("get", "success")
	m := &dto.Metric{}
	counter.Write(m)
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 get success (not found), got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_List_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{listResult: []Pasta{{ID: "a"}, {ID: "b"}}}
	s := NewInstrumented(inner)

	pastas, err := s.ListBySession(context.Background(), "sess", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pastas) != 2 {
		t.Errorf("expected 2 pastas, got %d", len(pastas))
	}

	counter, _ := metrics.DBOperationsTotal.GetMetricWithLabelValues("list", "success")
	m := &dto.Metric{}
	counter.Write(m)
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 list success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_Delete_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)
	s.DeleteByOwner(context.Background(), "id", "sess")

	counter, _ := metrics.DBOperationsTotal.GetMetricWithLabelValues("delete", "success")
	m := &dto.Metric{}
	counter.Write(m)
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 delete success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_SetPublic_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)
	s.SetPublicByOwner(context.Background(), "id", "sess", true)

	counter, _ := metrics.DBOperationsTotal.GetMetricWithLabelValues("set_public", "success")
	m := &dto.Metric{}
	counter.Write(m)
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 set_public success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_Ping_DelegatesToInner(t *testing.T) {
	inner := &mockPingStore{pingErr: nil}
	s := NewInstrumented(inner).(*InstrumentedStore)

	err := s.Ping(context.Background())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestInstrumentedStore_Ping_InnerError(t *testing.T) {
	inner := &mockPingStore{pingErr: errors.New("connection refused")}
	s := NewInstrumented(inner).(*InstrumentedStore)

	err := s.Ping(context.Background())
	if err == nil || err.Error() != "connection refused" {
		t.Errorf("expected 'connection refused', got %v", err)
	}
}

func TestInstrumentedStore_Ping_NotSupported(t *testing.T) {
	inner := &mockStore{} // no Ping method
	s := NewInstrumented(inner).(*InstrumentedStore)

	err := s.Ping(context.Background())
	if !errors.Is(err, ErrPingNotSupported) {
		t.Errorf("expected ErrPingNotSupported, got %v", err)
	}
}

func TestInstrumentedStore_Close_Delegates(t *testing.T) {
	inner := &mockStore{}
	s := NewInstrumented(inner)
	s.Close()
	if !inner.closed {
		t.Error("expected Close to be delegated to inner store")
	}
}

func TestInstrumentedStore_Duration_Recorded(t *testing.T) {
	metrics.DBOperationDuration.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)
	s.Save(context.Background(), &Pasta{ID: "test"})

	observer, _ := metrics.DBOperationDuration.GetMetricWithLabelValues("save")
	m := &dto.Metric{}
	observer.(interface{ Write(*dto.Metric) error }).Write(m)

	if m.GetHistogram().GetSampleCount() != 1 {
		t.Errorf("expected 1 duration observation, got %d", m.GetHistogram().GetSampleCount())
	}
}

func TestNewInstrumented_NilReturnsNil(t *testing.T) {
	s := NewInstrumented(nil)
	if s != nil {
		t.Error("expected nil for nil input")
	}
}

// Suppress unused import warning for time package.
var _ = time.Now
