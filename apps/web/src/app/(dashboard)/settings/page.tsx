'use client';

import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Save, UserPlus, Trash2, Shield, User, Eye } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { api } from '@/lib/api';
import { useMe } from '@/lib/hooks/use-auth';

// ── Types ─────────────────────────────────────────────────────────────────────

interface OrgSettings {
  id: string;
  name: string;
  slug: string;
  plan: string;
  quality_threshold: number;
  energy_budget_kwh: number | null;
}

interface Member {
  id: string;
  email: string;
  name: string;
  role: string;
  created_at: string;
}

// ── Schemas ───────────────────────────────────────────────────────────────────

const orgSchema = z.object({
  name: z.string().min(2, 'Name must be at least 2 characters'),
  quality_threshold: z.coerce.number().min(0).max(1),
  energy_budget_kwh: z.preprocess(
    (v) => (v === '' || v === null ? null : Number(v)),
    z.number().positive().nullable(),
  ),
});

const inviteSchema = z.object({
  email: z.string().email('Valid email required'),
  name: z.string().min(1, 'Name required'),
  password: z.string().min(8, 'Minimum 8 characters'),
  role: z.enum(['admin', 'member', 'viewer']),
});

type OrgFormValues = z.infer<typeof orgSchema>;
type InviteFormValues = z.infer<typeof inviteSchema>;

// ── Hooks ─────────────────────────────────────────────────────────────────────

function useOrgSettings(orgId: string | undefined) {
  return useQuery<OrgSettings>({
    queryKey: ['org', orgId],
    queryFn: () => api.get(`/organizations/${orgId}`),
    enabled: Boolean(orgId),
    staleTime: 60 * 1000,
  });
}

function useMembers(orgId: string | undefined) {
  return useQuery<{ org_id: string; members: Member[] }>({
    queryKey: ['members', orgId],
    queryFn: () => api.get(`/organizations/${orgId}/members`),
    enabled: Boolean(orgId),
    staleTime: 60 * 1000,
  });
}

function useUpdateOrg(orgId: string) {
  const qc = useQueryClient();
  return useMutation<OrgSettings, Error, OrgFormValues>({
    mutationFn: (data) => api.patch(`/organizations/${orgId}`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['org', orgId] }),
  });
}

function useInviteMember(orgId: string) {
  const qc = useQueryClient();
  return useMutation<Member, Error, InviteFormValues>({
    mutationFn: (data) => api.post(`/organizations/${orgId}/members`, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['members', orgId] }),
  });
}

function useRemoveMember(orgId: string) {
  const qc = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (userId) => api.delete(`/organizations/${orgId}/members/${userId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['members', orgId] }),
  });
}

// ── Role badge ────────────────────────────────────────────────────────────────

const roleIcons: Record<string, React.ReactNode> = {
  admin: <Shield className="h-3 w-3" />,
  member: <User className="h-3 w-3" />,
  viewer: <Eye className="h-3 w-3" />,
};

function RoleBadge({ role }: { role: string }) {
  return (
    <Badge variant="default" className="gap-1 text-xs">
      {roleIcons[role] ?? null}
      {role}
    </Badge>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function SettingsPage() {
  const [showInviteForm, setShowInviteForm] = useState(false);
  const { data: me } = useMe();
  const orgId = (me as any)?.org_id as string | undefined;

  const { data: org, isLoading: orgLoading } = useOrgSettings(orgId);
  const { data: membersData, isLoading: membersLoading } = useMembers(orgId);
  const updateOrg = useUpdateOrg(orgId ?? '');
  const inviteMember = useInviteMember(orgId ?? '');
  const removeMember = useRemoveMember(orgId ?? '');

  const orgForm = useForm<OrgFormValues>({
    resolver: zodResolver(orgSchema),
    values: org
      ? {
          name: org.name,
          quality_threshold: org.quality_threshold,
          energy_budget_kwh: org.energy_budget_kwh,
        }
      : undefined,
  });

  const inviteForm = useForm<InviteFormValues>({
    resolver: zodResolver(inviteSchema),
    defaultValues: { email: '', name: '', password: '', role: 'member' },
  });

  const onSaveOrg = orgForm.handleSubmit((data) => {
    updateOrg.mutate(data);
  });

  const onInvite = inviteForm.handleSubmit((data) => {
    inviteMember.mutate(data, {
      onSuccess: () => {
        inviteForm.reset();
        setShowInviteForm(false);
      },
    });
  });

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-white">Settings</h1>

      {/* ── Org Settings ── */}
      <Card className="p-5">
        <h2 className="mb-4 text-sm font-semibold text-gray-900 dark:text-white">Organization</h2>

        {orgLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-9 w-full" />
            <Skeleton className="h-9 w-1/2" />
          </div>
        ) : (
          <form onSubmit={onSaveOrg} className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-gray-500">Organization Name</label>
              <input
                {...orgForm.register('name')}
                className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800"
              />
              {orgForm.formState.errors.name && (
                <p className="mt-1 text-xs text-red-500">{orgForm.formState.errors.name.message}</p>
              )}
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">
                  Quality Threshold (0–1)
                </label>
                <input
                  type="number"
                  step="0.01"
                  {...orgForm.register('quality_threshold')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-gray-500">
                  Energy Budget (kWh/day, optional)
                </label>
                <input
                  type="number"
                  step="0.001"
                  placeholder="No limit"
                  {...orgForm.register('energy_budget_kwh')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
              </div>
            </div>

            {org && (
              <div className="flex items-center gap-3 text-xs text-gray-400">
                <span>Plan: <span className="capitalize font-medium text-gray-600">{org.plan}</span></span>
                <span>Slug: <span className="font-mono text-gray-600">{org.slug}</span></span>
              </div>
            )}

            <div className="flex justify-end">
              <Button
                type="submit"
                loading={updateOrg.isPending}
                size="sm"
              >
                <Save className="h-3.5 w-3.5" />
                Save Changes
              </Button>
            </div>

            {updateOrg.isSuccess && (
              <p className="text-xs text-green-600 text-right">Saved successfully.</p>
            )}
            {updateOrg.isError && (
              <p className="text-xs text-red-500 text-right">Failed to save. Please try again.</p>
            )}
          </form>
        )}
      </Card>

      {/* ── Team Members ── */}
      <Card className="p-5">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-gray-900 dark:text-white">Team Members</h2>
          <Button variant="outline" size="sm" onClick={() => setShowInviteForm((v) => !v)}>
            <UserPlus className="h-3.5 w-3.5" />
            Invite Member
          </Button>
        </div>

        {showInviteForm && (
          <form onSubmit={onInvite} className="mb-5 rounded-lg border border-gray-100 p-4 dark:border-gray-700 space-y-3">
            <h3 className="text-xs font-medium text-gray-700 dark:text-gray-300">New Member</h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-xs text-gray-500">Email</label>
                <input
                  type="email"
                  {...inviteForm.register('email')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
                {inviteForm.formState.errors.email && (
                  <p className="mt-0.5 text-xs text-red-500">{inviteForm.formState.errors.email.message}</p>
                )}
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-500">Name</label>
                <input
                  {...inviteForm.register('name')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-500">Temporary Password</label>
                <input
                  type="password"
                  {...inviteForm.register('password')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
                {inviteForm.formState.errors.password && (
                  <p className="mt-0.5 text-xs text-red-500">{inviteForm.formState.errors.password.message}</p>
                )}
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-500">Role</label>
                <select
                  {...inviteForm.register('role')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                >
                  <option value="member">Member</option>
                  <option value="admin">Admin</option>
                  <option value="viewer">Viewer</option>
                </select>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" size="sm" onClick={() => setShowInviteForm(false)}>
                Cancel
              </Button>
              <Button type="submit" size="sm" loading={inviteMember.isPending}>
                <UserPlus className="h-3.5 w-3.5" />
                Invite
              </Button>
            </div>
            {inviteMember.isError && (
              <p className="text-xs text-red-500">Failed to invite member.</p>
            )}
          </form>
        )}

        {membersLoading ? (
          <div className="space-y-2">
            {[0, 1, 2].map((i) => <Skeleton key={i} className="h-12 w-full" />)}
          </div>
        ) : (
          <ul className="divide-y divide-gray-100 dark:divide-gray-700">
            {(membersData?.members ?? []).map((member) => (
              <li key={member.id} className="flex items-center gap-3 py-3">
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-700 text-xs font-semibold text-gray-600 dark:text-gray-300 flex-shrink-0">
                  {(member.name || member.email)[0].toUpperCase()}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-gray-900 dark:text-white truncate">
                    {member.name || member.email}
                  </p>
                  <p className="text-xs text-gray-400 truncate">{member.email}</p>
                </div>
                <RoleBadge role={member.role} />
                {member.id !== orgId && (
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={() => removeMember.mutate(member.id)}
                    aria-label={`Remove ${member.name}`}
                    className="text-gray-400 hover:text-red-500"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                )}
              </li>
            ))}
            {membersData?.members?.length === 0 && (
              <li className="py-6 text-center text-sm text-gray-400">No members yet.</li>
            )}
          </ul>
        )}
      </Card>

      {/* ── Danger Zone ── */}
      <Card className="p-5 border-red-100 dark:border-red-900/30">
        <h2 className="mb-2 text-sm font-semibold text-red-600">Danger Zone</h2>
        <p className="text-xs text-gray-500 mb-3">
          Deleting your organization is permanent and cannot be undone.
          All data, API keys, and billing history will be removed.
        </p>
        <Button variant="destructive" size="sm" disabled>
          Delete Organization
        </Button>
        <p className="mt-1 text-xs text-gray-400">Contact support to delete your account.</p>
      </Card>
    </div>
  );
}
