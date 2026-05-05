'use client';

import { useCallback, useEffect, useRef } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { Send, Zap, Leaf, DollarSign, Clock, ChevronDown, ChevronUp } from 'lucide-react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import type { ChatCompletionResponse } from '@/types/api';

// ── Zustand store ─────────────────────────────────────────────────────────────

interface Message {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

interface PlaygroundState {
  messages: Message[];
  lastMeta: ChatCompletionResponse['ecollm'] | null;
  isLoading: boolean;
  apiKey: string;
  systemPrompt: string;
  settingsOpen: boolean;
  setApiKey: (k: string) => void;
  setSystemPrompt: (p: string) => void;
  toggleSettings: () => void;
  addUserMessage: (content: string) => void;
  setAssistantReply: (content: string, meta: ChatCompletionResponse['ecollm']) => void;
  addErrorMessage: (content: string) => void;
  setLoading: (v: boolean) => void;
  clearMessages: () => void;
}

const MAX_PERSISTED_MESSAGES = 50;

const usePlayground = create<PlaygroundState>()(
  persist(
    (set) => ({
      messages: [],
      lastMeta: null,
      isLoading: false,
      apiKey: '',
      systemPrompt: 'You are a helpful assistant.',
      settingsOpen: false,
      setApiKey: (apiKey) => set({ apiKey }),
      setSystemPrompt: (systemPrompt) => set({ systemPrompt }),
      toggleSettings: () => set((s) => ({ settingsOpen: !s.settingsOpen })),
      addUserMessage: (content) =>
        set((s) => ({ messages: [...s.messages, { role: 'user', content }] })),
      setAssistantReply: (content, lastMeta) =>
        set((s) => ({
          messages: [...s.messages, { role: 'assistant', content }],
          lastMeta,
          isLoading: false,
        })),
      addErrorMessage: (content) =>
        set((s) => ({
          messages: [...s.messages, { role: 'assistant', content }],
          isLoading: false,
        })),
      setLoading: (isLoading) => set({ isLoading }),
      clearMessages: () => set({ messages: [], lastMeta: null }),
    }),
    {
      name: 'ecollm-playground',
      partialize: (s) => ({
        apiKey: s.apiKey,
        systemPrompt: s.systemPrompt,
        messages: s.messages.slice(-MAX_PERSISTED_MESSAGES),
      }),
    },
  ),
);

// ── Form schema ───────────────────────────────────────────────────────────────

const schema = z.object({
  prompt: z.string().min(1, 'Message is required'),
  maxTokens: z.coerce.number().int().min(1).max(4096).default(512),
  temperature: z.coerce.number().min(0).max(2).default(0.7),
  prefer: z.enum(['efficiency', 'speed', 'quality']).default('efficiency'),
});

type FormValues = z.infer<typeof schema>;

// ── Components ────────────────────────────────────────────────────────────────

function MetaSidebar({ meta }: { meta: ChatCompletionResponse['ecollm'] }) {
  return (
    <div className="space-y-3 text-sm">
      <h3 className="font-semibold text-gray-900 dark:text-white">Last Request</h3>

      <div className="rounded-lg bg-gray-50 dark:bg-gray-800 p-3 space-y-2">
        <p className="text-xs font-medium text-gray-500 uppercase tracking-wider">Route</p>
        <div className="space-y-1 text-gray-700 dark:text-gray-300">
          <div className="flex justify-between">
            <span>Model</span>
            <span className="font-mono text-xs">{meta.route.model_selected}</span>
          </div>
          <div className="flex justify-between">
            <span>Task</span>
            <Badge variant="default" className="text-xs">{meta.route.task_type}</Badge>
          </div>
          <div className="flex justify-between">
            <span>Score</span>
            <span>{(meta.route.routing_score * 100).toFixed(1)}%</span>
          </div>
          {meta.route.used_fallback && (
            <div className="text-amber-600 text-xs">Used fallback model</div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-lg bg-green-50 dark:bg-green-900/20 p-2.5">
          <div className="flex items-center gap-1 text-green-700 dark:text-green-400 mb-1">
            <Leaf className="h-3.5 w-3.5" />
            <span className="text-xs font-medium">CO₂e</span>
          </div>
          <p className="text-sm font-semibold text-green-800 dark:text-green-300">
            {meta.energy.co2e_grams.toFixed(4)}g
          </p>
        </div>
        <div className="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-2.5">
          <div className="flex items-center gap-1 text-blue-700 dark:text-blue-400 mb-1">
            <Zap className="h-3.5 w-3.5" />
            <span className="text-xs font-medium">Energy</span>
          </div>
          <p className="text-sm font-semibold text-blue-800 dark:text-blue-300">
            {(meta.energy.total_energy_kwh * 1000).toFixed(4)} Wh
          </p>
        </div>
        <div className="rounded-lg bg-purple-50 dark:bg-purple-900/20 p-2.5">
          <div className="flex items-center gap-1 text-purple-700 dark:text-purple-400 mb-1">
            <DollarSign className="h-3.5 w-3.5" />
            <span className="text-xs font-medium">Cost</span>
          </div>
          <p className="text-sm font-semibold text-purple-800 dark:text-purple-300">
            ${meta.cost.total_cost_usd.toFixed(6)}
          </p>
        </div>
        <div className="rounded-lg bg-orange-50 dark:bg-orange-900/20 p-2.5">
          <div className="flex items-center gap-1 text-orange-700 dark:text-orange-400 mb-1">
            <Clock className="h-3.5 w-3.5" />
            <span className="text-xs font-medium">Latency</span>
          </div>
          <p className="text-sm font-semibold text-orange-800 dark:text-orange-300">
            {meta.performance.latency_ms}ms
          </p>
        </div>
      </div>

      {meta.cost.savings_vs_gpt4_percent > 0 && (
        <div className="rounded-lg bg-emerald-50 dark:bg-emerald-900/20 px-3 py-2 text-emerald-700 dark:text-emerald-300 text-xs">
          {meta.cost.savings_vs_gpt4_percent.toFixed(1)}% cheaper than GPT-4
        </div>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function PlaygroundPage() {
  const {
    messages, lastMeta, isLoading, apiKey, systemPrompt, settingsOpen,
    setApiKey, setSystemPrompt, toggleSettings,
    addUserMessage, setAssistantReply, addErrorMessage, setLoading, clearMessages,
  } = usePlayground();

  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const { register, handleSubmit, reset, formState: { errors }, getValues } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { prompt: '', maxTokens: 512, temperature: 0.7, prefer: 'efficiency' },
  });

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages]);

  const onSubmit = useCallback(async (data: FormValues) => {
    const token = apiKey || api.getToken();
    if (!token) {
      alert('Session expired. Please refresh the page and log in again.');
      return;
    }

    addUserMessage(data.prompt);
    setLoading(true);
    reset({ ...getValues(), prompt: '' });

    const history: Message[] = systemPrompt
      ? [{ role: 'system', content: systemPrompt }, ...messages, { role: 'user', content: data.prompt }]
      : [...messages, { role: 'user', content: data.prompt }];

    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/v1/chat/completions`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
          body: JSON.stringify({
            messages: history,
            max_tokens: data.maxTokens,
            temperature: data.temperature,
            ecollm: { prefer: data.prefer, include_metadata: true },
          }),
        },
      );

      if (!res.ok) throw new Error(`API error ${res.status}`);
      const json: ChatCompletionResponse = await res.json();
      const content = json.choices[0]?.message?.content ?? '';
      setAssistantReply(content, json.ecollm);
    } catch (err) {
      addErrorMessage(`Error: ${err instanceof Error ? err.message : 'request failed'}`);
    }
  }, [apiKey, messages, systemPrompt, addUserMessage, setLoading, setAssistantReply, addErrorMessage, reset, getValues]);

  // Ctrl+Enter submits
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
        e.preventDefault();
        handleSubmit(onSubmit)();
      }
    },
    [handleSubmit, onSubmit],
  );

  const { ref: formRef, ...promptRest } = register('prompt');

  return (
    <div className="flex h-[calc(100vh-8rem)] gap-4">
      {/* ── Chat panel ───────────────────────────────────────────────────── */}
      <div className="flex flex-1 flex-col gap-3 min-w-0">
        {/* Settings bar */}
        <Card className="p-3">
          <button
            onClick={toggleSettings}
            className="flex w-full items-center justify-between text-sm font-medium text-gray-700 dark:text-gray-300"
          >
            <span>Settings</span>
            {settingsOpen ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>

          {settingsOpen && (
            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <label className="mb-1 block text-xs text-gray-500">
                  API Key <span className="text-gray-400">(optional — uses your session by default)</span>
                </label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder="ek_live_... (leave blank to use your session)"
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
              </div>
              <div className="sm:col-span-2">
                <label className="mb-1 block text-xs text-gray-500">System Prompt</label>
                <textarea
                  value={systemPrompt}
                  onChange={(e) => setSystemPrompt(e.target.value)}
                  rows={2}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800 resize-none"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-500">Max Tokens</label>
                <input
                  type="number"
                  {...register('maxTokens')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-500">Temperature</label>
                <input
                  type="number"
                  step="0.1"
                  {...register('temperature')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                />
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-500">Prefer</label>
                <select
                  {...register('prefer')}
                  className="w-full rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm dark:border-gray-600 dark:bg-gray-800"
                >
                  <option value="efficiency">Efficiency</option>
                  <option value="speed">Speed</option>
                  <option value="quality">Quality</option>
                </select>
              </div>
            </div>
          )}
        </Card>

        {/* Message thread */}
        <div
          ref={scrollRef}
          className="flex-1 overflow-y-auto rounded-xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900 p-4 space-y-4"
        >
          {messages.length === 0 && (
            <div className="flex h-full items-center justify-center text-sm text-gray-400">
              Start a conversation. Press Ctrl+Enter to send.
            </div>
          )}
          {messages.map((msg, i) => (
            <div
              key={i}
              className={cn(
                'flex',
                msg.role === 'user' ? 'justify-end' : 'justify-start',
              )}
            >
              <div
                className={cn(
                  'max-w-[80%] rounded-2xl px-4 py-2.5 text-sm',
                  msg.role === 'user'
                    ? 'bg-green-600 text-white'
                    : 'bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100',
                )}
              >
                <p className="whitespace-pre-wrap">{msg.content}</p>
              </div>
            </div>
          ))}
          {isLoading && (
            <div className="flex justify-start">
              <div className="rounded-2xl bg-gray-100 dark:bg-gray-800 px-4 py-3">
                <span className="inline-flex gap-1">
                  {[0, 1, 2].map((i) => (
                    <span
                      key={i}
                      className="h-1.5 w-1.5 rounded-full bg-gray-400 animate-bounce"
                      style={{ animationDelay: `${i * 0.15}s` }}
                    />
                  ))}
                </span>
              </div>
            </div>
          )}
        </div>

        {/* Input form */}
        <form onSubmit={handleSubmit(onSubmit)} className="flex gap-2">
          <div className="relative flex-1">
            <textarea
              {...promptRest}
              ref={(el) => {
                formRef(el);
                (textareaRef as React.MutableRefObject<HTMLTextAreaElement | null>).current = el;
              }}
              onKeyDown={handleKeyDown}
              rows={3}
              placeholder="Type a message… Ctrl+Enter to send"
              className={cn(
                'w-full resize-none rounded-xl border border-gray-300 bg-white px-4 py-3 text-sm',
                'focus:border-green-500 focus:outline-none focus:ring-1 focus:ring-green-500',
                'dark:border-gray-600 dark:bg-gray-800 dark:text-white',
                errors.prompt && 'border-red-400',
              )}
            />
          </div>
          <div className="flex flex-col gap-2">
            <Button type="submit" loading={isLoading} size="lg" className="h-full">
              <Send className="h-4 w-4" />
            </Button>
            <Button type="button" variant="ghost" size="sm" onClick={clearMessages}>
              Clear
            </Button>
          </div>
        </form>
      </div>

      {/* ── Metadata sidebar ─────────────────────────────────────────────── */}
      <div className="hidden w-64 flex-shrink-0 xl:block">
        <Card className="p-4 h-full overflow-y-auto">
          {lastMeta ? (
            <MetaSidebar meta={lastMeta} />
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-gray-400 text-center px-4">
              Routing metadata will appear here after your first request.
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}
