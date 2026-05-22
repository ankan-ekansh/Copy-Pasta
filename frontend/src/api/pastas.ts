import { type ConvertMode } from './convert';

export interface Pasta {
  id: string;
  ascii_art: string;
  width: number;
  height: number;
  mode: ConvertMode;
  is_public: boolean;
  created_at: string;
}

export interface ListResponse {
  pastas: Pasta[];
}

export async function getPasta(id: string): Promise<Pasta> {
  const response = await fetch(`/api/pastas/${id}`, {
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Not found' }));
    throw new Error(error.error || 'Failed to load pasta');
  }

  return response.json();
}

export async function listPastas(limit = 20, offset = 0): Promise<Pasta[]> {
  const response = await fetch(`/api/pastas?limit=${limit}&offset=${offset}`, {
    credentials: 'include',
  });

  if (!response.ok) {
    if (response.status === 503) return []; // persistence not configured
    const error = await response.json().catch(() => ({ error: 'Failed to load history' }));
    throw new Error(error.error || 'Failed to load history');
  }

  const data: ListResponse = await response.json();
  return data.pastas || [];
}

export async function deletePasta(id: string): Promise<void> {
  const response = await fetch(`/api/pastas/${id}`, {
    method: 'DELETE',
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Delete failed' }));
    throw new Error(error.error || 'Delete failed');
  }
}
