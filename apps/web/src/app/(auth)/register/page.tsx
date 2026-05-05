'use client';

import { Suspense, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { CheckCircle2, XCircle } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useRegister } from '@/lib/hooks/use-auth';
import { ApiError } from '@/lib/api';
import {
  MobileLogo,
  OAuthSection,
  OAuthErrorBanner,
  ServerErrorBanner,
} from '@/components/auth/oauth-section';

const schema = z.object({
  name:             z.string().min(2, 'Name must be at least 2 characters'),
  email:            z.string().email('Invalid email'),
  password:         z.string().min(8, 'Password must be at least 8 characters'),
  confirm_password: z.string(),
  org_name:         z.string().min(2, 'Organisation name must be at least 2 characters'),
}).refine((data) => data.password === data.confirm_password, {
  message: 'Passwords do not match',
  path: ['confirm_password'],
});

type FormValues = z.infer<typeof schema>;

function RegisterInner() {
  const router        = useRouter();
  const searchParams  = useSearchParams();
  const register      = useRegister();
  const [serverError, setServerError]   = useState<string | null>(null);
  const [oauthLoading, setOauthLoading] = useState<'github' | 'google' | null>(null);

  const oauthError = searchParams.get('error');

  const {
    register: field,
    handleSubmit,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), reValidateMode: 'onBlur' });

  const watchedPassword        = watch('password', '');
  const watchedConfirmPassword = watch('confirm_password', '');

  const onSubmit = async ({ confirm_password: _, ...values }: FormValues) => {
    setServerError(null);
    try {
      await register.mutateAsync(values);
      router.push('/overview');
    } catch (err) {
      if (err instanceof ApiError) {
        setServerError(err.message);
      } else {
        setServerError('Registration failed. Please try again.');
      }
    }
  };

  return (
    <div className="auth-card-glow rounded-2xl bg-eco-800 px-6 py-4">

      {/* Mobile-only logo */}
      <MobileLogo className="mb-3" />

      <h1 className="mb-0.5 text-2xl font-bold text-eco-50 text-center">Create account</h1>
      <p className="mb-3 text-sm text-eco-400 text-center">Set up your workspace in seconds</p>

      {/* OAuth error banner */}
      {oauthError && <OAuthErrorBanner errorKey={oauthError} className="mb-3" />}

      {/* Social auth */}
      <OAuthSection
        oauthLoading={oauthLoading}
        onGitHub={() => {
          setOauthLoading('github');
          window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/auth/github/begin`;
        }}
        onGoogle={() => {
          setOauthLoading('google');
          window.location.href = `${process.env.NEXT_PUBLIC_API_URL}/auth/google/begin`;
        }}
        className="space-y-1.5 mb-2"
      />

      {/* Divider */}
      <div className="relative mb-2 flex items-center gap-3">
        <div className="h-px flex-1 bg-eco-700/50" />
        <span className="text-xs text-eco-600">or</span>
        <div className="h-px flex-1 bg-eco-700/50" />
      </div>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-2">
        {/* Name + Org on one row */}
        <div className="grid grid-cols-2 gap-3">
          <Input
            label="Full name"
            type="text"
            autoComplete="name"
            placeholder="Jane Smith"
            error={errors.name?.message}
            {...field('name')}
          />
          <Input
            label="Organization"
            type="text"
            autoComplete="organization"
            placeholder="Acme Inc."
            error={errors.org_name?.message}
            {...field('org_name')}
          />
        </div>

        <Input
          label="Email"
          type="email"
          autoComplete="email"
          placeholder="you@company.com"
          error={errors.email?.message}
          {...field('email')}
        />
        <Input
          label="Password"
          type="password"
          autoComplete="new-password"
          placeholder="••••••••"
          error={errors.password?.message}
          {...field('password')}
        />
        <Input
          label="Confirm password"
          type="password"
          autoComplete="new-password"
          placeholder="••••••••"
          error={watchedConfirmPassword ? undefined : errors.confirm_password?.message}
          {...field('confirm_password')}
        />
        {watchedConfirmPassword.length > 0 && (
          watchedPassword === watchedConfirmPassword ? (
            <div className="flex items-center gap-1.5 text-xs text-accent">
              <CheckCircle2 className="h-3.5 w-3.5 shrink-0" />
              Passwords match
            </div>
          ) : (
            <div className="flex items-center gap-1.5 text-xs text-red-400">
              <XCircle className="h-3.5 w-3.5 shrink-0" />
              Passwords don&apos;t match
            </div>
          )
        )}

        {serverError && <ServerErrorBanner message={serverError} />}

        <Button type="submit" className="w-full" size="lg" disabled={isSubmitting} loading={isSubmitting}>
          Create account
        </Button>
      </form>

      <p className="mt-3 text-center text-xs text-eco-300">
        Already have an account?{' '}
        <Link
          href="/login"
          className="text-accent hover:text-accent-bright underline underline-offset-2 transition-colors"
        >
          Sign in
        </Link>
      </p>
    </div>
  );
}

export default function RegisterPage() {
  return (
    <Suspense fallback={null}>
      <RegisterInner />
    </Suspense>
  );
}
