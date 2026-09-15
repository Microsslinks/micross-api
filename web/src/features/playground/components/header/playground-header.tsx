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
  CircleHelpIcon,
  PanelTopIcon,
  SparklesIcon,
  Trash2Icon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ModelGroupSelector } from '@/components/model-group-selector'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'

import type { GroupOption, ModelOption } from '../../types'

interface PlaygroundHeaderProps {
  models: ModelOption[]
  modelValue: string
  onModelChange: (value: string) => void
  isModelLoading?: boolean
  groups: GroupOption[]
  groupValue: string
  onGroupChange: (value: string) => void
  hasMessages?: boolean
  onClearMessages?: () => void
  isGenerating?: boolean
}

export function PlaygroundHeader({
  models,
  modelValue,
  onModelChange,
  isModelLoading = false,
  groups,
  groupValue,
  onGroupChange,
  hasMessages = false,
  onClearMessages,
  isGenerating = false,
}: PlaygroundHeaderProps) {
  const { t } = useTranslation()
  const [clearConfirmOpen, setClearConfirmOpen] = useState(false)
  const currentModel = models.find((m) => m.value === modelValue)

  return (
    <header
      className={cn(
        'bg-background/70 supports-[backdrop-filter]:bg-background/55',
        'border-border/60 sticky top-0 z-20 flex h-14 shrink-0 items-center gap-3',
        'border-b px-4 backdrop-blur-xl backdrop-saturate-150 md:gap-4 md:px-6'
      )}
    >
      <div className='flex min-w-0 items-center gap-2.5'>
        <div
          aria-hidden='true'
          className='from-primary/20 via-primary/10 to-primary/5 ring-primary/15 grid size-8 shrink-0 place-items-center rounded-lg bg-gradient-to-br shadow-[inset_0_1px_0_0_rgba(255,255,255,0.6)] ring-1'
        >
          <SparklesIcon className='text-primary size-4' />
        </div>
        <div className='min-w-0'>
          <div className='flex items-center gap-1.5'>
            <h1 className='truncate text-sm font-semibold tracking-tight'>
              {t('Playground')}
            </h1>
            <Badge
              className='h-5 shrink-0 px-1.5 font-mono text-[10px] tracking-wide uppercase'
              variant='secondary'
            >
              <PanelTopIcon className='mr-0.5 size-3' />
              {t('Beta')}
            </Badge>
          </div>
          <p className='text-muted-foreground hidden truncate text-xs leading-4 md:block'>
            {currentModel?.label ?? t('Test any model in your workspace')}
          </p>
        </div>
      </div>

      <div className='ml-auto flex items-center gap-1.5 md:gap-2'>
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                aria-label={t('How it works')}
                className='text-muted-foreground hover:text-foreground'
                size='icon'
                variant='ghost'
              >
                <CircleHelpIcon className='size-4' />
              </Button>
            }
          />
          <TooltipContent>
            <p>{t('Stream chat completions directly from the playground')}</p>
          </TooltipContent>
        </Tooltip>

        <div className='hidden h-7 w-px bg-border/70 md:block' />

        <div className='flex min-w-0 items-center'>
          <ModelGroupSelector
            disabled={isModelLoading || isGenerating}
            groups={groups}
            models={models}
            onGroupChange={onGroupChange}
            onModelChange={onModelChange}
            selectedGroup={groupValue}
            selectedModel={modelValue}
          />
        </div>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                aria-label={t('Clear conversation')}
                className='text-muted-foreground hover:text-destructive hover:bg-destructive/10'
                disabled={!hasMessages || !onClearMessages || isGenerating}
                onClick={() => setClearConfirmOpen(true)}
                size='icon'
                variant='ghost'
              >
                <Trash2Icon className='size-4' />
              </Button>
            }
          />
          <TooltipContent>
            <p>{t('Clear conversation')}</p>
          </TooltipContent>
        </Tooltip>

        <ConfirmDialog
          destructive
          confirmText={t('Clear')}
          desc={t(
            'All playground messages saved in this browser will be removed. This cannot be undone.'
          )}
          handleConfirm={() => {
            onClearMessages?.()
            setClearConfirmOpen(false)
          }}
          open={clearConfirmOpen}
          onOpenChange={setClearConfirmOpen}
          title={t('Clear chat history?')}
        />
      </div>

      <div
        aria-hidden='true'
        className='from-primary/8 via-transparent to-primary/8 pointer-events-none absolute inset-x-0 -bottom-px h-px bg-gradient-to-r'
      />
    </header>
  )
}