import { useEffect, useState } from 'react';

interface AsciiOutputProps {
  ascii: string;
  width: number;
  height: number;
}

export function AsciiOutput({ ascii, width, height }: AsciiOutputProps) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) {
      return;
    }

    const timeoutId = window.setTimeout(() => setCopied(false), 2000);
    return () => window.clearTimeout(timeoutId);
  }, [copied]);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(ascii);
    setCopied(true);
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
      <pre className="ascii-output">{ascii}</pre>
    </section>
  );
}
