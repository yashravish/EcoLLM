'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function OnboardingOrganisationPage() {
  const router = useRouter();
  useEffect(() => { router.replace('/playground'); }, [router]);
  return null;
}
