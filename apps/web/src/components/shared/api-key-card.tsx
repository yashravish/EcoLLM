'use client';

import { useState } from 'react';
import { Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { formatRelativeTime } from '@/lib/utils';
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
      <div className="flex items-center justify-between rounded-lg border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-900">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-gray-900 dark:text-white">{apiKey.name}</span>
            {isRevoked ? (
              <Badge variant="danger">Revoked</Badge>
            ) : (
              <Badge variant="success">Active</Badge>
            )}
          </div>
          <p className="mt-0.5 font-mono text-xs text-gray-500 dark:text-gray-400">
            {apiKey.key_prefix}••••••••
          </p>
          <div className="mt-1.5 flex flex-wrap gap-1">
            {apiKey.scopes.map((scope) => (
              <Badge key={scope} variant="outline" className="text-xs">
                {scope}
              </Badge>
            ))}
          </div>
          <div className="mt-1.5 flex gap-4 text-xs text-gray-400 dark:text-gray-500">
            <span>Created {formatRelativeTime(apiKey.created_at)}</span>
            {apiKey.last_used_at ? (
              <span>Last used {formatRelativeTime(apiKey.last_used_at)}</span>
            ) : (
              <span>Never used</span>
            )}
            {apiKey.expires_at && (
              <span>Expires {new Date(apiKey.expires_at).toLocaleDateString()}</span>
            )}
          </div>
        </div>

        <Button
          variant="ghost"
          size="icon"
          onClick={() => setConfirmOpen(true)}
          disabled={isRevoked || revoking}
          aria-label={`Revoke API key ${apiKey.name}`}
          className="ml-4 flex-shrink-0 text-red-500 hover:bg-red-50 hover:text-red-700 disabled:opacity-40 dark:hover:bg-red-900/20"
        >
          <Trash2 className="h-4 w-4" aria-hidden="true" />
        </Button>
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
            <Button variant="secondary" onClick={() => setConfirmOpen(false)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              loading={revoking}
              onClick={() => {
                onRevoke(apiKey.id);
                setConfirmOpen(false);
              }}
            >
              Revoke Key
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
