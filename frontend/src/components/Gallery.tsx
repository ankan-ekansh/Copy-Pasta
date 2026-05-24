import { useState, useEffect, useRef, useCallback } from 'react';
import { listGallery, likePasta, unlikePasta, type GalleryPasta } from '../api/gallery';

const PAGE_SIZE = 20;

function relativeTime(dateStr: string) {
  const diff = Date.now() - new Date(dateStr).getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  if (days < 30) return `${days}d ago`;
  return new Date(dateStr).toLocaleDateString();
}

export function Gallery() {
  const [pastas, setPastas] = useState<GalleryPasta[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedPasta, setSelectedPasta] = useState<GalleryPasta | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const loadingRef = useRef(false);

  useEffect(() => {
    let ignore = false;
    listGallery(PAGE_SIZE, 0).then(items => {
      if (!ignore) {
        setPastas(items);
        setHasMore(items.length === PAGE_SIZE);
        setLoading(false);
      }
    }).catch(err => {
      if (!ignore) {
        setError(err instanceof Error ? err.message : 'Failed to load gallery');
        setLoading(false);
      }
    });
    return () => { ignore = true; };
  }, []);

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

  const handleModalClose = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) setSelectedPasta(null);
  }, []);

  // Close modal on Escape
  useEffect(() => {
    if (!selectedPasta) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setSelectedPasta(null);
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [selectedPasta]);

  // Get truncated lines for preview
  const getPreviewLines = (art: string, maxLines: number = 30) => {
    const lines = art.split('\n');
    return lines.slice(0, maxLines).join('\n');
  };

  return (
    <div className="gallery">
      <div className="gallery-header">
        <h2>Community Creations</h2>
        <p className="gallery-subtitle">ASCII masterpieces from the community — click any to view full art</p>
      </div>

      {error && <div className="gallery-error">⚠️ {error}</div>}

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
            <div
              className="gallery-card-preview"
              onClick={() => setSelectedPasta(pasta)}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') setSelectedPasta(pasta); }}
              aria-label="View full ASCII art"
            >
              <pre className="gallery-card-ascii">
                {getPreviewLines(pasta.ascii_art)}
              </pre>
              <div className="gallery-card-fade" />
            </div>

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

      {hasMore && !loading && !error && pastas.length > 0 && (
        <button type="button" className="gallery-load-more" onClick={handleLoadMore}>
          Load more
        </button>
      )}

      {/* Modal overlay for full art view */}
      {selectedPasta && (
        <div className="gallery-modal-backdrop" onClick={handleModalClose} role="dialog" aria-modal="true" aria-label="Full ASCII art view">
          <div className="gallery-modal">
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
                >
                  ✕
                </button>
              </div>
            </div>
            <div className="gallery-modal-body">
              <pre className="gallery-modal-ascii">{selectedPasta.ascii_art}</pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
