'use client';

import { Suspense, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';
import { useLogin } from '@/lib/hooks/use-auth';
import { ApiError } from '@/lib/api';
import {
  MobileLogo,
  OAuthSection,
  OAuthErrorBanner,
  ServerErrorBanner,
} from '@/components/auth/oauth-section';

const schema = z.object({
  email:    z.string().min(1, 'Email is required'),
  password: z.string().min(1, 'Password is required'),
});

type FormValues = z.infer<typeof schema>;

function LoginInner() {
  const router        = useRouter();
  const searchParams  = useSearchParams();
  const login         = useLogin();
  const [serverError, setServerError]   = useState<string | null>(null);
  const [oauthLoading, setOauthLoading] = useState<'github' | 'google' | null>(null);

  const oauthError = searchParams.get('error');

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({ resolver: zodResolver(schema), mode: 'onTouched' });

  const onSubmit = async (values: FormValues) => {
    setServerError(null);
    try {
      await login.mutateAsync(values);
      router.push('/playground');
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setServerError('Incorrect email or password.');
      } else if (err instanceof ApiError && err.status === 429) {
        setServerError('Too many attempts. Please wait a moment and try again.');
      } else if (err instanceof TypeError && err.message === 'Failed to fetch') {
        setServerError('Unable to reach the server. Please check your connection.');
      } else {
        setServerError('Something went wrong. Please try again.');
      }
    }
  };

  return (
    <div>

      {/* Mobile-only logo */}
      <MobileLogo className="mb-5" />

      <h1 className="mb-0.5 text-2xl font-bold text-eco-50 text-center">Welcome back</h1>
      <p className="mb-6 text-sm text-eco-300 text-center">Sign in to your workspace</p>

      {/* OAuth error banner */}
      {oauthError && <OAuthErrorBanner errorKey={oauthError} className="mb-4" />}

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
        className="space-y-2.5 mb-5"
      />

      {/* Divider */}
      <div className="relative mb-5 flex items-center gap-3">
        <div className="h-px flex-1 bg-eco-700/60" />
        <span className="text-xs text-eco-500">or</span>
        <div className="h-px flex-1 bg-eco-700/60" />
      </div>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-4">
        <Input
          label="Email"
          type="email"
          autoComplete="email"
          placeholder="you@company.com"
          error={errors.email?.message}
          {...register('email')}
        />
        <Input
          label="Password"
          type="password"
          autoComplete="current-password"
          placeholder="••••••••"
          error={errors.password?.message}
          {...register('password')}
        />

        {serverError && <ServerErrorBanner message={serverError} />}

        <Button type="submit" className="w-full" size="lg" disabled={isSubmitting} loading={isSubmitting}>
          Sign in
        </Button>
      </form>

      <p className="mt-5 text-center text-xs text-eco-300">
        No account?{' '}
        <Link
          href="/register"
          className="text-accent hover:text-accent-bright underline underline-offset-2 transition-colors"
        >
          Create one
        </Link>
      </p>

    </div>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginInner />
    </Suspense>
  );
}
