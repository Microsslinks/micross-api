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
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { getDiscountModelLists } from '../api'
import { DISCOUNT_MODEL_LIST_LIMITS } from '../constants'
import type { DiscountModelList } from '../types'

/**
 * 清单列表的缓存键：维护页与「模型清单」零件的「一键带入」共用同一份。
 * 在维护页里新建 / 改名 / 删掉之后，另一边（尤其是已经开着的报价核算抽屉）
 * 必须立刻看到新结果，所以两边都靠这一个键来失效。
 */
export const DISCOUNT_MODEL_LISTS_QUERY_KEY = ['discount_model_lists'] as const

/**
 * 可复用的模型清单列表。
 * 拿不到就返回空清单：调用方照旧手输与粘贴，只是没有可带入的名单，
 * 所以这里不把错误抛给界面（与模型目录那一份选项同一个口径）。
 */
export function useDiscountModelLists() {
  const query = useQuery({
    queryKey: DISCOUNT_MODEL_LISTS_QUERY_KEY,
    queryFn: async (): Promise<DiscountModelList[]> => {
      const response = await getDiscountModelLists({
        page_size: DISCOUNT_MODEL_LIST_LIMITS.PAGE_SIZE,
      })
      return response.success ? (response.data?.items ?? []) : []
    },
    staleTime: 60 * 1000,
  })

  return { lists: query.data ?? [], isLoading: query.isLoading }
}

/** 改动清单之后调它，让维护页与所有「一键带入」都重新读一次。 */
export function useRefreshDiscountModelLists() {
  const queryClient = useQueryClient()
  return () =>
    queryClient.invalidateQueries({ queryKey: DISCOUNT_MODEL_LISTS_QUERY_KEY })
}
