export interface ConvertResponse {
  ascii: string;
  width: number;
  height: number;
}

export async function convertImage(
  file: File,
  width?: number,
  invert?: boolean,
): Promise<ConvertResponse> {
  const formData = new FormData();
  formData.append('image', file);
  if (width) formData.append('width', width.toString());
  if (invert) formData.append('invert', 'true');

  const response = await fetch('/api/convert', {
    method: 'POST',
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Conversion failed' }));
    throw new Error(error.error || 'Conversion failed');
  }

  return response.json();
}
