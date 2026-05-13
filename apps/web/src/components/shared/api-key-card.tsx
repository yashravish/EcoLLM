'use client';

import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { formatRelativeTime } from '@/lib/utils';
import { cn } from '@/lib/utils';
import type { ApiKey } from '@/types';

interface ApiKeyCardProps {
  apiKey: ApiKey;
  onRevoke: (id: string) => void;
  revoking?: boolean;
}

export function ApiKeyCard({ apiKey, onRevoke, revoking = false }: ApiKeyCardProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const isRevoked = Boolean(apiKey.revoked_at);

  return (
    <>
      <div className={cn(
        'group flex items-center gap-4 rounded-lg border px-5 py-4 transition-colors',
        isRevoked
          ? 'border-eco-700 bg-eco-800/30 opacity-50'
          : 'border-eco-600 bg-eco-800 hover:border-eco-500 hover:bg-eco-750',
      )}>
        {/* Status dot */}
        <div className={cn(
          'h-2 w-2 flex-shrink-0 rounded-full',
          isRevoked ? 'bg-eco-500' : 'bg-accent shadow-[0_0_6px_rgba(0,232,122,0.5)]',
        )} />

        {/* Main content */}
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-3">
            <span className="text-sm font-semibold text-eco-50">{apiKey.name}</span>
            <code className="font-mono text-xs text-eco-400">{apiKey.key_prefix}••••••••</code>
            {isRevoked && <span className="text-xs text-eco-400">Revoked</span>}
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-x-2.5 gap-y-1">
            {apiKey.scopes.map((scope) => (
              <span
                key={scope}
                className="rounded border border-eco-600 bg-eco-900 px-1.5 py-0.5 font-mono text-[11px] text-eco-300"
              >
                {scope}
              </span>
            ))}
            <span className="text-eco-600 select-none">·</span>
            <span className="text-xs text-eco-400">Created {formatRelativeTime(apiKey.created_at)}</span>
            <span className="text-eco-600 select-none">·</span>
            <span className="text-xs text-eco-400">
              {apiKey.last_used_at ? `Last used ${formatRelativeTime(apiKey.last_used_at)}` : 'Never used'}
            </span>
            {apiKey.expires_at && (
              <>
                <span className="text-eco-600 select-none">·</span>
                <span className="text-xs text-eco-400">Expires {new Date(apiKey.expires_at).toLocaleDateString()}</span>
              </>
            )}
          </div>
        </div>

        {/* Revoke */}
        {!isRevoked && (
          <button
            onClick={() => setConfirmOpen(true)}
            disabled={revoking}
            aria-label={`Revoke ${apiKey.name}`}
            className="flex-shrink-0 rounded p-1.5 text-eco-500 opacity-0 transition-all hover:bg-red-500/10 hover:text-red-400 group-hover:opacity-100 disabled:pointer-events-none focus-visible:opacity-100 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)}>
        <DialogContent
          onClose={() => setConfirmOpen(false)}
          aria-labelledby="revoke-dialog-title"
          aria-describedby="revoke-dialog-desc"
        >
          <DialogTitle id="revoke-dialog-title">Revoke API Key</DialogTitle>
          <DialogDescription id="revoke-dialog-desc">
            Are you sure you want to revoke <strong>{apiKey.name}</strong>? Any applications using
            this key will immediately lose access. This cannot be undone.
          </DialogDescription>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setConfirmOpen(false)}>Cancel</Button>
            <Button
              variant="destructive"
              loading={revoking}
              onClick={() => { onRevoke(apiKey.id); setConfirmOpen(false); }}
            >
              Revoke Key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
