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
  CodeSquareIcon,
  GraduationCapIcon,
  LightbulbIcon,
  MessagesSquareIcon,
  NotepadTextIcon,
  PenLineIcon,
  SparklesIcon,
  WandSparklesIcon,
  type LucideIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

type PlaygroundEmptyStateProps = {
  onSelectPrompt: (prompt: string) => void
}

type StarterPrompt = {
  icon: LucideIcon
  title: string
  prompt: string
  accent: string
}

const STARTER_PROMPTS: readonly StarterPrompt[] = [
  {
    icon: NotepadTextIcon,
    title: 'Summarize',
    prompt: 'Summarize the following text in three bullet points: ',
    accent: 'from-sky-500/15 to-sky-500/0 text-sky-600 dark:text-sky-400',
  },
  {
    icon: CodeSquareIcon,
    title: 'Write code',
    prompt: 'Write a TypeScript function that debounces an async handler.',
    accent:
      'from-emerald-500/15 to-emerald-500/0 text-emerald-600 dark:text-emerald-400',
  },
  {
    icon: BrainIcon,
    title: 'Reason step by step',
    prompt:
      'Walk me through your reasoning step by step before answering: why does the sky appear blue?',
    accent:
      'from-violet-500/15 to-violet-500/0 text-violet-600 dark:text-violet-400',
  },
  {
    icon: PenLineIcon,
    title: 'Draft an email',
    prompt:
      'Draft a friendly follow-up email to a customer asking about a delayed order.',
    accent: 'from-rose-500/15 to-rose-500/0 text-rose-600 dark:text-rose-400',
  },
  {
    icon: GraduationCapIcon,
    title: 'Explain a concept',
    prompt:
      'Explain how a transformer attention mechanism works, in plain language.',
    accent:
      'from-amber-500/15 to-amber-500/0 text-amber-600 dark:text-amber-400',
  },
  {
    icon: LightbulbIcon,
    title: 'Brainstorm ideas',
    prompt:
      'Give me ten product names for an AI-powered focus timer app for developers.',
    accent:
      'from-indigo-500/15 to-indigo-500/0 text-indigo-600 dark:text-indigo-400',
  },
]

export function PlaygroundEmptyState({
  onSelectPrompt,
}: PlaygroundEmptyStateProps) {
  const { t } = useTranslation()

  return (
    <div className='relative flex min-h-[min(540px,calc(100svh-18rem))] items-center justify-center px-1 py-8 md:py-14'>
      <div className='grid w-full max-w-2xl gap-7 text-center'>
        <div className='grid gap-4'>
          <div
            aria-hidden='true'
            className={cn(
              'mx-auto grid size-14 place-items-center rounded-2xl',
              'from-primary/30 via-primary/15 to-primary/5',
              'bg-gradient-to-br shadow-[inset_0_1px_0_0_rgba(255,255,255,0.6)]',
              'ring-primary/20 ring-1'
            )}
          >
            <SparklesIcon className='text-primary size-6' />
          </div>

          <div className='grid gap-2'>
            <h2 className='text-2xl font-semibold tracking-tight text-balance md:text-3xl'>
              {t('How can I help today?')}
            </h2>
            <p className='text-muted-foreground mx-auto max-w-md text-sm leading-6 text-balance'>
              {t(
                'Pick a starter prompt below, or type your own request to begin a streaming chat with any model in your workspace.'
              )}
            </p>
          </div>
        </div>

        <div className='grid gap-3 sm:grid-cols-2 md:gap-3.5'>
          {STARTER_PROMPTS.map(({ accent, icon: Icon, prompt, title }) => (
            <button
              className={cn(
                'group relative flex h-auto min-h-20 items-start gap-3 rounded-xl border',
                'border-border/70 bg-background/60 p-3.5 text-left backdrop-blur-sm',
                'transition-all duration-200 ease-out',
                'hover:-translate-y-0.5 hover:border-border hover:shadow-md',
                'hover:shadow-black/5 focus-visible:outline-none focus-visible:ring-2',
                'focus-visible:ring-primary/30 focus-visible:ring-offset-2 focus-visible:ring-offset-background'
              )}
              key={title}
              onClick={() => onSelectPrompt(t(prompt))}
              type='button'
            >
              <div
                aria-hidden='true'
                className={cn(
                  'grid size-9 shrink-0 place-items-center rounded-lg bg-gradient-to-br',
                  accent
                )}
              >
                <Icon className='size-4' />
              </div>

              <div className='grid min-w-0 gap-0.5'>
                <div className='flex items-center gap-1.5'>
                  <span className='text-sm font-medium leading-5'>
                    {t(title)}
                  </span>
                  <WandSparklesIcon
                    className='text-muted-foreground/0 size-3.5 transition-all duration-200 group-hover:text-primary group-hover:translate-x-0.5'
                  />
                </div>
                <span className='text-muted-foreground line-clamp-2 text-xs leading-5'>
                  {t(prompt)}
                </span>
              </div>
            </button>
          ))}
        </div>

        <div className='text-muted-foreground/60 flex items-center justify-center gap-2 pt-1 text-[11px]'>
          <MessagesSquareIcon className='size-3.5' aria-hidden='true' />
          <span>{t('Tip: switch models anytime from the header above')}</span>
        </div>
      </div>
    </div>
  )
}