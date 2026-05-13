'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Menu, ChevronDown, LogOut, Settings, User } from 'lucide-react';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import { useMe, useLogout } from '@/lib/hooks/use-auth';

interface HeaderProps {
  onMenuClick: () => void;
}

export function Header({ onMenuClick }: HeaderProps) {
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const router = useRouter();
  const { data: me } = useMe();
  const logoutMutation = useLogout();

  const userName = me?.user.name ?? 'Account';

  return (
    <header className="flex h-12 items-center justify-between border-b border-eco-600 bg-eco-800 px-4">
      {/* Left: hamburger + org path */}
      <div className="flex items-center gap-3">
        <button
          onClick={onMenuClick}
          className="rounded p-1.5 text-eco-400 hover:bg-eco-700 hover:text-eco-100 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent lg:hidden"
          aria-label="Open navigation menu"
        >
          <Menu className="h-4 w-4" aria-hidden="true" />
        </button>

      </div>

      {/* Right: user menu */}
      <div className="relative">
        <button
          onClick={() => setUserMenuOpen((o) => !o)}
          aria-expanded={userMenuOpen}
          aria-haspopup="true"
          className={cn(
            'flex items-center gap-2 rounded-md px-2.5 py-1.5 transition-colors duration-150',
            'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent',
            'hover:bg-eco-700',
          )}
        >
          <div
            className="flex h-6 w-6 items-center justify-center rounded-full bg-accent/10 ring-1 ring-accent/40 text-[11px] font-bold leading-none text-accent"
            aria-hidden="true"
          >
            {userName.charAt(0).toUpperCase()}
          </div>
          <span className="hidden font-mono text-xs text-eco-200 sm:block">{userName}</span>
          <ChevronDown
            className={cn(
              'h-3 w-3 text-eco-400 transition-transform duration-150',
              userMenuOpen && 'rotate-180',
            )}
            aria-hidden="true"
          />
        </button>

        {userMenuOpen && (
          <>
            <div
              className="fixed inset-0 z-10"
              onClick={() => setUserMenuOpen(false)}
              aria-hidden="true"
            />
            <div
              role="menu"
              aria-label="User menu"
              className={cn(
                'absolute right-0 z-20 mt-1.5 w-48 animate-fade-in',
                'rounded-lg border border-eco-600 bg-eco-800 py-1 shadow-xl',
                'shadow-black/50',
              )}
            >
              {me && (
                <p className="px-4 py-2 font-mono text-[10px] text-eco-400 border-b border-eco-600">
                  {me.user.email}
                </p>
              )}
              <Link
                href="/settings"
                role="menuitem"
                onClick={() => setUserMenuOpen(false)}
                className="flex items-center gap-2.5 px-4 py-2 font-mono text-xs text-eco-200 hover:bg-eco-700 hover:text-eco-50 focus-visible:outline-none focus-visible:bg-eco-700 transition-colors"
              >
                <Settings className="h-3.5 w-3.5" aria-hidden="true" />
                Settings
              </Link>
              <button
                role="menuitem"
                onClick={() => {
                  setUserMenuOpen(false);
                  logoutMutation.mutate(undefined, {
                    onSettled: () => router.push('/login'),
                  });
                }}
                className="flex w-full items-center gap-2.5 px-4 py-2 font-mono text-xs text-red-400 hover:bg-red-500/10 focus-visible:outline-none focus-visible:bg-red-500/10 transition-colors"
              >
                <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
                Sign out
              </button>
            </div>
          </>
        )}
      </div>
    </header>
  );
}
