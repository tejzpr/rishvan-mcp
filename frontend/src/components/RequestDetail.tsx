import { useState } from 'react';
import { Request } from '../types';
import { respondToRequest } from '../api';

interface RequestDetailProps {
  request: Request | null;
  onResponded: () => void;
}

export default function RequestDetail({ request, onResponded }: RequestDetailProps) {
  const [response, setResponse] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!request) {
    return (
      <div className="flex-1 flex items-center justify-center text-gray-600">
        <div className="text-center">
          <div className="text-4xl mb-3">💬</div>
          <p className="text-sm">Select a request from the sidebar</p>
        </div>
      </div>
    );
  }

  const isPending = request.status === 'pending';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!response.trim() || !isPending) return;

    setSubmitting(true);
    setError(null);
    try {
      await respondToRequest(request.ID, response.trim());
      setResponse('');
      onResponded();
    } catch (err: any) {
      setError(err.message || 'Failed to send response');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="px-6 py-4 border-b border-gray-800 bg-gray-900/50">
        <div className="flex items-center gap-3">
          <span
            className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
              isPending
                ? 'bg-amber-500/20 text-amber-400'
                : 'bg-green-500/20 text-green-400'
            }`}
          >
            {isPending ? 'Awaiting Response' : 'Responded'}
          </span>
          <span className="text-xs text-gray-500">
            {request.app_name}
          </span>
          <span className="text-xs text-gray-600">
            #{request.ID}
          </span>
        </div>
      </div>

      {/* Question */}
      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
          Question
        </div>
        <div className="bg-gray-800/50 rounded-lg p-4 text-gray-200 text-sm leading-relaxed whitespace-pre-wrap">
          {request.question}
        </div>

        {!isPending && request.response && (
          <>
            <div className="mt-6 mb-2 text-xs font-semibold text-gray-500 uppercase tracking-wider">
              Your Response
            </div>
            <div className="bg-green-900/20 border border-green-800/30 rounded-lg p-4 text-green-200 text-sm leading-relaxed whitespace-pre-wrap">
              {request.response}
            </div>
          </>
        )}
      </div>

      {/* Response input */}
      {isPending && (
        <form onSubmit={handleSubmit} className="px-6 py-4 border-t border-gray-800 bg-gray-900/50">
          {error && (
            <div className="mb-3 px-3 py-2 bg-red-900/30 border border-red-800/50 rounded text-red-300 text-xs">
              {error}
            </div>
          )}
          <div className="flex gap-3">
            <textarea
              value={response}
              onChange={(e) => setResponse(e.target.value)}
              placeholder="Type your response..."
              rows={3}
              className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-4 py-3 text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50 resize-none"
              disabled={submitting}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                  handleSubmit(e);
                }
              }}
            />
            <button
              type="submit"
              disabled={submitting || !response.trim()}
              className="self-end px-5 py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-700 disabled:text-gray-500 text-white text-sm font-medium rounded-lg transition-colors"
            >
              {submitting ? 'Sending...' : 'Send'}
            </button>
          </div>
          <p className="mt-2 text-[10px] text-gray-600">Press Cmd+Enter to send</p>
        </form>
      )}
    </div>
  );
}
