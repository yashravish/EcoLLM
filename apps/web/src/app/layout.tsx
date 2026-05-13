import type { Metadata } from 'next';
import { Outfit } from 'next/font/google';
import './globals.css';
import { QueryProvider } from '@/components/providers/query-provider';

const outfit = Outfit({
  subsets: ['latin'],
  variable: '--font-outfit',
  display: 'swap',
});

export const metadata: Metadata = {
  title: 'EcoLLM',
  description: 'Run LLM inference with 73% lower carbon footprint than GPT-4.',
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={outfit.variable}>
      <body>
        <QueryProvider>{children}</QueryProvider>
      </body>
    </html>
  );
}
