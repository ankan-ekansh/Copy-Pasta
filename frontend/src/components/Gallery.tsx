import { useState, useEffect } from 'react';
import { listGallery, likePasta, unlikePasta, type GalleryPasta } from '../api/gallery';

interface GalleryProps {
  onBack: () => void;
}

const PAGE_SIZE = 20;

export function Gallery({ onBack }: GalleryProps) {
  const [pastas, setPastas] = useState<GalleryPasta[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);

  const loadGallery = async (pageOffset: number): Promise<boolean> => {
    setLoading(true);
    setError(null);
    try {
      const items = await listGallery(PAGE_SIZE, pageOffset);
      if (pageOffset === 0) {
        setPastas(items);
      } else {
        setPastas(prev => [...prev, ...items]);
      }
      setHasMore(items.length === PAGE_SIZE);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load gallery');
      return false;
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    let ignore = false;
    // offset starts at 0; handleLoadMore is only reachable when pastas are loaded
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

  const handleLike = async (id: string, currentlyLiked: boolean) => {
    if (likingIds.has(id)) return;
    setLikingIds(prev => new Set(prev).add(id));
    try {
      const result = currentlyLiked
        ? await unlikePasta(id)
        : await likePasta(id);

      setPastas(prev => prev.map(p =>
        p.id === id
          ? { ...p, like_count: result.like_count, liked_by_me: result.liked_by_me }
          : p
      ));
    } catch {
      // silently ignore like errors
    } finally {
      setLikingIds(prev => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  };

  const handleLoadMore = async () => {
    const newOffset = offset + PAGE_SIZE;
    const success = await loadGallery(newOffset);
    if (success) setOffset(newOffset);
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard?.writeText(text)?.catch(() => {});
  };

  return (
    <div className="gallery">
      <div className="gallery-header">
        <button type="button" className="gallery-back-btn" onClick={onBack}>← Back</button>
        <h2>🖼️ Public Gallery</h2>
        <p className="gallery-subtitle">Community ASCII masterpieces</p>
      </div>

      {error && <div className="gallery-error">{error}</div>}

      {pastas.length === 0 && !loading && !error && (
        <div className="gallery-empty">
          <p>No public pastas yet. Be the first to publish! 🎨</p>
          <p className="gallery-empty-hint">Use the "Publish" button on your pastas to share them here.</p>
        </div>
      )}

      <div className="gallery-grid">
        {pastas.map(pasta => (
          <div key={pasta.id} className="gallery-card">
            <button
              type="button"
              className="gallery-card-preview"
              onClick={() => setExpandedId(expandedId === pasta.id ? null : pasta.id)}
              aria-expanded={expandedId === pasta.id}
              aria-label={`Toggle preview for pasta ${pasta.id}`}
            >
              <pre className="gallery-card-ascii">
                {pasta.ascii_art.slice(0, 500)}{pasta.ascii_art.length > 500 ? '...' : ''}
              </pre>
            </button>

            <div className="gallery-card-footer">
              <div className="gallery-card-meta">
                <span className="gallery-card-mode">{pasta.mode === 'braille' ? '⣿' : 'A'}</span>
                <span className="gallery-card-size">{pasta.width}w</span>
                <span className="gallery-card-date">
                  {new Date(pasta.created_at).toLocaleDateString()}
                </span>
              </div>
              <div className="gallery-card-actions">
                <button
                  type="button"
                  className={`gallery-like-btn ${pasta.liked_by_me ? 'liked' : ''}`}
                  onClick={() => handleLike(pasta.id, pasta.liked_by_me)}
                  disabled={likingIds.has(pasta.id)}
                  aria-pressed={pasta.liked_by_me}
                  aria-label={`${pasta.liked_by_me ? 'Unlike' : 'Like'} (${pasta.like_count} likes)`}
                  title={pasta.liked_by_me ? 'Unlike' : 'Like'}
                >
                  {pasta.liked_by_me ? '❤️' : '🤍'} {pasta.like_count}
                </button>
                <button
                  type="button"
                  className="gallery-copy-btn"
                  onClick={() => copyToClipboard(pasta.ascii_art)}
                  aria-label="Copy ASCII art"
                  title="Copy ASCII"
                >
                  📋
                </button>
              </div>
            </div>

            {expandedId === pasta.id && (
              <div className="gallery-card-expanded">
                <pre className="gallery-card-full-ascii">{pasta.ascii_art}</pre>
                <button
                  type="button"
                  className="gallery-copy-full-btn"
                  onClick={() => copyToClipboard(pasta.ascii_art)}
                >
                  📋 Copy full ASCII
                </button>
              </div>
            )}
          </div>
        ))}
      </div>

      {hasMore && !loading && (
        <button type="button" className="gallery-load-more" onClick={handleLoadMore}>
          Load more
        </button>
      )}

      {loading && <div className="gallery-loading">Loading...</div>}
    </div>
  );
}
