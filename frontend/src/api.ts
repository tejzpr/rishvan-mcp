import { Request } from './types';

const BASE = '';

export async function fetchRequests(appName?: string): Promise<Request[]> {
  const params = appName ? `?app_name=${encodeURIComponent(appName)}` : '';
  const res = await fetch(`${BASE}/api/requests${params}`);
  if (!res.ok) throw new Error(`Failed to fetch requests: ${res.statusText}`);
  return res.json();
}

export async function fetchRequest(id: number): Promise<Request> {
  const res = await fetch(`${BASE}/api/requests/${id}`);
  if (!res.ok) throw new Error(`Failed to fetch request: ${res.statusText}`);
  return res.json();
}

export async function respondToRequest(id: number, response: string): Promise<void> {
  const res = await fetch(`${BASE}/api/requests/${id}/respond`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ response }),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || res.statusText);
  }
}

export async function fetchIDEName(): Promise<string> {
  const res = await fetch(`${BASE}/api/ide`);
  if (!res.ok) throw new Error(`Failed to fetch IDE name: ${res.statusText}`);
  const data = await res.json();
  return data.ide_name;
}

export function subscribeSSE(onNewRequest: (data: { id: number; app_name: string; question: string }) => void): EventSource {
  const es = new EventSource(`${BASE}/api/events`);
  es.addEventListener('new-request', (e) => {
    try {
      const data = JSON.parse(e.data);
      onNewRequest(data);
    } catch {
      // ignore parse errors
    }
  });
  return es;
}
