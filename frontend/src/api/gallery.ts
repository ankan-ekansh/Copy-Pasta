import { type Pasta } from './pastas';

export interface GalleryPasta extends Omit<Pasta, 'is_public'> {
  like_count: number;
  liked_by_me: boolean;
}

export interface GalleryResponse {
  pastas: GalleryPasta[];
}

export async function listGallery(limit = 20, offset = 0): Promise<GalleryPasta[]> {
  const response = await fetch(`/api/gallery?limit=${limit}&offset=${offset}`, {
    credentials: 'include',
  });

  if (!response.ok) {
    if (response.status === 503) return [];
    const error = await response.json().catch(() => ({ error: response.statusText || 'Request failed' }));
    throw new Error(error.error || 'Failed to load gallery');
  }

  const data: GalleryResponse = await response.json();
  return data.pastas || [];
}

export async function likePasta(id: string): Promise<{ like_count: number; liked: boolean }> {
  const response = await fetch(`/api/pastas/${encodeURIComponent(id)}/like`, {
    method: 'POST',
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText || 'Request failed' }));
    throw new Error(error.error || 'Like failed');
  }

  return response.json();
}

export async function unlikePasta(id: string): Promise<{ like_count: number; liked: boolean }> {
  const response = await fetch(`/api/pastas/${encodeURIComponent(id)}/like`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText || 'Request failed' }));
    throw new Error(error.error || 'Unlike failed');
  }

  return response.json();
}
