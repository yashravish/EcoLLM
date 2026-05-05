'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Leaf } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { api } from '@/lib/api';

const schema = z.object({
  org_name: z.string().min(2, 'Organisation name must be at least 2 characters'),
});

type FormValues = z.infer<typeof schema>;

export default function OnboardingOrganisationPage() {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema) });

  const onSubmit = async (values: FormValues) => {
    setServerError(null);
    try {
      await api.post('/auth/register/org', { org_name: values.org_name });
      router.push('/overview');
    } catch {
      setServerError('Failed to create organisation. Please try again.');
    }
  };

  return (
    <div className="flex min-h-full items-center justify-center">
      <div className="w-full max-w-sm rounded-2xl border border-white/[0.06] bg-eco-800/60 p-8 shadow-[0_0_0_1px_rgba(0,0,0,0.3),0_20px_48px_rgba(0,0,0,0.45)] backdrop-blur-sm">

        <div className="mb-5 flex items-center gap-2">
          <div className="flex h-8 w-8 items-center justify-center rounded bg-accent/10 ring-1 ring-accent/30">
            <Leaf className="h-4 w-4 text-accent" aria-hidden="true" />
          </div>
        </div>

        <h1 className="mb-0.5 text-2xl font-bold text-eco-50">Name your organisation</h1>
        <p className="mb-6 text-sm text-eco-400">
          You&apos;re almost in. Give your workspace a name to get started.
        </p>

        <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
          <Input
            label="Organisation name"
            type="text"
            autoComplete="organization"
            placeholder="Acme Inc."
            error={errors.org_name?.message}
            {...register('org_name')}
          />

          {serverError && (
            <div
              role="alert"
              className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-400"
            >
              {serverError}
            </div>
          )}

          <Button type="submit" className="w-full" size="lg" disabled={isSubmitting} loading={isSubmitting}>
            Continue
          </Button>
        </form>
      </div>
    </div>
  );
}
