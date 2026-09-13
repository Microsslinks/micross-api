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
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { FadeIn } from '@/components/page-transition'
import {
  EMPTY_SIZE,
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  type EmptySize,
} from '@/components/ui/empty'
import { cn } from '@/lib/utils'

interface LoadingStateProps {
  className?: string
  message?: string
  /** 区块高度档位，默认 md。弹窗与小面板用 sm，整页与表格用 lg。 */
  size?: EmptySize
  inline?: boolean
}

const sizeMap = {
  sm: 'size-4',
  md: 'size-5',
  lg: 'size-6',
} as const

export function LoadingState(props: LoadingStateProps) {
  const { t } = useTranslation()
  const iconSize = sizeMap[props.size ?? 'md']

  if (props.inline) {
    return (
      <span className={cn('inline-flex items-center gap-2', props.className)}>
        <Loader2 className={cn(iconSize, 'animate-spin')} />
        {props.message != null && (
          <span className='text-muted-foreground text-sm'>{props.message}</span>
        )}
      </span>
    )
  }

  // 与 EmptyState / ErrorState 共用同一套几何：图标徽章 + 次级文案，区块高度走 EMPTY_SIZE。
  // FadeIn 这一层带上 flex 布局，Empty 的 flex-1 才能真正撑开父容器。
  return (
    <FadeIn className='flex min-h-0 flex-1 flex-col'>
      <Empty className={cn(EMPTY_SIZE[props.size ?? 'md'], props.className)}>
        <EmptyHeader>
          <EmptyMedia variant='icon'>
            <Loader2 className={cn(iconSize, 'animate-spin')} />
          </EmptyMedia>
          <EmptyDescription>
            {props.message ?? t('Loading...')}
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    </FadeIn>
  )
}
