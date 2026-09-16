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
import { useState, useEffect, useCallback } from 'react'

import {
  getCommissionBalance,
  listCommissionRecords,
  getCommissionSummary,
} from '../api'
import type {
  CommissionRecord,
  CommissionSummaryResponse,
} from '@/features/wallet/types'

// 去掉 ApiResponse 外壳后的 summary 形状
type CommissionSummary = NonNullable<CommissionSummaryResponse['data']>

// ============================================================================
// Commission Hook
//
// 并行拉 balance / records / summary；分页切换 records。
// 配套前端 wallet 面板：展示独立钱包余额 + 历史流水 + 汇总。
//
// 注意：与 useAffiliate 的语义区别——affiliate quota 可转账到主 quota，
// commission 永远不允许（见 transferCommissionQuota 的 doc）。
// ============================================================================

export function useCommission(pageSize = 20) {
  const [balance, setBalance] = useState<number>(0)
  const [records, setRecords] = useState<CommissionRecord[]>([])
  const [summary, setSummary] = useState<CommissionSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)

  const fetchAll = useCallback(async () => {
    try {
      setLoading(true)
      const [bal, recs, sum] = await Promise.all([
        getCommissionBalance(),
        listCommissionRecords(page, pageSize),
        getCommissionSummary(),
      ])
      if (bal.success && bal.data) setBalance(bal.data.balance)
      if (recs.success && recs.data) {
        setRecords(recs.data.items)
        setTotal(recs.data.total)
      }
      if (sum.success && sum.data) setSummary(sum.data)
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch commission data:', error)
    } finally {
      setLoading(false)
    }
  }, [page, pageSize])

  useEffect(() => {
    fetchAll()
  }, [fetchAll])

  return {
    balance,
    records,
    summary,
    loading,
    page,
    total,
    pageSize,
    setPage,
    refetch: fetchAll,
  }
}