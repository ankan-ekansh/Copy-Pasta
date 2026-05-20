import { useEffect, useId, useMemo, useRef, useState } from 'react';

interface ImageUploaderProps {
  onFileSelected: (file: File) => void;
}

const ACCEPTED_TYPES = ['image/jpeg', 'image/png', 'image/gif'];

function isSupportedImage(file: File) {
  return ACCEPTED_TYPES.includes(file.type);
}

export function ImageUploader({ onFileSelected }: ImageUploaderProps) {
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [dragActive, setDragActive] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [feedback, setFeedback] = useState('');

  const previewUrl = useMemo(() => {
    if (!selectedFile) return '';
    return URL.createObjectURL(selectedFile);
  }, [selectedFile]);

  useEffect(() => {
    return () => {
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);

  useEffect(() => {
    const handlePaste = (event: ClipboardEvent) => {
      const file = Array.from(event.clipboardData?.items ?? [])
        .filter((item) => item.kind === 'file')
        .map((item) => item.getAsFile())
        .find((item): item is File => item !== null && isSupportedImage(item));

      if (!file) {
        return;
      }

      event.preventDefault();
      setFeedback(`Pasted ${file.name || 'clipboard image'} ✨`);
      setSelectedFile(file);
      onFileSelected(file);
    };

    window.addEventListener('paste', handlePaste);
    return () => window.removeEventListener('paste', handlePaste);
  }, [onFileSelected]);

  const handleFile = (file: File | undefined) => {
    if (!file) {
      return;
    }

    if (!isSupportedImage(file)) {
      setFeedback('Please choose a JPEG, PNG, or GIF.');
      return;
    }

    setFeedback(`${file.name} is ready for pasta-fication 🍝`);
    setSelectedFile(file);
    onFileSelected(file);
  };

  return (
    <section className="uploader-card">
      <div
        className={`dropzone ${dragActive ? 'dropzone-active' : ''}`}
        onDragEnter={(event) => {
          event.preventDefault();
          setDragActive(true);
        }}
        onDragOver={(event) => {
          event.preventDefault();
          setDragActive(true);
        }}
        onDragLeave={(event) => {
          event.preventDefault();
          setDragActive(false);
        }}
        onDrop={(event) => {
          event.preventDefault();
          setDragActive(false);
          handleFile(event.dataTransfer.files[0]);
        }}
      >
        <input
          ref={inputRef}
          id={inputId}
          className="sr-only"
          type="file"
          accept={ACCEPTED_TYPES.join(',')}
          onChange={(event) => handleFile(event.target.files?.[0])}
        />
        <div className="dropzone-copy">
          <span className="dropzone-emoji" aria-hidden="true">
            🖼️
          </span>
          <h2>Drop a meme, pick a file, or paste from clipboard</h2>
          <p>
            JPEG, PNG, and GIF are welcome. We&apos;ll turn every spicy pixel into tasty ASCII.
          </p>
          <div className="dropzone-actions">
            <button type="button" className="primary-button" onClick={() => inputRef.current?.click()}>
              Choose image
            </button>
            <label className="ghost-pill" htmlFor={inputId}>
              Ctrl/Cmd + V also works
            </label>
          </div>
        </div>
      </div>

      {feedback && <p className="uploader-feedback">{feedback}</p>}

      {selectedFile && previewUrl && (
        <div className="preview-card">
          <div>
            <p className="eyebrow">Preview</p>
            <h3>{selectedFile.name || 'Clipboard image'}</h3>
            <p className="preview-meta">{Math.max(1, Math.round(selectedFile.size / 1024))} KB of meme energy</p>
          </div>
          <img src={previewUrl} alt="Selected upload preview" className="preview-image" />
        </div>
      )}
    </section>
  );
}
