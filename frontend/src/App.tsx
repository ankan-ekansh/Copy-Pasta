import { useState, useEffect } from 'react';
import { convertImage, type ConvertResponse, type ConvertMode } from './api/convert';
import { AsciiOutput } from './components/AsciiOutput';
import { ImageUploader } from './components/ImageUploader';
import { HistoryPanel } from './components/HistoryPanel';
import { ThemeToggle } from './components/ThemeToggle';
import { Gallery } from './components/Gallery';
import './App.css';

const DEFAULT_WIDTH = 150;

interface SharePreset {
  label: string;
  emoji: string;
  width: number;
  mode: ConvertMode;
}

const SHARE_PRESETS: SharePreset[] = [
  { label: 'iMessage', emoji: '💭', width: 18, mode: 'braille' },
  { label: 'WhatsApp', emoji: '💬', width: 22, mode: 'braille' },
  { label: 'Twitter/X', emoji: '𝕏', width: 30, mode: 'braille' },
  { label: 'Telegram', emoji: '✈️', width: 32, mode: 'braille' },
  { label: 'Discord', emoji: '🎮', width: 45, mode: 'braille' },
  { label: 'Reddit', emoji: '🤖', width: 50, mode: 'braille' },
  { label: 'Desktop', emoji: '🖥️', width: 80, mode: 'braille' },
];

function App() {
  const [file, setFile] = useState<File | null>(null);
  const [width, setWidth] = useState(DEFAULT_WIDTH);
  const [invert, setInvert] = useState(false);
  const [mode, setMode] = useState<ConvertMode>('braille');
  const [result, setResult] = useState<ConvertResponse | null>(null);
  const [error, setError] = useState('');
  const [isConverting, setIsConverting] = useState(false);
  const [historyRefresh, setHistoryRefresh] = useState(0);
  const [view, setView] = useState<'app' | 'gallery'>('app');
  const [settingsOpen, setSettingsOpen] = useState(() => {
    const saved = localStorage.getItem('copy-pasta-settings-open');
    return saved === null ? true : saved === 'true';
  });

  const toggleSettings = () => {
    setSettingsOpen(prev => !prev);
  };

  useEffect(() => {
    localStorage.setItem('copy-pasta-settings-open', String(settingsOpen));
  }, [settingsOpen]);

  const shareUrl = result?.id
    ? `${window.location.origin}/pasta/${encodeURIComponent(result.id)}`
    : null;

  const handleConvert = async () => {
    if (!file) {
      setError('Pick a meme first so we have something to noodle.');
      return;
    }

    try {
      setIsConverting(true);
      setError('');
      const response = await convertImage(file, {
        width,
        invert,
        mode,
      });
      setResult(response);
      setHistoryRefresh((n) => n + 1);
    } catch (conversionError) {
      setResult(null);
      setError(
        conversionError instanceof Error
          ? conversionError.message
          : 'Conversion failed. Please try another image.',
      );
    } finally {
      setIsConverting(false);
    }
  };

  return (
    <div className="app-shell">
      <header className="hero-panel">
        <ThemeToggle />
        <div>
          <p className="eyebrow">🍜 Pixel pasta maker</p>
          <h1>🍝 Copy-Pasta</h1>
          <p className="hero-subtitle">Turn memes into ASCII art</p>
          <p className="hero-description">
            Drop in a cursed image, dial in your character width, and serve it back as gloriously
            nerdy text art.
          </p>
        </div>
        <div className="hero-badges" aria-label="App features">
          <span>📋 Paste friendly</span>
          <span>🎚️ Width controls</span>
          <span>🌗 Invert mode</span>
        </div>
        <nav className="view-tabs" aria-label="Main navigation">
          <button
            type="button"
            className={`view-tab ${view === 'app' ? 'active' : ''}`}
            onClick={() => setView('app')}
            aria-pressed={view === 'app'}
          >
            🎨 Create
          </button>
          <button
            type="button"
            className={`view-tab ${view === 'gallery' ? 'active' : ''}`}
            onClick={() => setView('gallery')}
            aria-pressed={view === 'gallery'}
          >
            🖼️ Gallery
          </button>
        </nav>
      </header>

      {view === 'gallery' ? (
        <Gallery />
      ) : (
      <main className="app-grid">
        <div className="panel-stack">
          <ImageUploader
            onFileSelected={(nextFile) => {
              setFile(nextFile);
              setResult(null);
              setError('');
            }}
          />

          <section className="controls-card">
            <button
              type="button"
              className="controls-header controls-toggle"
              onClick={toggleSettings}
              aria-expanded={settingsOpen}
              aria-controls="controls-panel"
            >
              <div>
                <p className="eyebrow">Season to taste</p>
                <h2 id="controls-heading">Conversion settings</h2>
              </div>
              <div className="controls-header-right">
                <span className="width-pill">{width} chars wide</span>
                <span className={`controls-chevron ${settingsOpen ? 'open' : ''}`}>▾</span>
              </div>
            </button>

            <div id="controls-panel" role="region" aria-labelledby="controls-heading" className={`controls-body ${settingsOpen ? 'open' : ''}`} inert={!settingsOpen ? true : undefined}>
            <div className="controls-body-inner">

            <label className="range-control" htmlFor="width">
              <span>Output width</span>
              <input
                id="width"
                type="range"
                min="60"
                max="250"
                value={width}
                onChange={(event) => setWidth(Number(event.target.value))}
              />
              <div className="range-labels">
                <span>60</span>
                <span>150</span>
                <span>250</span>
              </div>
            </label>

            <label className="checkbox-row" htmlFor="invert">
              <input
                id="invert"
                type="checkbox"
                checked={invert}
                onChange={(event) => setInvert(event.target.checked)}
              />
              <span>Invert brightness for dramatic meme energy</span>
            </label>

            <div className="mode-toggle">
              <p className="mode-label">Conversion mode</p>
              <div className="mode-buttons">
                <button
                  type="button"
                  className={`mode-btn ${mode === 'ascii' ? 'active' : ''}`}
                  onClick={() => setMode('ascii')}
                >
                  ABC ASCII
                </button>
                <button
                  type="button"
                  className={`mode-btn ${mode === 'braille' ? 'active' : ''}`}
                  onClick={() => setMode('braille')}
                >
                  ⠿ Braille
                </button>
              </div>
              <p className="mode-hint">
                {mode === 'braille'
                  ? 'Unicode Braille — higher resolution, great for recognizable memes'
                  : 'Classic ASCII chars — retro terminal aesthetic'}
              </p>
            </div>

            <div className="share-presets">
              <p className="mode-label">📱 Quick share presets</p>
              <div className="preset-chips">
                {SHARE_PRESETS.map((preset) => (
                  <button
                    key={preset.label}
                    type="button"
                    className={`preset-chip ${width === preset.width && mode === preset.mode ? 'active' : ''}`}
                    onClick={() => {
                      setWidth(preset.width);
                      setMode(preset.mode);
                    }}
                  >
                    <span className="preset-emoji">{preset.emoji}</span>
                    <span className="preset-name">{preset.label}</span>
                    <span className="preset-width">{preset.width}w</span>
                  </button>
                ))}
              </div>
              <p className="mode-hint">
                Tap a preset to auto-set width for that platform. Braille works best on narrow screens.
              </p>
            </div>

            <button
              type="button"
              className="primary-button convert-button"
              onClick={handleConvert}
              disabled={!file || isConverting}
            >
              {isConverting ? 'Cooking ASCII…' : 'Convert to ASCII'}
            </button>

            <p className="controls-tip">
              Tip: wider outputs keep more detail, while narrower ones make the joke hit faster.
            </p>

            {error && <p className="error-banner">⚠️ {error}</p>}
            </div>
            </div>
          </section>
        </div>

        <div className="output-column">
          {isConverting && (
            <section className="status-card" aria-live="polite">
              <p className="eyebrow">Simmering...</p>
              <h2>Rendering your fresh pasta</h2>
              <p>Crunching pixels into tasty monospace noodles.</p>
            </section>
          )}

          {result ? (
            <>
              <AsciiOutput ascii={result.ascii} width={result.width} height={result.height} shareUrl={shareUrl} />
            </>
          ) : (
            !isConverting && (
              <section className="status-card empty-state">
                <p className="eyebrow">Ready when you are</p>
                <h2>No ASCII yet</h2>
                <p>
                  Upload a meme, tap convert, and this panel will fill up with beautiful terminal
                  chaos.
                </p>
              </section>
            )
          )}
        </div>

        <div className="history-column">
          <HistoryPanel refreshTrigger={historyRefresh} />
        </div>
      </main>
      )}
    </div>
  );
}

export default App;
