import { useState, useEffect, useRef, useCallback, useMemo, type MouseEvent } from 'react';
import { createPortal } from 'react-dom';
import { listGallery, likePasta, unlikePasta, type GalleryPasta } from '../api/gallery';

const PAGE_SIZE = 20;

function relativeTime(dateStr: string) {
  const date = new Date(dateStr);
  if (isNaN(date.getTime())) return 'Unknown';
  const diff = Date.now() - date.getTime();
  if (diff < 0) return 'just now';
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  return date.toLocaleDateString();
}

export function Gallery() {
  const [pastas, setPastas] = useState<GalleryPasta[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPasta, setSelectedPasta] = useState<GalleryPasta | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const loadingRef = useRef(false);

  const mountedRef = useRef(true);
  useEffect(() => { return () => { mountedRef.current = false; }; }, []);

  const loadInitialRef = useRef(false);

  useEffect(() => {
    if (loadInitialRef.current) return;
    loadInitialRef.current = true;
    listGallery(PAGE_SIZE, 0).then(items => {
      if (!mountedRef.current) return;
      setPastas(items);
      setHasMore(items.length === PAGE_SIZE);
      setLoading(false);
    }).catch(err => {
      if (!mountedRef.current) return;
      setError(err instanceof Error ? err.message : 'Failed to load gallery');
      setLoading(false);
    });
  }, []);

  const retryInitialLoad = () => {
    setLoading(true);
    setError(null);
    listGallery(PAGE_SIZE, 0).then(items => {
      if (!mountedRef.current) return;
      setPastas(items);
      setHasMore(items.length === PAGE_SIZE);
      setOffset(0);
      setLoading(false);
    }).catch(err => {
      if (!mountedRef.current) return;
      setError(err instanceof Error ? err.message : 'Failed to load gallery');
      setLoading(false);
    });
  };

  const [likingIds, setLikingIds] = useState<Set<string>>(new Set());
  const likingRef = useRef<Set<string>>(new Set());

  const handleLike = async (id: string, currentlyLiked: boolean) => {
    if (likingRef.current.has(id)) return;
    likingRef.current.add(id);
    setLikingIds(new Set(likingRef.current));
    try {
      const result = currentlyLiked
        ? await unlikePasta(id)
        : await likePasta(id);

      setPastas(prev => prev.map(p =>
        p.id === id
          ? { ...p, like_count: result.like_count, liked_by_me: result.liked_by_me }
          : p
      ));
      // Update modal if open
      setSelectedPasta(prev =>
        prev?.id === id
          ? { ...prev, like_count: result.like_count, liked_by_me: result.liked_by_me }
          : prev
      );
    } catch {
      // silently ignore like errors
    } finally {
      likingRef.current.delete(id);
      setLikingIds(new Set(likingRef.current));
    }
  };

  const handleLoadMore = async () => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setLoading(true);
    setError(null);
    try {
      const newOffset = offset + PAGE_SIZE;
      const items = await listGallery(PAGE_SIZE, newOffset);
      setPastas(prev => [...prev, ...items]);
      setHasMore(items.length === PAGE_SIZE);
      setOffset(newOffset);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load more');
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard?.writeText(text)?.catch(() => {});
  };

  const handleModalClose = useCallback((e: MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) setSelectedPasta(null);
  }, []);

  const modalRef = useRef<HTMLDivElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);

  // Modal: Escape to close, scroll lock, focus trap, inert background
  useEffect(() => {
    if (!selectedPasta) return;
    const previousFocus = document.activeElement as HTMLElement | null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';

    // Mark #root as inert (modal is portaled to document.body, outside #root)
    const appRoot = document.getElementById('root');
    if (appRoot) {
      appRoot.setAttribute('inert', '');
      appRoot.setAttribute('aria-hidden', 'true');
    }

    closeButtonRef.current?.focus();

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setSelectedPasta(null);
        return;
      }
      // Focus trap
      if (e.key === 'Tab' && modalRef.current) {
        const focusable = modalRef.current.querySelectorAll<HTMLElement>(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );
        if (focusable.length === 0) return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = previousOverflow;
      if (appRoot) {
        appRoot.removeAttribute('inert');
        appRoot.removeAttribute('aria-hidden');
      }
      previousFocus?.focus();
    };
  }, [selectedPasta]);

  // Precompute preview lines and font sizes to avoid splitting on every render
  const previewMap = useMemo(() => {
    const map = new Map<string, { text: string; fontSize: string }>();
    for (const pasta of pastas) {
      const lines = pasta.ascii_art.split('\n');
      const preview = lines.slice(0, 30).join('\n');
      const maxLineLen = Math.max(...lines.slice(0, 30).map(l => l.length), 1);
      // Target ~280px card content width; scale font to fit
      const fontSize = Math.min(5.5, Math.max(2.5, 280 / (maxLineLen * 0.6)));
      map.set(pasta.id, { text: preview, fontSize: `${fontSize.toFixed(1)}px` });
    }
    return map;
  }, [pastas]);

  return (
    <div className="gallery">
      <div className="gallery-header">
        <h2>Community Creations</h2>
        <p className="gallery-subtitle">ASCII masterpieces from the community — click any to view full art</p>
      </div>

      {error && (
        <div className="gallery-error">
          <span>⚠️ {error}</span>
          {pastas.length === 0 && (
            <button type="button" className="gallery-retry-btn" onClick={retryInitialLoad}>Retry</button>
          )}
        </div>
      )}

      {pastas.length === 0 && !loading && !error && (
        <div className="gallery-empty">
          <span className="gallery-empty-icon">🎨</span>
          <h3>No public pastas yet</h3>
          <p>Be the first to publish! Use the "Publish" button in your history panel.</p>
        </div>
      )}

      <div className="gallery-grid">
        {pastas.map(pasta => (
          <article key={pasta.id} className="gallery-card">
            <button
              type="button"
              className="gallery-card-preview"
              onClick={() => setSelectedPasta(pasta)}
              aria-label="View full ASCII art"
            >
              <pre className="gallery-card-ascii" style={{ fontSize: previewMap.get(pasta.id)?.fontSize }}>
                {previewMap.get(pasta.id)?.text}
              </pre>
              <div className="gallery-card-fade" />
            </button>

            <div className="gallery-card-footer">
              <div className="gallery-card-meta">
                <span className="gallery-card-mode-badge">
                  {pasta.mode === 'braille' ? '⣿ Braille' : 'ABC ASCII'}
                </span>
                <span className="gallery-card-date">{relativeTime(pasta.created_at)}</span>
              </div>
              <div className="gallery-card-actions">
                <button
                  type="button"
                  className={`gallery-like-btn ${pasta.liked_by_me ? 'liked' : ''}`}
                  onClick={() => handleLike(pasta.id, pasta.liked_by_me)}
                  disabled={likingIds.has(pasta.id)}
                  aria-pressed={pasta.liked_by_me}
                  aria-label={`${pasta.liked_by_me ? 'Unlike' : 'Like'} (${pasta.like_count} likes)`}
                >
                  {pasta.liked_by_me ? '❤️' : '🤍'} {pasta.like_count}
                </button>
                <button
                  type="button"
                  className="gallery-copy-btn"
                  onClick={() => copyToClipboard(pasta.ascii_art)}
                  aria-label="Copy ASCII art"
                  title="Copy"
                >
                  📋
                </button>
              </div>
            </div>
          </article>
        ))}

        {loading && Array.from({ length: 3 }).map((_, i) => (
          <div key={`skeleton-${i}`} className="gallery-card gallery-skeleton">
            <div className="gallery-skeleton-preview" />
            <div className="gallery-skeleton-footer">
              <div className="gallery-skeleton-bar" style={{ width: '40%' }} />
              <div className="gallery-skeleton-bar" style={{ width: '20%' }} />
            </div>
          </div>
        ))}
      </div>

      {hasMore && !loading && pastas.length > 0 && (
        <button type="button" className="gallery-load-more" onClick={handleLoadMore}>
          {error ? 'Retry' : 'Load more'}
        </button>
      )}

      {/* Modal rendered via portal to document.body so #root can be inerted */}
      {selectedPasta && createPortal(
        <div className="gallery-modal-backdrop" onClick={handleModalClose}>
          <div className="gallery-modal" ref={modalRef} role="dialog" aria-modal="true" aria-label="Full ASCII art view" tabIndex={-1}>
            <div className="gallery-modal-header">
              <div className="gallery-modal-meta">
                <span className="gallery-card-mode-badge">
                  {selectedPasta.mode === 'braille' ? '⣿ Braille' : 'ABC ASCII'}
                </span>
                <span>{selectedPasta.width}×{selectedPasta.height}</span>
                <span>{relativeTime(selectedPasta.created_at)}</span>
              </div>
              <div className="gallery-modal-actions">
                <button
                  type="button"
                  className={`gallery-like-btn ${selectedPasta.liked_by_me ? 'liked' : ''}`}
                  onClick={() => handleLike(selectedPasta.id, selectedPasta.liked_by_me)}
                  disabled={likingIds.has(selectedPasta.id)}
                  aria-pressed={selectedPasta.liked_by_me}
                >
                  {selectedPasta.liked_by_me ? '❤️' : '🤍'} {selectedPasta.like_count}
                </button>
                <button
                  type="button"
                  className="gallery-modal-copy-btn"
                  onClick={() => copyToClipboard(selectedPasta.ascii_art)}
                >
                  📋 Copy
                </button>
                <button
                  type="button"
                  className="gallery-modal-close"
                  onClick={() => setSelectedPasta(null)}
                  aria-label="Close"
                  ref={closeButtonRef}
                >
                  ✕
                </button>
              </div>
            </div>
            <div className="gallery-modal-body">
              <pre className="gallery-modal-ascii">{selectedPasta.ascii_art}</pre>
            </div>
          </div>
        </div>,
        document.body
      )}
    </div>
  );
}
