import { useState, useEffect, useReducer } from 'react';
import { Link } from 'react-router-dom';
import { listPastas, deletePasta, type Pasta } from '../api/pastas';

interface HistoryPanelProps {
  refreshTrigger?: number;
}

type State = { pastas: Pasta[]; loading: boolean; error: string };
type Action =
  | { type: 'fetch' }
  | { type: 'loaded'; pastas: Pasta[] }
  | { type: 'error'; message: string }
  | { type: 'remove'; id: string };

function reducer(state: State, action: Action): State {
  switch (action.type) {
    case 'fetch': return { ...state, loading: true, error: '' };
    case 'loaded': return { pastas: action.pastas, loading: false, error: '' };
    case 'error': return { ...state, loading: false, error: action.message };
    case 'remove': return { ...state, pastas: state.pastas.filter((p) => p.id !== action.id) };
    default: return state;
  }
}

export function HistoryPanel({ refreshTrigger }: HistoryPanelProps) {
  const [state, dispatch] = useReducer(reducer, { pastas: [], loading: true, error: '' });
  const [deleteError, setDeleteError] = useState('');

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
      setDeleteError('');
      await deletePasta(id);
      dispatch({ type: 'remove', id });
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Delete failed');
    }
  };

  const handleCopyLink = (id: string) => {
    const url = `${window.location.origin}/pasta/${id}`;
    navigator.clipboard.writeText(url).catch(() => {
      // Clipboard API unavailable or denied — silent fallback
    });
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
        <p className="mode-hint">No saved pastas yet. Convert an image to get started!</p>
      </section>
    );
  }

  return (
    <section className="history-card">
      <p className="eyebrow">📜 Recent pastas</p>
      {deleteError && <p className="error-banner">⚠️ {deleteError}</p>}
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
              <button
                type="button"
                className="history-btn"
                onClick={() => handleCopyLink(pasta.id)}
                aria-label="Copy share link"
              >
                🔗
              </button>
              <Link
                to={`/pasta/${pasta.id}`}
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
