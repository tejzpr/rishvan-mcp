import { useEffect, useState, useCallback, useRef } from 'react';
import { Request } from './types';
import { fetchRequests, fetchSourceName, subscribeSSE } from './api';
import Sidebar from './components/Sidebar';
import RequestDetail from './components/RequestDetail';

export default function App() {
  const [requests, setRequests] = useState<Request[]>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [sourceName, setSourceName] = useState<string>('');
  const notifPermissionRef = useRef(false);

  const loadRequests = useCallback(async () => {
    try {
      const data = await fetchRequests();
      setRequests(data);
    } catch {
      // retry silently
    } finally {
      setLoading(false);
    }
  }, []);

  // Request notification permission on mount
  useEffect(() => {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission().then((perm) => {
        notifPermissionRef.current = perm === 'granted';
      });
    } else if ('Notification' in window && Notification.permission === 'granted') {
      notifPermissionRef.current = true;
    }
  }, []);

  // Initial load
  useEffect(() => {
    loadRequests();
    fetchSourceName().then(setSourceName).catch(() => {});
  }, [loadRequests]);

  // SSE subscription
  useEffect(() => {
    const es = subscribeSSE((data) => {
      // Show browser notification
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(`New request from ${data.app_name}`, {
          body: data.question.slice(0, 120),
          tag: `rishvan-${data.id}`,
        });
      }

      // Reload requests list
      loadRequests();

      // Auto-select the new request
      setSelectedId(data.id);
    });

    // Fallback polling
    const interval = setInterval(loadRequests, 5000);

    return () => {
      es.close();
      clearInterval(interval);
    };
  }, [loadRequests]);

  const selectedRequest = requests.find((r) => r.ID === selectedId) || null;

  return (
    <div className="flex h-screen bg-gray-950 text-gray-100">
      <Sidebar
        requests={requests}
        selectedId={selectedId}
        onSelect={setSelectedId}
        sourceName={sourceName}
      />
      <RequestDetail
        request={selectedRequest}
        onResponded={loadRequests}
      />
    </div>
  );
}
