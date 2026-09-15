/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import {
  BrainIcon,
  SendIcon,
  StopCircleIcon,
  Trash2Icon,
} from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Conversation,
  ConversationContent,
  ConversationEmptyState,
  ConversationScrollButton,
} from '@/components/ai-elements/conversation'
import {
  Message,
  MessageContent,
} from '@/components/ai-elements/message'
import {
  PromptInput,
  PromptInputBody,
  PromptInputFooter,
  PromptInputSubmit,
  PromptInputTextarea,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'
import { Reasoning, ReasoningContent, ReasoningTrigger } from '@/components/ai-elements/reasoning'
import { Response } from '@/components/ai-elements/response'
import { Shimmer } from '@/components/ai-elements/shimmer'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Separator } from '@/components/ui/separator'
import { chatTestChannel } from '../../api'

type ChatRole = 'user' | 'assistant' | 'system'

type ChatTurn = {
  id: string
  role: ChatRole
  content: string
  reasoningContent?: string
  pending?: boolean
  error?: { message: string; code?: string }
  durationMs?: number
  usage?: { prompt_tokens: number; completion_tokens: number; total_tokens?: number }
  finishReason?: string
}

type ChannelChatTesterProps = {
  channelId: number
  channelDisplayName: string
  models: string[]
  defaultModel?: string
  endpointType: string
  stream: boolean
}

const makeId = (prefix: string) =>
  `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`

export function ChannelChatTester({
  channelId,
  channelDisplayName,
  models,
  defaultModel,
  endpointType,
  stream,
}: ChannelChatTesterProps) {
  const { t } = useTranslation()
  const initialModel = useMemo(() => {
    if (defaultModel && models.includes(defaultModel)) return defaultModel
    return models[0] ?? ''
  }, [defaultModel, models])
  const [selectedModel, setSelectedModel] = useState(initialModel)
  const [turns, setTurns] = useState<ChatTurn[]>([])
  const [isSending, setIsSending] = useState(false)

  const stop = useCallback(() => {
    // The chat endpoint is synchronous today; clearing isSending is the
    // only thing we can do on click. Keep the hook so the UI wiring stays
    // consistent if we add abortable streaming later.
    setIsSending(false)
  }, [])

  const send = useCallback(
    async (text: string) => {
      const trimmed = text.trim()
      if (!trimmed || !selectedModel || isSending) return

      const userTurn: ChatTurn = {
        id: makeId('user'),
        role: 'user',
        content: trimmed,
      }
      const pendingTurn: ChatTurn = {
        id: makeId('assistant'),
        role: 'assistant',
        content: '',
        pending: true,
      }

      setTurns((prev) => [...prev, userTurn, pendingTurn])
      setIsSending(true)

      const apiMessages = [...turns, userTurn]
        .filter((turn) => turn.role !== 'assistant' || !turn.pending)
        .map((turn) => ({ role: turn.role, content: turn.content }))

      try {
        const data = await chatTestChannel(channelId, {
          model: selectedModel,
          messages: apiMessages,
          endpoint_type: endpointType === 'auto' ? undefined : endpointType,
          stream,
          include_response: true,
        })

        setTurns((prev) =>
          prev.map((turn) => {
            if (turn.id !== pendingTurn.id) return turn
            if (data?.success && data.response) {
              return {
                ...turn,
                pending: false,
                content: (data.response.content as string | undefined) ?? '',
                reasoningContent: data.response.reasoning_content as
                  | string
                  | undefined,
                usage: data.response.usage as ChatTurn['usage'],
                finishReason: data.response.finish_reason as string | undefined,
                durationMs:
                  typeof data.time === 'number'
                    ? Math.round(data.time * 1000)
                    : undefined,
              }
            }
            return {
              ...turn,
              pending: false,
              error: {
                message: data?.message ?? t('Test failed'),
                code: data?.error_code,
              },
            }
          })
        )
      } catch (err: unknown) {
        const message =
          err instanceof Error ? err.message : t('Network error')
        setTurns((prev) =>
          prev.map((turn) =>
            turn.id === pendingTurn.id
              ? { ...turn, pending: false, error: { message } }
              : turn
          )
        )
      } finally {
        setIsSending(false)
      }
    },
    [channelId, endpointType, isSending, selectedModel, stream, turns, t]
  )

  const handleSubmit = useCallback(
    (message: PromptInputMessage) => {
      const text = (message.text ?? '').trim()
      if (!text) return
      void send(text)
    },
    [send]
  )

  const handleClear = useCallback(() => {
    setTurns([])
  }, [])

  const modelOptions = useMemo(() => models.filter(Boolean), [models])

  return (
    <div className='flex h-full min-h-0 flex-col gap-3'>
      <div className='flex shrink-0 items-center gap-2 px-1'>
        <span className='text-muted-foreground text-xs whitespace-nowrap'>
          {t('Model')}
        </span>
        <Select value={selectedModel} onValueChange={setSelectedModel}>
          <SelectTrigger className='h-8 flex-1 min-w-0'>
            <SelectValue placeholder={t('Select a model')} />
          </SelectTrigger>
          <SelectContent>
            {modelOptions.map((model) => (
              <SelectItem key={model} value={model}>
                {model}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant='ghost'
          size='icon'
          type='button'
          onClick={handleClear}
          disabled={turns.length === 0 || isSending}
          aria-label={t('Clear conversation')}
          title={t('Clear conversation')}
        >
          <Trash2Icon className='size-4' />
        </Button>
      </div>

      <Conversation className='bg-muted/30 rounded-md border'>
        <ConversationContent>
          {turns.length === 0 ? (
            <ConversationEmptyState
              icon={
                <BrainIcon className='text-muted-foreground size-6' />
              }
              title={t('Test the channel as a chat')}
              description={
                channelDisplayName
                  ? t('Channel: {{name}}', { name: channelDisplayName })
                  : t('Pick a model above and send a message')
              }
            />
          ) : (
            turns.map((turn) => (
              <ChatTurnView key={turn.id} turn={turn} />
            ))
          )}
        </ConversationContent>
        <ConversationScrollButton />
      </Conversation>

      <PromptInput
        className='shrink-0'
        groupClassName='bg-background border-border/70 rounded-xl shadow-sm'
        onSubmit={handleSubmit}
      >
        <PromptInputBody>
          <PromptInputTextarea
            className='min-h-12 px-4 py-3 leading-6 md:text-sm'
            placeholder={t('Type a message...')}
            disabled={isSending || !selectedModel}
          />
        </PromptInputBody>
        <PromptInputFooter className='border-border/60 bg-muted/10 border-t px-3 py-2'>
          <span className='text-muted-foreground text-xs'>
            {t('Enter to send · Shift+Enter for newline')}
          </span>
          {isSending ? (
            <Button
              type='button'
              size='sm'
              variant='outline'
              onClick={stop}
              className='gap-1.5'
            >
              <StopCircleIcon className='size-3.5' />
              {t('Stop')}
            </Button>
          ) : (
            <PromptInputSubmit
              status='ready'
              className='gap-1.5'
              disabled={!selectedModel}
            >
              <SendIcon className='size-3.5' />
              {t('Send')}
            </PromptInputSubmit>
          )}
        </PromptInputFooter>
      </PromptInput>
    </div>
  )
}

function ChatTurnView({ turn }: { turn: ChatTurn }) {
  const { t } = useTranslation()

  return (
    <Message
      from={turn.role}
      className={turn.role === 'user' ? 'justify-end' : 'justify-start'}
    >
      <MessageContent
        variant={turn.role === 'user' ? 'contained' : 'flat'}
        className={
          turn.role === 'user'
            ? 'max-w-[85%]'
            : 'text-foreground max-w-full leading-relaxed'
        }
      >
        {turn.reasoningContent ? (
          <Reasoning
            defaultOpen={turn.pending ?? false}
            isStreaming={turn.pending ?? false}
          >
            <ReasoningTrigger />
            <ReasoningContent>{turn.reasoningContent}</ReasoningContent>
          </Reasoning>
        ) : null}

        {turn.pending ? (
          <Shimmer>{turn.role === 'user' ? '' : t('Thinking...')}</Shimmer>
        ) : turn.error ? (
          <ErrorBlock message={turn.error.message} code={turn.error.code} />
        ) : turn.role === 'user' ? (
          <p className='m-0 leading-relaxed wrap-break-word whitespace-pre-wrap'>
            {turn.content}
          </p>
        ) : (
          <Response parserId='new-api-response'>{turn.content}</Response>
        )}

        {turn.role === 'assistant' && !turn.pending && !turn.error ? (
          <AssistantMeta
            usage={turn.usage}
            durationMs={turn.durationMs}
            finishReason={turn.finishReason}
          />
        ) : null}
      </MessageContent>
    </Message>
  )
}

function AssistantMeta({
  usage,
  durationMs,
  finishReason,
}: {
  usage?: ChatTurn['usage']
  durationMs?: number
  finishReason?: string
}) {
  const { t } = useTranslation()
  const parts: string[] = []
  if (typeof durationMs === 'number') {
    parts.push(`${(durationMs / 1000).toFixed(2)}s`)
  }
  if (usage) {
    const prompt = usage.prompt_tokens ?? 0
    const completion = usage.completion_tokens ?? 0
    const total = usage.total_tokens ?? prompt + completion
    parts.push(
      t('{{prompt}} + {{completion}} · {{total}} tokens', {
        prompt,
        completion,
        total,
      })
    )
  }
  if (finishReason) {
    parts.push(finishReason)
  }
  if (parts.length === 0) return null
  return (
    <div className='text-muted-foreground flex items-center gap-2 text-xs'>
      <Separator orientation='horizontal' className='flex-1' />
      <span className='whitespace-nowrap'>{parts.join(' · ')}</span>
    </div>
  )
}

function ErrorBlock({ message, code }: { message: string; code?: string }) {
  const { t } = useTranslation()
  return (
    <div className='border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-sm'>
      <div className='flex items-center gap-1.5 font-medium'>{t('Test failed')}</div>
      <p className='mt-1 leading-relaxed wrap-break-word'>{message}</p>
      {code ? (
        <p className='text-destructive/70 mt-1 font-mono text-xs'>{code}</p>
      ) : null}
    </div>
  )
}