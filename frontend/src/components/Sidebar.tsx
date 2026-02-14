import { Request } from '../types';

interface SidebarProps {
  requests: Request[];
  selectedId: number | null;
  onSelect: (id: number) => void;
  ideName: string;
}

function timeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = Math.floor((now - then) / 1000);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

export default function Sidebar({ requests, selectedId, onSelect, ideName }: SidebarProps) {
  const grouped = requests.reduce<Record<string, Request[]>>((acc, req) => {
    if (!acc[req.app_name]) acc[req.app_name] = [];
    acc[req.app_name].push(req);
    return acc;
  }, {});

  const appNames = Object.keys(grouped).sort();

  return (
    <aside className="w-80 bg-gray-900 border-r border-gray-800 flex flex-col h-full overflow-hidden">
      <div className="px-4 py-4 border-b border-gray-800">
        <h1 className="text-lg font-bold text-white tracking-tight">Rishvan</h1>
        <p className="text-xs text-gray-500 mt-0.5">
          {ideName ? `Connected to ${ideName}` : 'Human-in-the-loop assistant'}
        </p>
      </div>
      <div className="flex-1 overflow-y-auto">
        {appNames.length === 0 && (
          <div className="px-4 py-8 text-center text-gray-600 text-sm">
            No requests yet. Waiting for incoming questions...
          </div>
        )}
        {appNames.map((appName) => (
          <div key={appName}>
            <div className="px-4 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider bg-gray-900/50 sticky top-0">
              {appName}
            </div>
            {grouped[appName].map((req) => (
              <button
                key={req.ID}
                onClick={() => onSelect(req.ID)}
                className={`w-full text-left px-4 py-3 border-b border-gray-800/50 transition-colors ${
                  selectedId === req.ID
                    ? 'bg-blue-600/20 border-l-2 border-l-blue-500'
                    : 'hover:bg-gray-800/50 border-l-2 border-l-transparent'
                }`}
              >
                <div className="flex items-center justify-between mb-1">
                  <span
                    className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium ${
                      req.status === 'pending'
                        ? 'bg-amber-500/20 text-amber-400'
                        : 'bg-green-500/20 text-green-400'
                    }`}
                  >
                    {req.status === 'pending' ? 'PENDING' : 'DONE'}
                  </span>
                  <span className="text-[10px] text-gray-600">{timeAgo(req.CreatedAt)}</span>
                </div>
                <p className="text-sm text-gray-300 truncate">{req.question}</p>
              </button>
            ))}
          </div>
        ))}
      </div>
    </aside>
  );
}
