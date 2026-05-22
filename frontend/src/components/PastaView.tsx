import { useState, useEffect, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import { getPasta, type Pasta } from '../api/pastas';
import { ThemeToggle } from './ThemeToggle';

export function PastaView() {
  const { id } = useParams<{ id: string }>();
  const [pasta, setPasta] = useState<Pasta | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [copied, setCopied] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => { if (timerRef.current) clearTimeout(timerRef.current); };
  }, []);

  useEffect(() => {
    if (!id) return;
    getPasta(id)
      .then(setPasta)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [id]);

  const handleCopy = async () => {
    if (!pasta) return;
    try {
      await navigator.clipboard.writeText(pasta.ascii_art);
      setCopied(true);
      timerRef.current = setTimeout(() => setCopied(false), 2000);
    } catch {
      // Clipboard API unavailable or denied
    }
  };

  if (loading) {
    return (
      <div className="app-shell">
        <header className="hero-panel">
          <ThemeToggle />
          <h1>🍝 Loading pasta...</h1>
        </header>
      </div>
    );
  }

  if (error || !pasta) {
    return (
      <div className="app-shell">
        <header className="hero-panel">
          <ThemeToggle />
          <h1>🍝 Copy-Pasta</h1>
          <p className="hero-subtitle">Pasta not found</p>
        </header>
        <main className="app-grid">
          <section className="status-card empty-state">
            <h2>😵 {error || 'This pasta does not exist'}</h2>
            <p>It may have been deleted or the link is incorrect.</p>
            <Link to="/" className="primary-button" style={{ display: 'inline-block', marginTop: '1rem', textDecoration: 'none' }}>
              ← Make your own pasta
            </Link>
          </section>
        </main>
      </div>
    );
  }

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
        <div className="output-column" style={{ gridColumn: '1 / -1' }}>
          <section className="ascii-card">
            <div className="ascii-header">
              <span>
                {pasta.width}×{pasta.height} • Created {new Date(pasta.created_at).toLocaleDateString()}
              </span>
              <div style={{ display: 'flex', gap: '0.5rem' }}>
                <button type="button" className="secondary-button" onClick={handleCopy}>
                  {copied ? '✅ Copied!' : '📋 Copy'}
                </button>
                <Link to="/" className="secondary-button" style={{ textDecoration: 'none' }}>
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
