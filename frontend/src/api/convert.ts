export interface ConvertResponse {
  id?: string;
  ascii: string;
  width: number;
  height: number;
}

export type ConvertMode = 'ascii' | 'braille';

export interface ConvertOptions {
  width?: number;
  invert?: boolean;
  mode?: ConvertMode;
  threshold?: number;
}

export async function convertImage(
  file: File,
  options: ConvertOptions = {},
): Promise<ConvertResponse> {
  const formData = new FormData();
  formData.append('image', file);
  if (options.width) formData.append('width', options.width.toString());
  if (options.invert) formData.append('invert', 'true');
  if (options.mode) formData.append('mode', options.mode);
  if (options.threshold) formData.append('threshold', options.threshold.toString());

  const response = await fetch('/api/convert', {
    method: 'POST',
    body: formData,
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Conversion failed' }));
    throw new Error(error.error || 'Conversion failed');
  }

  return response.json();
}
