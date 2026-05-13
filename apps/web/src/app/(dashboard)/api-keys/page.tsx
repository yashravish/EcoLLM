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

            <div className="mt-4 rounded-md border border-amber-500/30 bg-amber-500/10 p-3">
              <p className="mb-2 text-xs font-medium text-amber-400">
                Save this key now — it will not be shown again.
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 break-all rounded border border-eco-700 bg-eco-900 px-2 py-1.5 font-mono text-xs text-eco-100">
                  {createdKey.key}
                </code>
                <Button
                  variant="secondary"
                  size="icon"
                  onClick={handleCopy}
                  aria-label="Copy API key to clipboard"
                >
                  {copied ? (
                    <Check className="h-4 w-4 text-accent" aria-hidden="true" />
                  ) : (
                    <Copy className="h-4 w-4" aria-hidden="true" />
                  )}
                </Button>
              </div>
              {copied && (
                <p role="status" className="mt-1 font-mono text-xs text-accent">
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
                <legend className="mb-2 text-xs font-medium text-eco-400">
                  Permissions
                </legend>
                <div className="space-y-2">
                  {SCOPES.map((scope) => (
                    <label key={scope} className="flex cursor-pointer items-center gap-2.5">
                      <input
                        type="checkbox"
                        value={scope}
                        className="h-3.5 w-3.5 rounded border-eco-600 bg-eco-800 accent-[#00E87A] focus:ring-2 focus:ring-accent focus:ring-offset-eco-800"
                        {...register('scopes')}
                      />
                      <span className="text-xs capitalize text-eco-300">{scope}</span>
                    </label>
                  ))}
                </div>
                {errors.scopes && (
                  <p role="alert" className="mt-1 text-xs text-red-400">
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
        <div>
          <h1 className="text-base font-semibold text-eco-50">API Keys</h1>
          <p className="mt-0.5 text-xs text-eco-400">Authenticate requests to the EcoLLM API</p>
        </div>
        <button
          onClick={() => setDialogOpen(true)}
          className="flex items-center gap-1.5 rounded-md border border-eco-500 bg-eco-700 px-3 py-1.5 text-xs font-medium text-eco-100 transition-colors hover:border-eco-400 hover:bg-eco-600 hover:text-eco-50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
        >
          <Plus className="h-3.5 w-3.5" aria-hidden="true" />
          New key
        </button>
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
