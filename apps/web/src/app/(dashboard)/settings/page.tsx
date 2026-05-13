'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { useMe } from '@/lib/hooks/use-auth';
import { Dialog, DialogContent, DialogTitle, DialogDescription, DialogFooter } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

const profileSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
});

type ProfileFormValues = z.infer<typeof profileSchema>;

function useUpdateProfile() {
  const qc = useQueryClient();
  return useMutation<void, Error, ProfileFormValues>({
    mutationFn: (data) => api.patch('/auth/me', data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['me'] }),
  });
}

function useDeleteAccount() {
  return useMutation<void, Error, void>({
    mutationFn: () => api.delete('/auth/me'),
  });
}

export default function SettingsPage() {
  const router = useRouter();
  const { data: me } = useMe();
  const updateProfile = useUpdateProfile();
  const deleteAccount = useDeleteAccount();
  const [deleteOpen, setDeleteOpen] = useState(false);

  const profileForm = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    values: me ? { name: me.user.name } : undefined,
  });

  const onSaveProfile = profileForm.handleSubmit((data) => {
    updateProfile.mutate(data);
  });

  const handleDelete = () => {
    deleteAccount.mutate(undefined, {
      onSuccess: () => {
        api.setToken('');
        router.replace('/login');
      },
    });
  };

  return (
    <div className="max-w-xl mx-auto space-y-6">
      <h1 className="text-base font-semibold text-eco-50">Settings</h1>

      <div className="rounded-lg border border-eco-600 bg-eco-800 divide-y divide-eco-700">
        <div className="px-5 py-4">
          <p className="text-xs font-medium text-eco-400 mb-0.5">Email</p>
          <p className="text-sm text-eco-100">{me?.user.email ?? '—'}</p>
        </div>

        <form onSubmit={onSaveProfile} className="px-5 py-4 space-y-3">
          <div>
            <label className="block text-xs font-medium text-eco-400 mb-1">Display Name</label>
            <input
              {...profileForm.register('name')}
              className="w-full rounded-md border border-eco-600 bg-eco-900 px-3 py-2 text-sm text-eco-100 placeholder:text-eco-600 focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent/20"
            />
            {profileForm.formState.errors.name && (
              <p className="mt-1 text-xs text-red-400">{profileForm.formState.errors.name.message}</p>
            )}
          </div>

          <div className="flex items-center justify-between pt-1">
            <div>
              {updateProfile.isSuccess && <p className="text-xs text-accent">Saved.</p>}
              {updateProfile.isError && <p className="text-xs text-red-400">Failed to save.</p>}
            </div>
            <button
              type="submit"
              disabled={updateProfile.isPending}
              className="rounded-md border border-eco-500 bg-eco-700 px-3 py-1.5 text-xs font-medium text-eco-100 transition-colors hover:border-eco-400 hover:bg-eco-600 disabled:opacity-50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent"
            >
              {updateProfile.isPending ? 'Saving…' : 'Save changes'}
            </button>
          </div>
        </form>
      </div>

      <div className="rounded-lg border border-red-900/50 bg-eco-800 px-5 py-4">
        <p className="text-xs font-medium text-eco-300 mb-0.5">Delete account</p>
        <p className="text-xs text-eco-500 mb-3">Permanent and cannot be undone.</p>
        <button
          onClick={() => setDeleteOpen(true)}
          className="rounded-md border border-red-800/60 bg-red-950/40 px-3 py-1.5 text-xs font-medium text-red-400 transition-colors hover:border-red-700 hover:bg-red-950/70 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-red-500"
        >
          Delete account
        </button>
      </div>

      <Dialog open={deleteOpen} onClose={() => setDeleteOpen(false)}>
        <DialogContent onClose={() => setDeleteOpen(false)} aria-labelledby="delete-title" aria-describedby="delete-desc">
          <DialogTitle id="delete-title">Delete account</DialogTitle>
          <DialogDescription id="delete-desc">
            This will permanently delete your account, API keys, and all associated data. This cannot be undone.
          </DialogDescription>
          {deleteAccount.isError && (
            <p className="mt-2 text-xs text-red-400">Something went wrong. Please try again.</p>
          )}
          <DialogFooter>
            <Button variant="secondary" onClick={() => setDeleteOpen(false)} disabled={deleteAccount.isPending}>
              Cancel
            </Button>
            <Button variant="destructive" loading={deleteAccount.isPending} onClick={handleDelete}>
              Delete my account
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
