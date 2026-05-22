import { useEffect, useRef, useReducer, useState } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getPasta, type Pasta } from '../api/pastas';
import { ThemeToggle } from './ThemeToggle';

type FetchState = { pasta: Pasta | null; loading: boolean; error: string };
type FetchAction =
  | { type: 'fetch' }
  | { type: 'success'; pasta: Pasta }
  | { type: 'failure'; message: string };

function fetchReducer(state: FetchState, action: FetchAction): FetchState {
  switch (action.type) {
    case 'fetch': return { pasta: null, loading: true, error: '' };
    case 'success': return { pasta: action.pasta, loading: false, error: '' };
    case 'failure': return { pasta: null, loading: false, error: action.message };
    default: return state;
  }
}

export function PastaView() {
  const { id } = useParams<{ id: string }>();
  const [state, dispatch] = useReducer(fetchReducer, { pasta: null, loading: true, error: '' });
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
  }, []);

  useEffect(() => {
    if (!id) {
      dispatch({ type: 'failure', message: 'No pasta ID provided' });
      return;
    }
    let cancelled = false;
    dispatch({ type: 'fetch' });
    getPasta(id)
      .then((p) => { if (!cancelled) dispatch({ type: 'success', pasta: p }); })
      .catch((err) => { if (!cancelled) dispatch({ type: 'failure', message: err instanceof Error ? err.message : 'Failed to load pasta' }); });
    return () => { cancelled = true; };
  }, [id]);

  const handleCopy = async () => {
    if (!state.pasta || !navigator.clipboard) return;
    try {
      await navigator.clipboard.writeText(state.pasta.ascii_art);
      if (timerRef.current) clearTimeout(timerRef.current);
      setCopied(true);
      timerRef.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard write denied
    }
  };

  if (state.loading) {
    return (
      <div className="app-shell">
        <header className="hero-panel">
          <ThemeToggle />
          <h1>🍝 Loading pasta...</h1>
        </header>
      </div>
    );
  }

  if (state.error || !state.pasta) {
    return (
      <div className="app-shell">
        <header className="hero-panel">
          <ThemeToggle />
          <h1>🍝 Copy-Pasta</h1>
          <p className="hero-subtitle">Something went wrong</p>
        </header>
        <main className="app-grid">
          <section className="status-card empty-state">
            <h2>😵 {state.error || 'This pasta does not exist'}</h2>
            <p>It may have been deleted or the link is incorrect.</p>
            <Link to="/" className="primary-button pasta-view-back-link">
              ← Make your own pasta
            </Link>
          </section>
        </main>
      </div>
    );
  }

  const { pasta } = state;

  return (
    <div className="app-shell">
      <header className="hero-panel">
        <ThemeToggle />
        <div>
          <p className="eyebrow">🔗 Shared pasta</p>
          <h1>🍝 Copy-Pasta</h1>
          <p className="hero-subtitle">
            {pasta.mode === 'braille' ? '⠿ Braille' : 'ABC ASCII'} • {pasta.width} chars wide
          </p>
        </div>
      </header>

      <main className="app-grid">
        <div className="output-column pasta-view-full-width">
          <section className="ascii-card">
            <div className="ascii-header">
              <span>
                {pasta.width}×{pasta.height} • Created {new Date(pasta.created_at).toLocaleDateString()}
              </span>
              <div className="pasta-view-actions">
                <button type="button" className="secondary-button" onClick={handleCopy}>
                  {copied ? '✅ Copied!' : '📋 Copy'}
                </button>
                <Link to="/" className="secondary-button pasta-view-nav-link">
                  🍝 Make your own
                </Link>
              </div>
            </div>
            <pre className="ascii-output">{pasta.ascii_art}</pre>
          </section>
        </div>
      </main>
    </div>
  );
}
