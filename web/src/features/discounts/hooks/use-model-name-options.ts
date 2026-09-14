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
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getModelNameOptions } from '../api'

/**
 * 可挑的模型名清单——平台当前真正能提供服务的模型（挂了启用渠道的），
 * 和模型广场同源（abilities 表里 enabled 的模型），不是全量模型目录。
 * 三处（方案规则 / 报价核算 / 查价）共用同一份缓存，打开第二个抽屉不会再请求一次。
 * 拿不到就返回空选项：调用方仍可手输与粘贴，只是没有可挑的候选。
 */
export function useModelNameOptions() {
  const query = useQuery({
    queryKey: ['discount_model_options'],
    queryFn: getModelNameOptions,
    staleTime: 5 * 60 * 1000,
  })

  const options = useMemo(
    () => (query.data ?? []).map((name) => ({ value: name, label: name })),
    [query.data]
  )

  return { options, isCatalogUnavailable: query.isError }
}
