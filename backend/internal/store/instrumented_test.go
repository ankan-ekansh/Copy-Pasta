package store

import (
	"context"
	"errors"
	"testing"

	"github.com/ankan-ekansh/Copy-Pasta/backend/internal/metrics"
	dto "github.com/prometheus/client_model/go"
)

// mockStore implements Store for testing InstrumentedStore.
type mockStore struct {
	saveErr         error
	getResult       *Pasta
	getErr          error
	listResult      []Pasta
	listErr         error
	deleteErr       error
	setPublicErr    error
	listPublicResult []GalleryPasta
	listPublicErr   error
	likeErr         error
	unlikeErr       error
	likeCount       int
	likedByMe       bool
	likeCountErr    error
	isPublicErr     error
	closed          bool
}

func (m *mockStore) Save(_ context.Context, _ *Pasta) error              { return m.saveErr }
func (m *mockStore) Get(_ context.Context, _ string) (*Pasta, error)     { return m.getResult, m.getErr }
func (m *mockStore) ListBySession(_ context.Context, _ string, _, _ int) ([]Pasta, error) {
	return m.listResult, m.listErr
}
func (m *mockStore) ListPublic(_ context.Context, _ string, _, _ int) ([]GalleryPasta, error) {
	return m.listPublicResult, m.listPublicErr
}
func (m *mockStore) DeleteByOwner(_ context.Context, _, _ string) error  { return m.deleteErr }
func (m *mockStore) SetPublicByOwner(_ context.Context, _, _ string, _ bool) error {
	return m.setPublicErr
}
func (m *mockStore) LikePasta(_ context.Context, _, _ string) error   { return m.likeErr }
func (m *mockStore) UnlikePasta(_ context.Context, _, _ string) error  { return m.unlikeErr }
func (m *mockStore) GetLikeCount(_ context.Context, _, _ string) (int, bool, error) {
	return m.likeCount, m.likedByMe, m.likeCountErr
}
func (m *mockStore) IsPublicPasta(_ context.Context, _ string) error { return m.isPublicErr }
func (m *mockStore) Close()                                          { m.closed = true }

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

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("save", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
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

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("save", "error")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
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
	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("get", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
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

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("list", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 list success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_Delete_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)
	s.DeleteByOwner(context.Background(), "id", "sess")

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("delete", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 delete success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_SetPublic_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)
	s.SetPublicByOwner(context.Background(), "id", "sess", true)

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("set_public", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 set_public success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_IsPublicPasta_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)
	_ = s.IsPublicPasta(context.Background(), "any-id")

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("is_public_pasta", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 is_public_pasta success, got %f", m.GetCounter().GetValue())
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

	observer, err := metrics.DBOperationDuration.GetMetricWithLabelValues("save")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	writer, ok := observer.(interface{ Write(*dto.Metric) error })
	if !ok {
		t.Fatal("observer does not implement Write")
	}
	if err := writer.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}

	if m.GetHistogram().GetSampleCount() != 1 {
		t.Errorf("expected 1 duration observation, got %d", m.GetHistogram().GetSampleCount())
	}
}

func TestInstrumentedStore_ListPublic_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{listPublicResult: []GalleryPasta{{ID: "a"}}}
	s := NewInstrumented(inner)

	pastas, err := s.ListPublic(context.Background(), "sess", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pastas) != 1 {
		t.Errorf("expected 1 pasta, got %d", len(pastas))
	}

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("list_public", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 list_public success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_LikePasta_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)

	err := s.LikePasta(context.Background(), "pasta1", "sess1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("like", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 like success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_UnlikePasta_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{}
	s := NewInstrumented(inner)

	err := s.UnlikePasta(context.Background(), "pasta1", "sess1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("unlike", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 unlike success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_GetLikeCount_RecordsMetrics(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{likeCount: 5, likedByMe: true}
	s := NewInstrumented(inner)

	count, liked, err := s.GetLikeCount(context.Background(), "pasta1", "sess1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 || !liked {
		t.Errorf("expected count=5 liked=true, got count=%d liked=%v", count, liked)
	}

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("get_like_count", "success")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 get_like_count success, got %f", m.GetCounter().GetValue())
	}
}

func TestInstrumentedStore_LikePasta_Error_RecordsErrorStatus(t *testing.T) {
	metrics.DBOperationsTotal.Reset()

	inner := &mockStore{likeErr: errors.New("db down")}
	s := NewInstrumented(inner)

	err := s.LikePasta(context.Background(), "pasta1", "sess1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	counter, err := metrics.DBOperationsTotal.GetMetricWithLabelValues("like", "error")
	if err != nil {
		t.Fatalf("failed to get metric: %v", err)
	}
	m := &dto.Metric{}
	if err := counter.Write(m); err != nil {
		t.Fatalf("failed to write metric: %v", err)
	}
	if m.GetCounter().GetValue() != 1 {
		t.Errorf("expected 1 like error, got %f", m.GetCounter().GetValue())
	}
}

func TestNewInstrumented_NilReturnsNil(t *testing.T) {
	s := NewInstrumented(nil)
	if s != nil {
		t.Error("expected nil for nil input")
	}
}
