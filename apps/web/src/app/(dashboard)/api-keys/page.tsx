'use client';

import { useState, useId } from 'react';
import { Plus, Copy, Check } from 'lucide-react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { ApiKeyCard } from '@/components/shared/api-key-card';
import { EmptyState } from '@/components/shared/empty-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select } from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { useApiKeys, useCreateApiKey, useRevokeApiKey } from '@/lib/hooks/use-api-keys';
import type { CreateApiKeyResponse } from '@/types';

// ===== Form schema =====

const createKeySchema = z.object({
  name: z.string().min(1, 'Name is required').max(100, 'Name must be 100 characters or fewer'),
  scopes: z.array(z.string()).min(1, 'Select at least one scope'),
  expires_in_days: z.number().optional(),
});

type CreateKeyForm = z.infer<typeof createKeySchema>;

const EXPIRY_OPTIONS = [
  { value: '', label: 'Never' },
  { value: '30', label: '30 days' },
  { value: '60', label: '60 days' },
  { value: '90', label: '90 days' },
];

const SCOPES = ['inference', 'admin', 'billing'] as const;

// ===== Create Key Dialog =====

interface CreateKeyDialogProps {
  open: boolean;
  onClose: () => void;
}

function CreateKeyDialog({ open, onClose }: CreateKeyDialogProps) {
  const titleId = useId();
  const descId = useId();
  const [createdKey, setCreatedKey] = useState<CreateApiKeyResponse | null>(null);
  const [copied, setCopied] = useState(false);

  const createMutation = useCreateApiKey();

  const {
    register,
    handleSubmit,
    formState: { errors },
    reset,
  } = useForm<CreateKeyForm>({
    resolver: zodResolver(createKeySchema),
    defaultValues: { scopes: ['inference'] },
  });

  function handleClose() {
    reset();
    setCreatedKey(null);
    setCopied(false);
    onClose();
  }

  async function onSubmit(data: CreateKeyForm) {
    const result = await createMutation.mutateAsync({
      name: data.name,
      scopes: data.scopes,
      expires_in_days: data.expires_in_days,
    });
    setCreatedKey(result);
  }

  async function handleCopy() {
    if (!createdKey) return;
    await navigator.clipboard.writeText(createdKey.key);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <Dialog open={open} onClose={handleClose}>
      <DialogContent
        onClose={handleClose}
        aria-labelledby={titleId}
        aria-describedby={descId}
      >
        {createdKey ? (
          /* Success state — show key one time */
          <>
            <DialogTitle id={titleId}>API Key Created</DialogTitle>
            <DialogDescription id={descId}>
              Copy your key now. It will not be shown again.
            </DialogDescription>

            <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950">
              <p className="mb-2 text-xs font-medium text-amber-800 dark:text-amber-300">
                ⚠ Save this key now. It will not be shown again.
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 break-all rounded bg-white px-2 py-1.5 font-mono text-xs text-gray-900 dark:bg-gray-800 dark:text-white">
                  {createdKey.key}
                </code>
                <Button
                  variant="secondary"
                  size="icon"
                  onClick={handleCopy}
                  aria-label="Copy API key to clipboard"
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-green-600" aria-hidden="true" />
                  ) : (
                    <Copy className="h-4 w-4" aria-hidden="true" />
                  )}
                </Button>
              </div>
              {copied && (
                <p role="status" className="mt-1 text-xs text-green-700 dark:text-green-400">
                  Copied!
                </p>
              )}
            </div>

            <DialogFooter>
              <Button onClick={handleClose}>Done</Button>
            </DialogFooter>
          </>
        ) : (
          /* Creation form */
          <>
            <DialogTitle id={titleId}>Create New API Key</DialogTitle>
            <DialogDescription id={descId}>
              Choose a name and permissions for your new API key.
            </DialogDescription>

            <form onSubmit={handleSubmit(onSubmit)} noValidate className="mt-4 space-y-4">
              <Input
                label="Key name"
                placeholder="e.g. Production"
                error={errors.name?.message}
                {...register('name')}
              />

              {/* Scopes */}
              <fieldset>
                <legend className="mb-1.5 text-sm font-medium text-gray-700 dark:text-gray-300">
                  Permissions
                </legend>
                <div className="space-y-2">
                  {SCOPES.map((scope) => (
                    <label key={scope} className="flex cursor-pointer items-center gap-2.5">
                      <input
                        type="checkbox"
                        value={scope}
                        className="h-4 w-4 rounded border-gray-300 text-green-600 focus:ring-2 focus:ring-green-500 focus:ring-offset-1"
                        {...register('scopes')}
                      />
                      <span className="text-sm capitalize text-gray-700 dark:text-gray-300">
                        {scope}
                      </span>
                    </label>
                  ))}
                </div>
                {errors.scopes && (
                  <p role="alert" className="mt-1 text-xs text-red-600 dark:text-red-400">
                    {errors.scopes.message}
                  </p>
                )}
              </fieldset>

              {/* Expiry */}
              <Select
                label="Expiry"
                options={EXPIRY_OPTIONS}
                {...register('expires_in_days', {
                  setValueAs: (v) => (v === '' ? undefined : Number(v)),
                })}
              />

              <DialogFooter>
                <Button type="button" variant="secondary" onClick={handleClose}>
                  Cancel
                </Button>
                <Button type="submit" loading={createMutation.isPending}>
                  Create Key
                </Button>
              </DialogFooter>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

// ===== Page =====

export default function ApiKeysPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { data: keys, isLoading } = useApiKeys();
  const revokeMutation = useRevokeApiKey();

  const isEmpty = !isLoading && (keys?.length ?? 0) === 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-white">API Keys</h1>
        <Button onClick={() => setDialogOpen(true)}>
          <Plus className="h-4 w-4" aria-hidden="true" />
          Create New Key
        </Button>
      </div>

      {isLoading && <LoadingSkeleton variant="table" lines={3} />}

      {isEmpty && (
        <EmptyState
          title="No API keys yet"
          description="Create your first key to start using the API."
          action={{ label: 'Create Key', onClick: () => setDialogOpen(true) }}
        />
      )}

      {!isLoading && !isEmpty && (
        <div
          role="list"
          aria-label="API keys"
          className="space-y-3"
        >
          {keys?.map((key) => (
            <div key={key.id} role="listitem">
              <ApiKeyCard
                apiKey={key}
                onRevoke={(id) => revokeMutation.mutate(id)}
                revoking={revokeMutation.isPending && revokeMutation.variables === key.id}
              />
            </div>
          ))}
        </div>
      )}

      <CreateKeyDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </div>
  );
}
