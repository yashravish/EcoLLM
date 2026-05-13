'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useQuery } from '@tanstack/react-query';
import {
  Send, Square, Leaf, ChevronDown, ChevronUp,
  PanelRightOpen, PanelRightClose,
} from 'lucide-react';

// ── Custom dropdown (replaces native <select> to avoid browser flash + blue highlight) ──

interface SelectOption { value: string; label: string }

function CustomSelect({
  value, onChange, options, placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  options: SelectOption[];
  placeholder?: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const label = options.find((o) => o.value === value)?.label ?? placeholder ?? value;

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1.5 rounded-md border border-eco-700 bg-eco-800 px-2.5 py-1 font-mono text-xs text-eco-200 hover:border-eco-600 hover:text-eco-100 transition-colors"
      >
        <span>{label}</span>
        <ChevronDown className="h-3 w-3 text-eco-500" />
      </button>
      {open && (
        <div className="absolute left-0 top-full z-50 mt-1 w-full overflow-hidden rounded-lg border border-eco-700 bg-eco-800 shadow-xl">
          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className={cn(
                'w-full px-3 py-1.5 text-center font-mono text-xs transition-colors',
                opt.value === value
                  ? 'bg-accent/10 text-accent'
                  : 'text-eco-200 hover:bg-eco-700 hover:text-eco-50',
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { useChatStore } from '@/stores/chat-store';
import { MessageBubble } from '@/components/chat/message-bubble';
import type { ChatCompletionResponse, ModelListResponse, EcoLLMMetadata } from '@/types/api';

// ── Form schema ────────────────────────────────────────────────────────────────

const schema = z.object({
  prompt: z.string().min(1),
  maxTokens: z.coerce.number().int().min(1).max(4096).default(512),
  temperature: z.coerce.number().min(0).max(2).default(0.7),
});

type FormValues = z.infer<typeof schema>;

// ── Suggestion chips ───────────────────────────────────────────────────────────

const SUGGESTIONS = [
  'Explain how large language models work',
  'Write a Python function to merge two sorted lists',
  'What\'s the carbon footprint of running AI models?',
  'Summarize the key ideas behind transformer architecture',
] as const;

// ── MetaSidebar ────────────────────────────────────────────────────────────────

function MetaSidebar({ messages }: { messages: import('@/stores/chat-store').Message[] }) {
  const metas = messages.flatMap((m) => (m.meta ? [m.meta] : []));
  const session = {
    requests: metas.length,
    co2e_grams: metas.reduce((s, m) => s + m.energy.co2e_grams, 0),
    energy_kwh: metas.reduce((s, m) => s + m.energy.total_energy_kwh, 0),
    cost_usd: metas.reduce((s, m) => s + m.cost.total_cost_usd, 0),
    avg_latency_ms: Math.round(metas.reduce((s, m) => s + m.performance.latency_ms, 0) / metas.length),
  };
  const modelCounts = metas.reduce<Record<string, number>>((acc, m) => {
    const model = m.route.model_selected;
    acc[model] = (acc[model] ?? 0) + 1;
    return acc;
  }, {});
  const modelEntries = Object.entries(modelCounts).sort((a, b) => b[1] - a[1]);

  return (
    <div className="space-y-3 text-sm">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold uppercase tracking-widest text-eco-400">Session Data</h3>
        <span className="text-[10px] text-eco-400">{session.requests} req{session.requests !== 1 ? 's' : ''}</span>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-lg bg-accent/10 border border-accent/20 p-2.5">
          <p className="text-[10px] font-medium text-accent mb-1">CO₂e</p>
          <p className="text-xs font-semibold text-accent">{session.co2e_grams.toFixed(4)}g</p>
        </div>
        <div className="rounded-lg bg-accent/10 border border-accent/20 p-2.5">
          <p className="text-[10px] font-medium text-accent mb-1">Energy</p>
          <p className="text-xs font-semibold text-accent">{(session.energy_kwh * 1000).toFixed(4)} Wh</p>
        </div>
        <div className="rounded-lg bg-accent/10 border border-accent/20 p-2.5">
          <p className="text-[10px] font-medium text-accent mb-1">Cost</p>
          <p className="text-xs font-semibold text-accent">${session.cost_usd.toFixed(6)}</p>
        </div>
        <div className="rounded-lg bg-accent/10 border border-accent/20 p-2.5">
          <p className="text-[10px] font-medium text-accent mb-1">Avg Latency</p>
          <p className="text-xs font-semibold text-accent">{session.avg_latency_ms}ms</p>
        </div>
      </div>

      <div className="rounded-lg bg-eco-800 border border-eco-700 p-3 space-y-2.5">
        <p className="text-[10px] font-semibold uppercase tracking-widest text-eco-400">Models Used</p>
        {modelEntries.map(([model, count]) => (
          <div key={model} className="space-y-1">
            <div className="flex justify-between items-center">
              <span className="text-[10px] text-eco-200 truncate">{model.replace(/_/g, '-')}</span>
              <span className="text-[10px] font-semibold text-accent ml-2 flex-shrink-0">{count}×</span>
            </div>
            <div className="h-1 rounded-full bg-eco-700 overflow-hidden">
              <div
                className="h-full rounded-full bg-accent/70 transition-all duration-300"
                style={{ width: `${(count / session.requests) * 100}%` }}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Page ───────────────────────────────────────────────────────────────────────

export default function PlaygroundPage() {
  const {
    conversations, activeConversationId, selectedModel, apiKey, systemPrompt,
    prefer, settingsOpen, metaSidebarOpen, isStreaming, streamingMessageId,
    addUserMessage, startAssistantMessage, appendToken, completeAssistantMessage,
    addErrorMessage, removeLastAssistantMessage, editUserMessage,
    setSystemPrompt, setPrefer,
    toggleSettings, toggleMetaSidebar,
  } = useChatStore();

  const activeConv = conversations.find((c) => c.id === activeConversationId);
  const messages = activeConv?.messages ?? [];
  const lastMeta = activeConv?.lastMeta ?? null;

  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const { register, handleSubmit, reset, getValues, setValue, watch } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { prompt: '', maxTokens: 512, temperature: 0.7 },
  });

  // Model list from API
  const { data: modelsData } = useQuery({
    queryKey: ['models'],
    queryFn: () => api.get<ModelListResponse>('/v1/models'),
    staleTime: 60_000,
  });
  const models = modelsData?.models ?? [];

  // Auto-scroll to bottom on new content
  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
  }, [messages.length, streamingMessageId]);

  // Auto-grow textarea
  const adjustTextareaHeight = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = 'auto';
    el.style.height = `${Math.min(el.scrollHeight, 150)}px`;
  }, []);

  const promptValue = watch('prompt');
  useEffect(() => {
    adjustTextareaHeight();
  }, [promptValue, adjustTextareaHeight]);

  // ── Core streaming request ────────────────────────────────────────────────

  const streamRequest = useCallback(
    async (
      history: Array<{ role: string; content: string }>,
      opts: { maxTokens: number; temperature: number },
    ) => {
      const token = apiKey || api.getToken();
      if (!token) {
        addErrorMessage('Session expired — please refresh and log in again.');
        return;
      }

      const controller = new AbortController();
      abortRef.current = controller;
      startAssistantMessage();

      const baseUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

      try {
        const res = await fetch(`${baseUrl}/v1/chat/completions`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${token}`,
          },
          signal: controller.signal,
          body: JSON.stringify({
            messages: history,
            ...(selectedModel ? { model: selectedModel } : {}),
            max_tokens: opts.maxTokens,
            temperature: opts.temperature,
            stream: true,
            ecollm: { prefer, include_metadata: true },
          }),
        });

        if (!res.ok) {
          const errText = await res.text().catch(() => `HTTP ${res.status}`);
          addErrorMessage(`API error: ${errText}`);
          return;
        }

        let lastMeta: EcoLLMMetadata | null = null;
        const contentType = res.headers.get('content-type') ?? '';
        const isSSE =
          contentType.includes('text/event-stream') ||
          contentType.includes('application/octet-stream');

        if (isSSE && res.body) {
          const reader = res.body.getReader();
          const decoder = new TextDecoder();
          let buf = '';

          outer: while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buf += decoder.decode(value, { stream: true });
            const lines = buf.split('\n');
            buf = lines.pop() ?? '';

            for (const line of lines) {
              if (!line.startsWith('data: ')) continue;
              const raw = line.slice(6).trim();
              if (raw === '[DONE]') break outer;
              try {
                const chunk = JSON.parse(raw);
                const delta = chunk.choices?.[0]?.delta?.content;
                if (delta) appendToken(delta);
                if (chunk.ecollm) lastMeta = chunk.ecollm as EcoLLMMetadata;
              } catch {
                // ignore parse errors mid-stream
              }
            }
          }
        } else {
          // Non-streaming response (stream: true not supported by this endpoint variant).
          const json: ChatCompletionResponse = await res.json();
          const content = json.choices?.[0]?.message?.content ?? '';
          appendToken(content);
          lastMeta = json.ecollm ?? null;
        }

        completeAssistantMessage(lastMeta);
      } catch (err) {
        if ((err as Error).name === 'AbortError') {
          completeAssistantMessage(null);
        } else {
          addErrorMessage(`Error: ${err instanceof Error ? err.message : 'request failed'}`);
        }
      } finally {
        abortRef.current = null;
      }
    },
    [apiKey, selectedModel, prefer, startAssistantMessage, appendToken, completeAssistantMessage, addErrorMessage],
  );

  // ── Send a new message ────────────────────────────────────────────────────

  const sendMessage = useCallback(
    async (prompt: string, opts: { maxTokens: number; temperature: number }) => {
      // Capture history before modifying store
      const storeState = useChatStore.getState();
      const conv = storeState.conversations.find((c) => c.id === storeState.activeConversationId);
      const currentMessages = conv?.messages ?? [];

      const history = [
        ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
        ...currentMessages.map((m) => ({ role: m.role, content: m.content })),
        { role: 'user', content: prompt },
      ];

      addUserMessage(prompt);
      await streamRequest(history, opts);
    },
    [systemPrompt, addUserMessage, streamRequest],
  );

  // ── Form submit ───────────────────────────────────────────────────────────

  const doSubmit = useCallback(
    async (data: FormValues) => {
      if (isStreaming) return;
      reset({ ...getValues(), prompt: '' });
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }
      await sendMessage(data.prompt, { maxTokens: data.maxTokens, temperature: data.temperature });
    },
    [isStreaming, reset, getValues, sendMessage],
  );

  const onFormSubmit = handleSubmit(doSubmit);

  // ── Keyboard: Enter = send, Shift+Enter = newline ──────────────────────

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        if (!isStreaming) onFormSubmit(e as unknown as React.FormEvent);
      }
    },
    [isStreaming, onFormSubmit],
  );

  // ── Stop streaming ────────────────────────────────────────────────────────

  const handleStop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  // ── Regenerate ────────────────────────────────────────────────────────────

  const handleRegenerate = useCallback(async () => {
    const storeState = useChatStore.getState();
    const conv = storeState.conversations.find((c) => c.id === storeState.activeConversationId);
    if (!conv) return;

    const msgs = conv.messages;
    if (msgs[msgs.length - 1]?.role !== 'assistant') return;

    const remaining = msgs.slice(0, -1);
    const opts = getValues();
    removeLastAssistantMessage();

    const history = [
      ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
      ...remaining.map((m) => ({ role: m.role, content: m.content })),
    ];

    await streamRequest(history, { maxTokens: opts.maxTokens, temperature: opts.temperature });
  }, [systemPrompt, removeLastAssistantMessage, streamRequest, getValues]);

  // ── Edit user message ─────────────────────────────────────────────────────

  const handleEdit = useCallback(
    async (messageId: string, newContent: string) => {
      editUserMessage(messageId, newContent);

      // Read updated state after edit
      const storeState = useChatStore.getState();
      const conv = storeState.conversations.find((c) => c.id === storeState.activeConversationId);
      const opts = getValues();

      const history = [
        ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
        ...(conv?.messages ?? []).map((m) => ({ role: m.role, content: m.content })),
      ];

      await streamRequest(history, { maxTokens: opts.maxTokens, temperature: opts.temperature });
    },
    [editUserMessage, systemPrompt, streamRequest, getValues],
  );

  // ── Suggestion chips ──────────────────────────────────────────────────────

  const handleSuggestion = useCallback(
    async (text: string) => {
      if (isStreaming) return;
      const opts = getValues();
      await sendMessage(text, { maxTokens: opts.maxTokens, temperature: opts.temperature });
    },
    [isStreaming, getValues, sendMessage],
  );

  // ── Textarea ref merge ────────────────────────────────────────────────────

  const { ref: formRef, ...promptRest } = register('prompt');

  return (
    <div className="flex h-[calc(100vh-4rem)] gap-0 -mx-5 -my-6 sm:-mx-6">
      {/* ── Main chat column ──────────────────────────────────────────── */}
      <div className="relative flex flex-1 flex-col min-w-0">

        {/* Toolbar */}
        <div className="flex items-center justify-between border-b border-eco-700 bg-eco-900 px-4 pt-4 pb-2 gap-3">
          <div className="flex items-center gap-2">
            <CustomSelect
              value={prefer}
              onChange={(v) => setPrefer(v as typeof prefer)}
              options={[
                { value: 'efficiency', label: 'Eco Efficiency' },
                { value: 'speed', label: 'Speed' },
                { value: 'quality', label: 'Quality' },
              ]}
            />
            <button
              type="button"
              onClick={toggleSettings}
              aria-label="Toggle settings"
              className="flex items-center gap-1.5 rounded-md border border-eco-700 bg-eco-800 px-2.5 py-1 font-mono text-xs text-eco-200 hover:border-eco-600 hover:text-eco-100 transition-colors"
            >
              <span>Settings</span>
              {settingsOpen ? <ChevronUp className="h-3 w-3 text-eco-500" /> : <ChevronDown className="h-3 w-3 text-eco-500" />}
            </button>
          </div>

          <div className="flex items-center">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              onClick={toggleMetaSidebar}
              aria-label="Toggle metadata panel"
            >
              {metaSidebarOpen
                ? <PanelRightClose className="h-3.5 w-3.5" />
                : <PanelRightOpen className="h-3.5 w-3.5" />}
            </Button>
          </div>
        </div>

        {/* Settings panel — floats over the chat area, doesn't push content */}
        {settingsOpen && (
          <>
            {/* Backdrop — click outside to close */}
            <div className="absolute inset-0 z-10" onClick={toggleSettings} />
            {/* Panel — sits below the toolbar (top-[41px]) */}
            <div className="absolute left-0 right-0 top-[41px] z-20 mx-3 mt-1 rounded-xl border border-white/10 bg-eco-900/80 px-8 py-4 shadow-2xl backdrop-blur-xl">
              <div className="flex items-center gap-4">
                <div className="flex flex-1 items-center gap-2">
                  <span className="flex-shrink-0 font-mono text-xs text-eco-200">System Prompt</span>
                  <textarea
                    value={systemPrompt}
                    onChange={(e) => setSystemPrompt(e.target.value)}
                    rows={1}
                    className="flex-1 resize-none rounded-md border border-eco-600 bg-eco-800 px-3 py-1.5 text-xs text-eco-100 placeholder:text-eco-500 focus:border-accent focus:outline-none"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <span className="flex-shrink-0 font-mono text-xs text-eco-200">Max Tokens</span>
                  <input
                    type="number"
                    {...register('maxTokens')}
                    className="w-20 rounded-md border border-eco-600 bg-eco-800 px-3 py-1.5 text-xs text-eco-100 focus:border-accent focus:outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                  />
                </div>
                <div className="flex items-center gap-2">
                  <span className="flex-shrink-0 font-mono text-xs text-eco-200">Temperature</span>
                  <input
                    type="number"
                    step="0.1"
                    {...register('temperature')}
                    className="w-16 rounded-md border border-eco-600 bg-eco-800 px-3 py-1.5 text-xs text-eco-100 focus:border-accent focus:outline-none [appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none"
                  />
                </div>
                <button
                  type="button"
                  onClick={toggleSettings}
                  className="flex-shrink-0 text-eco-300 hover:text-eco-100 transition-colors"
                  aria-label="Close settings"
                >
                  <ChevronUp className="h-4 w-4" />
                </button>
              </div>
            </div>
          </>
        )}

        {/* Message thread */}
        <div
          ref={scrollRef}
          className="flex-1 overflow-y-auto px-4 py-6 space-y-6"
        >
          {messages.length === 0 ? (
            /* Empty state */
            <div className="flex h-full flex-col items-center justify-center gap-6 px-4 text-center">
              <div className="flex flex-col items-center gap-3">
                <Leaf className="h-10 w-10 text-accent" />
                <div>
                  <h2 className="text-xl font-semibold text-eco-50">EcoLLM</h2>
                  <p className="mt-1 text-sm text-eco-400">Low-carbon Language Models</p>
                </div>
              </div>

              <div className="grid max-w-lg grid-cols-1 gap-2 sm:grid-cols-2">
                {SUGGESTIONS.map((s) => (
                  <button
                    key={s}
                    onClick={() => handleSuggestion(s)}
                    disabled={isStreaming}
                    className={cn(
                      'rounded-xl border border-eco-700 bg-eco-800/50 px-4 py-3 text-left text-xs text-eco-300',
                      'hover:border-accent/40 hover:bg-eco-800 hover:text-eco-100 transition-all',
                      'disabled:pointer-events-none disabled:opacity-40',
                    )}
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            messages.map((msg, i) => (
              <MessageBubble
                key={msg.id}
                message={msg}
                isStreaming={msg.id === streamingMessageId}
                isLast={i === messages.length - 1}
                onRegenerate={
                  i === messages.length - 1 && msg.role === 'assistant' && !isStreaming
                    ? handleRegenerate
                    : undefined
                }
                onEdit={msg.role === 'user' && !isStreaming ? handleEdit : undefined}
              />
            ))
          )}
        </div>

        {/* Input area */}
        <div className="border-t border-eco-700 bg-eco-900 px-4 py-3">
          <form onSubmit={onFormSubmit}>
            <div className={cn(
              'flex overflow-hidden rounded-xl border bg-eco-800 transition-colors',
              'border-eco-700 focus-within:border-accent focus-within:ring-1 focus-within:ring-accent',
            )}>
              <textarea
                {...promptRest}
                ref={(el) => {
                  formRef(el);
                  (textareaRef as React.MutableRefObject<HTMLTextAreaElement | null>).current = el;
                }}
                onKeyDown={handleKeyDown}
                rows={1}
                placeholder="Message EcoLLM… (Enter to send, Shift+Enter for newline)"
                disabled={isStreaming}
                style={{ minHeight: '2.75rem', maxHeight: '9rem' }}
                className="flex-1 resize-none overflow-y-auto bg-transparent py-3 pl-4 pr-2 text-sm text-eco-100 placeholder-eco-400 focus:outline-none disabled:opacity-50"
              />
              {isStreaming ? (
                <button
                  type="button"
                  onClick={handleStop}
                  aria-label="Stop generation"
                  className="flex w-12 flex-shrink-0 items-center justify-center self-stretch bg-red-600/20 text-red-400 transition-colors hover:bg-red-600/30"
                >
                  <Square className="h-4 w-4" />
                </button>
              ) : (
                <button
                  type="submit"
                  aria-label="Send message"
                  className="flex w-12 flex-shrink-0 items-center justify-center self-stretch bg-eco-500 text-accent transition-colors hover:bg-eco-400"
                >
                  <Send className="h-4 w-4" />
                </button>
              )}
            </div>
          </form>
        </div>
      </div>

      {/* ── Metadata sidebar ──────────────────────────────────────────── */}
      {metaSidebarOpen && (
        <aside className="hidden w-64 flex-shrink-0 border-l border-eco-700 bg-eco-900 overflow-y-auto p-4 xl:block">
          {messages.some((m) => m.meta) ? (
            <MetaSidebar messages={messages} />
          ) : (
            <div className="flex h-full items-center justify-center text-center px-4">
              <div className="space-y-2">
                <div className="flex justify-center">
                  <Leaf className="h-8 w-8 text-eco-700" />
                </div>
                <p className="text-xs text-eco-500">
                  Routing metadata will appear here after your first request.
                </p>
              </div>
            </div>
          )}
        </aside>
      )}
    </div>
  );
}
