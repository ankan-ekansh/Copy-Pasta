import { useEffect, useRef, useState } from 'react';

interface AsciiOutputProps {
  ascii: string;
  width: number;
  height: number;
  shareUrl?: string | null;
  pastaId?: string | null;
  isPublic?: boolean;
  onTogglePublic?: (id: string, newValue: boolean) => Promise<void>;
}

export function AsciiOutput({ ascii, width, height, shareUrl, pastaId, isPublic = false, onTogglePublic }: AsciiOutputProps) {
  const [copied, setCopied] = useState(false);
  const [linkCopied, setLinkCopied] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const preRef = useRef<HTMLPreElement>(null);
  const [hasOverflowX, setHasOverflowX] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timeoutId = window.setTimeout(() => setCopied(false), 2000);
    return () => window.clearTimeout(timeoutId);
  }, [copied]);

  useEffect(() => {
    if (!linkCopied) return;
    const timeoutId = window.setTimeout(() => setLinkCopied(false), 2000);
    return () => window.clearTimeout(timeoutId);
  }, [linkCopied]);

  // Detect horizontal overflow for scroll indicator
  useEffect(() => {
    const el = preRef.current;
    if (!el) return;
    const check = () => setHasOverflowX(el.scrollWidth > el.clientWidth);
    check();
    if (typeof ResizeObserver === 'undefined') {
      window.addEventListener('resize', check);
      return () => window.removeEventListener('resize', check);
    }
    const observer = new ResizeObserver(check);
    observer.observe(el);
    return () => observer.disconnect();
  }, [ascii]);

  const handleCopy = async () => {
    if (!navigator.clipboard?.writeText) return;
    try {
      await navigator.clipboard.writeText(ascii);
      setCopied(true);
    } catch { /* clipboard unavailable */ }
  };

  const handleCopyLink = async () => {
    if (!shareUrl || !navigator.clipboard?.writeText) return;
    try {
      await navigator.clipboard.writeText(shareUrl);
      setLinkCopied(true);
    } catch { /* clipboard unavailable */ }
  };

  const handleTogglePublic = async () => {
    if (!pastaId || !onTogglePublic || publishing) return;
    setPublishing(true);
    try {
      await onTogglePublic(pastaId, !isPublic);
    } finally {
      setPublishing(false);
    }
  };

  return (
    <section className="ascii-card">
      <div className="ascii-header">
        <div>
          <p className="eyebrow">Fresh out of the pasta press</p>
          <h2>ASCII masterpiece</h2>
        </div>
        <button type="button" className="secondary-button" onClick={handleCopy}>
          {copied ? 'Copied! 📋' : 'Copy ASCII'}
        </button>
      </div>
      <p className="ascii-dimensions">
        {width} × {height} chars
      </p>
      <div className={`ascii-output-wrapper ${hasOverflowX ? 'has-overflow-x' : ''}`}>
        <pre className="ascii-output" ref={preRef}>{ascii}</pre>
      </div>
      {shareUrl && (
        <div className="ascii-share-row">
          <span className="ascii-share-label">🔗</span>
          <input
            type="text"
            readOnly
            aria-label="Shareable pasta link"
            value={shareUrl}
            className="share-link-input"
          />
          <button type="button" className="secondary-button share-copy-btn" onClick={handleCopyLink}>
            {linkCopied ? 'Copied!' : '📋 Copy link'}
          </button>
          {pastaId && onTogglePublic && (
            <button
              type="button"
              className={`secondary-button publish-btn ${isPublic ? 'publish-btn-active' : ''}`}
              onClick={handleTogglePublic}
              disabled={publishing}
              aria-pressed={isPublic}
              title={isPublic ? 'Published to gallery' : 'Publish to gallery'}
            >
              {publishing ? '⏳' : isPublic ? '🌐 Published' : '📤 Publish'}
            </button>
          )}
        </div>
      )}
    </section>
  );
}
