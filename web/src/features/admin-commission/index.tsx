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

import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import {
  adminListCommissionRecords,
  adminReverseCommissionRecord,
} from './api'
import { AdminCommissionTable } from './components/admin-commission-table'
import {
  FiltersBar,
  type BreachFilter,
  type ReversedFilter,
} from './components/filters-bar'
import { ReverseDialog } from './components/reverse-dialog'
import type { AdminCommissionRecord } from './types'

const DEFAULT_PAGE_SIZE = 20

// ============================================================================
// Admin Commission Management Page (task-20 §20.8)
//
// 把 §20.7 的两个 HTTP 端点包成单页 admin 视图：
//   - GET  /api/admin/commission/records         列表 + 过滤
//   - POST /api/admin/commission/records/:id/reverse  撤销
//
// 关键流程：
//   1. 页面加载时拉默认 page=1 的 20 条。
//   2. 过滤器点"应用"触发 reloadFilters() → 拉新数据；过滤器即时改 state
//      但不立即发请求，避免输入到一半的非法值触 500。
//   3. 撤销成功后乐观刷新（不重发 GET，删除本地 row；保留 total 减 1
//      但保留分页位置以免分页跳变）。
//   4. 撤销按钮点击只打开对话框；用户输入 reason 并按"确认撤销"才调 API。
//
// 权限：admin group 校验由 router 中间件 AdminAuth() 保证；
// 这层组件不做额外 role 检查。
// ============================================================================
export function AdminCommissionPage() {
  const { t } = useTranslation()

  // 数据状态
  const [records, setRecords] = useState<AdminCommissionRecord[]>([])
  const [page, setPage] = useState(1)
  const [pageSize] = useState(DEFAULT_PAGE_SIZE)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  // 过滤 state（user 在编辑区，"应用"按钮才提交）
  const [inviterIdInput, setInviterIdInput] = useState('')
  const [inviteeIdInput, setInviteeIdInput] = useState('')
  const [reversedFilter, setReversedFilter] = useState<ReversedFilter>('all')
  const [breachFilter, setBreachFilter] = useState<BreachFilter>('all')

  // 真正提交的过滤 state（点击"应用"才同步给这个）
  const [applied, setApplied] = useState({
    inviterId: '',
    inviteeId: '',
    reversed: 'all' as ReversedFilter,
    breach: 'all' as BreachFilter,
  })

  // 撤销对话框 state
  const [pending, setPending] = useState<AdminCommissionRecord | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const fetchList = useCallback(async () => {
    setLoading(true)
    try {
      const filters: Parameters<typeof adminListCommissionRecords>[0] = {
        page,
        page_size: pageSize,
      }
      if (applied.inviterId.trim() !== '') {
        const n = Number.parseInt(applied.inviterId.trim(), 10)
        if (!Number.isNaN(n)) filters.inviter_id = n
      }
      if (applied.inviteeId.trim() !== '') {
        const n = Number.parseInt(applied.inviteeId.trim(), 10)
        if (!Number.isNaN(n)) filters.invitee_id = n
      }
      if (applied.reversed !== 'all') filters.reversed = applied.reversed === 'yes'
      if (applied.breach !== 'all') filters.breach = applied.breach === 'yes'

      const res = await adminListCommissionRecords(filters)
      if (res.success && res.data) {
        setRecords(res.data.items || [])
        setTotal(res.data.total || 0)
      } else {
        toast.error(res.message || t('admin-commission.error.list-failed', '列表加载失败'))
      }
    } catch (e) {
      toast.error(
        `${t('admin-commission.error.list-failed', '列表加载失败')}: ${(e as Error).message}`,
      )
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, applied, t])

  useEffect(() => {
    void fetchList()
  }, [fetchList])

  // 过滤器点"应用"：同步 input state + 重置到 page=1
  const applyFilters = useCallback(() => {
    setPage(1)
    setApplied({
      inviterId: inviterIdInput,
      inviteeId: inviteeIdInput,
      reversed: reversedFilter,
      breach: breachFilter,
    })
  }, [inviterIdInput, inviteeIdInput, reversedFilter, breachFilter])

  const resetFilters = useCallback(() => {
    setInviterIdInput('')
    setInviteeIdInput('')
    setReversedFilter('all')
    setBreachFilter('all')
    setApplied({ inviterId: '', inviteeId: '', reversed: 'all', breach: 'all' })
    setPage(1)
  }, [])

  const onReverseClick = useCallback((record: AdminCommissionRecord) => {
    setPending(record)
    setDialogOpen(true)
  }, [])

  const onConfirmReverse = useCallback(
    async (recordId: number, reason: string) => {
      setSubmitting(true)
      try {
        const res = await adminReverseCommissionRecord(recordId, reason)
        if (res.success && res.data) {
          // 乐观刷新：把本地这条 records 的 reversed 字段同步更新。
          // 不重发 GET，保留分页位置避免跳变。
          setRecords((prev) =>
            prev.map((r) => (r.id === recordId ? (res.data ?? r) : r)),
          )
          setTotal((prev) => prev) // total 不变（record 还在表里）
          toast.success(
            t('admin-commission.reverse.success', '撤销成功'),
          )
          setDialogOpen(false)
          setPending(null)
        } else {
          // controller 走 HTTP 409 时 axios 会抛 axios 错而不是这里 success=false。
          // 此分支命中：record 不存在 / 字段校验失败等。
          toast.error(res.message || t('admin-commission.reverse.failed', '撤销失败'))
        }
      } catch (e) {
        // axios 捕获 HTTP 4xx/5xx。
        // §20.7 controller 对二次撤销返 HTTP 409 → 这里被 catch。
        const err = e as { response?: { status?: number; data?: { message?: string } } }
        const status = err?.response?.status
        const msg = err?.response?.data?.message
        if (status === 409) {
          toast.error(t('admin-commission.reverse.conflict', '该返佣已被撤销'))
        } else {
          toast.error(`${t('admin-commission.reverse.failed', '撤销失败')}: ${msg ?? (e as Error).message}`)
        }
      } finally {
        setSubmitting(false)
      }
    },
    [t],
  )

  return (
    <div className='space-y-4'>
      <Card>
        <CardHeader>
          <CardTitle>
            {t('admin-commission.title', '返佣记录管理')}
          </CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          <FiltersBar
            inviterId={inviterIdInput}
            inviteeId={inviteeIdInput}
            reversed={reversedFilter}
            breach={breachFilter}
            onChange={(next) => {
              setInviterIdInput(next.inviterId)
              setInviteeIdInput(next.inviteeId)
              setReversedFilter(next.reversed)
              setBreachFilter(next.breach)
            }}
            onApply={applyFilters}
            onReset={resetFilters}
          />

          <AdminCommissionTable
            records={records}
            page={page}
            total={total}
            pageSize={pageSize}
            loading={loading}
            onPageChange={setPage}
            onReverse={onReverseClick}
          />
        </CardContent>
      </Card>

      <ReverseDialog
        record={pending}
        open={dialogOpen}
        onOpenChange={(o) => {
          setDialogOpen(o)
          if (!o) setPending(null)
        }}
        onConfirm={onConfirmReverse}
        busy={submitting}
      />
    </div>
  )
}