'use client';

import { useState } from 'react';
import { Menu, ChevronDown, LogOut, Settings, User } from 'lucide-react';
import Link from 'next/link';
import { cn } from '@/lib/utils';
import { useMe, useLogout } from '@/lib/hooks/use-auth';

interface HeaderProps {
  onMenuClick: () => void;
}

export function Header({ onMenuClick }: HeaderProps) {
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const { data: me } = useMe();
  const logoutMutation = useLogout();

  const orgName = me?.org.name ?? 'My Organization';
  const userName = me?.user.name ?? 'Account';

  return (
    <header className="flex h-14 items-center justify-between border-b border-gray-200 bg-white px-4 dark:border-gray-700 dark:bg-gray-900">
      {/* Left: hamburger + org name */}
      <div className="flex items-center gap-3">
        <button
          onClick={onMenuClick}
          className="rounded p-1.5 text-gray-500 hover:bg-gray-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500 lg:hidden dark:text-gray-400 dark:hover:bg-gray-800"
          aria-label="Open navigation menu"
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>
        <span className="hidden text-sm font-medium text-gray-900 dark:text-white sm:block">
          {orgName}
        </span>
      </div>

      {/* Right: user dropdown */}
      <div className="relative">
        <button
          onClick={() => setUserMenuOpen((o) => !o)}
          aria-expanded={userMenuOpen}
          aria-haspopup="true"
          className={cn(
            'flex items-center gap-2 rounded-md px-3 py-1.5 text-sm transition-colors',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-green-500',
            'hover:bg-gray-50 dark:hover:bg-gray-800',
          )}
        >
          <div
            className="flex h-7 w-7 items-center justify-center rounded-full bg-green-100 text-xs font-semibold text-green-700 dark:bg-green-900/40 dark:text-green-400"
            aria-hidden="true"
          >
            {userName.charAt(0).toUpperCase()}
          </div>
          <span className="hidden text-gray-700 dark:text-gray-300 sm:block">{userName}</span>
          <ChevronDown
            className={cn(
              'h-4 w-4 text-gray-400 transition-transform',
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
              className="absolute right-0 z-20 mt-1 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900"
            >
              <Link
                href="/settings"
                role="menuitem"
                onClick={() => setUserMenuOpen(false)}
                className="flex items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50 focus-visible:outline-none focus-visible:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
              >
                <Settings className="h-4 w-4" aria-hidden="true" />
                Settings
              </Link>
              {me && (
                <p className="px-4 py-1.5 text-xs text-gray-400 dark:text-gray-500">
                  {me.user.email}
                </p>
              )}
              <button
                role="menuitem"
                onClick={() => {
                  setUserMenuOpen(false);
                  logoutMutation.mutate();
                }}
                className="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 focus-visible:outline-none focus-visible:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
              >
                <LogOut className="h-4 w-4" aria-hidden="true" />
                Logout
              </button>
            </div>
          </>
        )}
      </div>
    </header>
  );
}
