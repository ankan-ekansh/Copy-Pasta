import { useEffect, useReducer } from 'react';
import { Link } from 'react-router-dom';
import { listPastas, deletePasta, type Pasta } from '../api/pastas';

interface HistoryPanelProps {
  refreshTrigger?: number;
  onTogglePublic?: (id: string, newValue: boolean) => Promise<boolean>;
  publishingIds?: Set<string>;
  onDelete?: (id: string) => void;
}

type State = { pastas: Pasta[]; loading: boolean; error: string; deleteError: string; publishError: string };
type Action =
  | { type: 'fetch' }
  | { type: 'loaded'; pastas: Pasta[] }
  | { type: 'error'; message: string }
  | { type: 'remove'; id: string }
  | { type: 'toggle-public'; id: string; isPublic: boolean }
  | { type: 'delete-error'; message: string }
  | { type: 'publish-error'; message: string }
  | { type: 'clear-errors' };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'fetch': return { ...state, loading: state.pastas.length === 0, error: '', deleteError: '', publishError: '' };
    case 'loaded': return { ...state, pastas: action.pastas, loading: false, error: '' };
    case 'error': return { ...state, loading: false, error: action.message };
    case 'remove': return { ...state, pastas: state.pastas.filter((p) => p.id !== action.id) };
    case 'toggle-public': return { ...state, pastas: state.pastas.map((p) => p.id === action.id ? { ...p, is_public: action.isPublic } : p) };
    case 'delete-error': return { ...state, deleteError: action.message };
    case 'publish-error': return { ...state, publishError: action.message };
    case 'clear-errors': return { ...state, deleteError: '', publishError: '' };
    default: return state;
  }
}

export function HistoryPanel({ refreshTrigger, onTogglePublic, publishingIds: externalPublishingIds, onDelete }: HistoryPanelProps) {
  const [state, dispatch] = useReducer(reducer, { pastas: [], loading: true, error: '', deleteError: '', publishError: '' });

  useEffect(() => {
    let cancelled = false;
    dispatch({ type: 'fetch' });
    listPastas(10)
      .then((items) => { if (!cancelled) dispatch({ type: 'loaded', pastas: items }); })
      .catch((err) => { if (!cancelled) dispatch({ type: 'error', message: err instanceof Error ? err.message : 'Failed to load' }); });
    return () => { cancelled = true; };
  }, [refreshTrigger]);

  const handleDelete = async (id: string) => {
    try {
      dispatch({ type: 'clear-errors' });
      await deletePasta(id);
      dispatch({ type: 'remove', id });
      onDelete?.(id);
    } catch (err) {
      dispatch({ type: 'delete-error', message: err instanceof Error ? err.message : 'Delete failed' });
    }
  };

  const handleCopyLink = (id: string) => {
    const url = `${window.location.origin}/pasta/${encodeURIComponent(id)}`;
    navigator.clipboard?.writeText(url)?.catch(() => {});
  };

  const handleTogglePublic = async (id: string, currentlyPublic: boolean) => {
    if (!onTogglePublic || externalPublishingIds?.has(id)) return;
    dispatch({ type: 'clear-errors' });
    try {
      const performed = await onTogglePublic(id, !currentlyPublic);
      if (performed) {
        dispatch({ type: 'toggle-public', id, isPublic: !currentlyPublic });
      }
    } catch {
      dispatch({ type: 'publish-error', message: 'Failed to update visibility' });
    }
  };

  if (state.loading) {
    return (
      <section className="history-card">
        <p className="eyebrow">📜 Recent pastas</p>
        <p className="mode-hint">Loading...</p>
      </section>
    );
  }

  if (state.error) {
    return (
      <section className="history-card">
        <p className="eyebrow">📜 Recent pastas</p>
        <p className="error-banner">⚠️ {state.error}</p>
      </section>
    );
  }

  if (state.pastas.length === 0) {
    return (
      <section className="history-card">
        <p className="eyebrow">📜 Recent pastas</p>
        <p className="mode-hint">No saved pastas yet. History appears here when persistence is enabled.</p>
      </section>
    );
  }

  return (
    <section className="history-card">
      <p className="eyebrow">📜 Recent pastas</p>
      {state.deleteError && <p className="error-banner">⚠️ {state.deleteError}</p>}
      {state.publishError && <p className="error-banner">⚠️ {state.publishError}</p>}
      <ul className="history-list">
        {state.pastas.map((pasta) => (
          <li key={pasta.id} className="history-item">
            <div className="history-item-info">
              <span className="history-mode">{pasta.mode === 'braille' ? '⠿' : 'A'}</span>
              <span className="history-meta">
                {pasta.width}w • {new Date(pasta.created_at).toLocaleDateString()}
              </span>
            </div>
            <div className="history-item-actions">
              {onTogglePublic && (
                <button
                  type="button"
                  className={`history-btn ${pasta.is_public ? 'history-btn-active' : ''}`}
                  onClick={() => handleTogglePublic(pasta.id, pasta.is_public)}
                  disabled={externalPublishingIds?.has(pasta.id) ?? false}
                  aria-pressed={pasta.is_public}
                  aria-label={pasta.is_public ? 'Unpublish from gallery' : 'Publish to gallery'}
                  title={pasta.is_public ? 'Published ✓' : 'Publish to gallery'}
                >
                  {pasta.is_public ? '🌐' : '📤'}
                </button>
              )}
              <button
                type="button"
                className="history-btn"
                onClick={() => handleCopyLink(pasta.id)}
                aria-label="Copy share link"
              >
                🔗
              </button>
              <Link
                to={`/pasta/${encodeURIComponent(pasta.id)}`}
                className="history-btn"
                aria-label="View pasta"
              >
                👁️
              </Link>
              <button
                type="button"
                className="history-btn history-btn-danger"
                onClick={() => handleDelete(pasta.id)}
                aria-label="Delete pasta"
              >
                🗑️
              </button>
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}
