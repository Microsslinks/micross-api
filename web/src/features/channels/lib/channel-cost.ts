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
import type { QueryClient } from '@tanstack/react-query'
import i18next from 'i18next'
import { toast } from 'sonner'

import { batchSetChannelCost } from '../api'
import { channelsQueryKeys } from './channel-actions'

/**
 * 进货折扣多久没更新就在列表里提示「该核一下了」。
 * 与后端 /api/channel/cost/stale 的默认天数保持一致。
 */
export const COST_STALE_DAYS = 30

/**
 * 进货折扣的上限，与后端 controller/channel_cost.go 的 maxChannelCostRatio 对齐。
 * 进价高于标价是可能的（亏本引流），但不设上限的话「2.7 折」很容易被写成「27」。
 */
export const MAX_COST_RATIO = 100

export const channelCostQueryKeys = {
  all: ['channel-cost'] as const,
  stale: (days: number) =>
    [...channelCostQueryKeys.all, 'stale', days] as const,
}

/**
 * 一条渠道的进货折扣算不算「有」。空白、非数字、非正数都等于没录。
 */
export function isCostRatioConfigured(costRatio?: null | string): boolean {
  const raw = (costRatio ?? '').trim()
  if (!raw) return false

  const parsed = Number(raw)
  return Number.isFinite(parsed) && parsed > 0
}

/**
 * 表单输入框里的值能不能用。跟上面那个的区别是「空」在这里是合法的——
 * 空表示还没录，不是填错了。与后端 validateChannelCostRatio 同一口径。
 */
export function isCostRatioInputAllowed(costRatio?: null | string): boolean {
  const raw = (costRatio ?? '').trim()
  if (!raw) return true

  return isCostRatioConfigured(raw)
}

/** 填了值就不能超过上限，否则就是「0.27 写成 27」那种手滑。 */
export function isCostRatioWithinMax(costRatio?: null | string): boolean {
  if (!isCostRatioConfigured(costRatio)) return true

  return Number((costRatio ?? '').trim()) <= MAX_COST_RATIO
}

/**
 * 已录入的进货折扣是不是太久没更新了。没有录入时间的一律算过期——
 * 那种值多半是绕过录入界面写进来的，无从判断新旧。
 */
export function isCostRatioStale(
  costUpdatedAt?: null | number,
  days = COST_STALE_DAYS
): boolean {
  if (!costUpdatedAt || costUpdatedAt <= 0) return true

  const nowSeconds = Date.now() / 1000
  return nowSeconds - costUpdatedAt > days * 24 * 60 * 60
}

/**
 * 折叠掉多余的零：0.270000 显示成 0.27，与录入时的写法一致。
 */
export function formatCostRatio(costRatio?: null | string): string {
  if (!isCostRatioConfigured(costRatio)) return ''

  return String(Number((costRatio ?? '').trim()))
}

/**
 * 换成人话的百分比：0.27 -> 27%，用于 tooltip 里解释这个数是什么意思。
 */
export function formatCostRatioPercent(costRatio?: null | string): string {
  if (!isCostRatioConfigured(costRatio)) return ''

  const percent = Number((costRatio ?? '').trim()) * 100
  return `${Number.isInteger(percent) ? percent : percent.toFixed(1)}%`
}

/**
 * 批量给选中的渠道录同一个进货折扣。传空串表示清空。
 */
export async function handleBatchSetCost(
  ids: number[],
  costRatio: string,
  queryClient?: QueryClient,
  onSuccess?: () => void
): Promise<void> {
  if (ids.length === 0) {
    toast.error(i18next.t('No channels selected'))
    return
  }

  try {
    const response = await batchSetChannelCost({ ids, cost_ratio: costRatio })
    if (response.success) {
      const count = response.data ?? ids.length
      if (costRatio) {
        toast.success(
          i18next.t('Cost ratio {{ratio}} set for {{count}} channel(s)', {
            ratio: formatCostRatio(costRatio),
            count,
          })
        )
      } else {
        toast.success(
          i18next.t('Cost ratio cleared for {{count}} channel(s)', { count })
        )
      }
      queryClient?.invalidateQueries({ queryKey: channelsQueryKeys.lists() })
      queryClient?.invalidateQueries({ queryKey: channelCostQueryKeys.all })
      onSuccess?.()
    } else {
      toast.error(response.message || i18next.t('Failed to set cost ratio'))
    }
  } catch {
    toast.error(i18next.t('Failed to set cost ratio'))
  }
}
