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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  PromptInput,
  PromptInputFooter,
  PromptInputTextarea,
  type PromptInputMessage,
} from '@/components/ai-elements/prompt-input'
import { cn } from '@/lib/utils'

import { getSubmittableInputText } from '../../lib'
import type { ModelOption, ParameterEnabled, PlaygroundConfig } from '../../types'
import { PlaygroundInputControls } from './playground-input-controls'
import { PlaygroundInputTools } from './playground-input-tools'

interface PlaygroundInputProps {
  config: PlaygroundConfig
  onSubmit: (text: string) => void
  onStop?: () => void
  disabled?: boolean
  isGenerating?: boolean
  models: ModelOption[]
  onConfigChange: <K extends keyof PlaygroundConfig>(
    key: K,
    value: PlaygroundConfig[K]
  ) => void
  onParameterEnabledChange: (
    key: keyof ParameterEnabled,
    value: boolean
  ) => void
  parameterEnabled: ParameterEnabled
}

export function PlaygroundInput({
  config,
  onSubmit,
  onStop,
  disabled,
  isGenerating,
  models,
  onConfigChange,
  onParameterEnabledChange,
  parameterEnabled,
}: PlaygroundInputProps) {
  const { t } = useTranslation()
  const [text, setText] = useState('')

  const handleSubmit = (message: PromptInputMessage) => {
    const submittableText = getSubmittableInputText(message, disabled)

    if (!submittableText) return
    onSubmit(submittableText)
    setText('')
  }

  return (
    <div className='grid shrink-0 gap-2 px-1 pb-4 md:pb-6'>
      <PromptInput
        className='relative'
        groupClassName={cn(
          'bg-background/95 dark:bg-background/80 border-border/70',
          'shadow-[0_18px_60px_-32px_rgba(0,0,0,0.65)] ring-1 ring-foreground/5',
          'rounded-2xl overflow-hidden transition-all duration-200',
          'focus-within:border-primary/45 focus-within:ring-primary/20',
          'focus-within:shadow-[0_22px_70px_-30px_rgba(0,0,0,0.75)]',
          'focus-within:shadow-primary/10'
        )}
        onSubmit={handleSubmit}
      >
        <PromptInputTextarea
          autoCapitalize='off'
          autoComplete='off'
          autoCorrect='off'
          className='min-h-22 px-5 pt-4 pb-2 text-[0.95rem] leading-7 md:min-h-26 md:text-base'
          disabled={disabled}
          onChange={(event) => setText(event.target.value)}
          placeholder={t('Ask anything — press Enter to send')}
          spellCheck={false}
          value={text}
        />

        <PromptInputFooter className='border-border/60 bg-muted/20 dark:bg-muted/10 border-t px-2 py-2 backdrop-blur'>
          <PlaygroundInputControls
            disabled={disabled}
            isGenerating={isGenerating}
            models={models}
            onStop={onStop}
            text={text}
            tools={
              <PlaygroundInputTools
                config={config}
                disabled={disabled}
                onConfigChange={onConfigChange}
                onParameterEnabledChange={onParameterEnabledChange}
                parameterEnabled={parameterEnabled}
              />
            }
          />
        </PromptInputFooter>
      </PromptInput>

      <div className='text-muted-foreground/70 flex items-center justify-center gap-3 text-[10px] tracking-wide uppercase'>
        <kbd className='bg-muted/60 text-muted-foreground/80 rounded border border-border/60 px-1.5 py-0.5 font-mono'>
          Enter
        </kbd>
        <span>{t('to send')}</span>
        <span className='text-muted-foreground/40'>·</span>
        <kbd className='bg-muted/60 text-muted-foreground/80 rounded border border-border/60 px-1.5 py-0.5 font-mono'>
          Shift + Enter
        </kbd>
        <span>{t('for newline')}</span>
      </div>
    </div>
  )
}