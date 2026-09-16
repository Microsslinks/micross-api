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
import { Database } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { Badge } from '@/components/ui/badge'
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { TitledCard } from '@/components/ui/titled-card'
import type { CommissionRecord } from '@/features/wallet/types'
import dayjs from '@/lib/dayjs'
import { formatQuota } from '@/lib/format'

interface CommissionRecordsTableProps {
  records: CommissionRecord[]
  page: number
  total: number
  pageSize: number
  loading: boolean
  onPageChange: (page: number) => void
}

/**
 * Commission records table — 任务文档 §五"流水面板"。
 *
 * 列：日期 / 被邀请人 / 模型 / 消费金额 / 返佣率 / 返佣额 / 状态
 *   - 模型字段后端暂未实现（task-10 commission_records 没存 model_name），
 *     暂时显示 "—" 占位，待后续 task 扩展。
 *   - breach 行红 badge "Breach" + tooltip "Margin capped"
 *   - 被邀请人列以 #ID 形式展示（commission_records 不返回 invitee 用户名映射；
 *     取用户名需新加 GET /api/user/username/:id 端点，留待后续 task）
 *   - 空数据：EmptyState "No commission records yet"
 *
 * 分页：ui/pagination 组件（hugeicons），page state 由 useCommission hook
 * 管，本组件只负责渲染 + 调 onPageChange。
 */
export function CommissionRecordsTable({
  records,
  page,
  total,
  pageSize,
  loading,
  onPageChange,
}: CommissionRecordsTableProps) {
  const { t } = useTranslation()

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const prevDisabled = page <= 1
  const nextDisabled = page >= totalPages

  // 短数字分页：当前页 + 邻居 + 头尾，超出用省略号
  const pageNumbers: (number | 'ellipsis')[] = []
  for (let i = 1; i <= totalPages; i++) {
    if (
      i === 1 ||
      i === totalPages ||
      i === page ||
      i === page - 1 ||
      i === page + 1
    ) {
      pageNumbers.push(i)
    } else if (pageNumbers[pageNumbers.length - 1] !== 'ellipsis') {
      pageNumbers.push('ellipsis')
    }
  }

  return (
    <TitledCard
      title={t('Commission records')}
      description={t(
        'Earn commission on your invitees\' API consumption. Withdrawals are not allowed.'
      )}
    >
      {loading ? (
        <div className='space-y-2'>
          {Array.from({ length: Math.min(pageSize, 8) }).map((_, i) => (
            <Skeleton key={i} className='h-8 w-full' />
          ))}
        </div>
      ) : records.length === 0 ? (
        <EmptyState
          icon={Database}
          title={t('No commission records yet')}
          description={t(
            'Commission records will appear here once your invitees consume API quota.'
          )}
          size='sm'
        />
      ) : (
        <>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Date')}</TableHead>
                <TableHead>{t('Invitee')}</TableHead>
                <TableHead>{t('Model')}</TableHead>
                <TableHead className='text-right'>{t('Gross')}</TableHead>
                <TableHead className='text-right'>{t('Rate')}</TableHead>
                <TableHead className='text-right'>{t('Amount')}</TableHead>
                <TableHead>{t('Status')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className='text-muted-foreground'>
                    {dayjs(r.settled_at * 1000).format('YYYY-MM-DD HH:mm')}
                  </TableCell>
                  <TableCell className='font-mono text-xs'>
                    #{r.invitee_id}
                  </TableCell>
                  <TableCell className='text-muted-foreground'>—</TableCell>
                  <TableCell className='text-right tabular-nums'>
                    {formatQuota(r.gross)}
                  </TableCell>
                  <TableCell className='text-right tabular-nums'>
                    {(parseFloat(r.rate) * 100).toFixed(2)}%
                  </TableCell>
                  <TableCell className='text-right font-semibold tabular-nums'>
                    {formatQuota(r.amount)}
                  </TableCell>
                  <TableCell>
                    {r.breach ? (
                      <Badge
                        variant='destructive'
                        title={t('Margin capped')}
                      >
                        {t('Breach')}
                      </Badge>
                    ) : (
                      <span className='text-muted-foreground text-xs'>
                        {t('Normal')}
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {totalPages > 1 && (
            <Pagination className='mt-4'>
              <PaginationContent>
                <PaginationItem>
                  <PaginationPrevious
                    onClick={(e) => {
                      e.preventDefault()
                      if (!prevDisabled) onPageChange(page - 1)
                    }}
                    aria-disabled={prevDisabled}
                    className={
                      prevDisabled ? 'pointer-events-none opacity-50' : ''
                    }
                  />
                </PaginationItem>
                {pageNumbers.map((p, i) =>
                  p === 'ellipsis' ? (
                    <PaginationItem key={`ell-${i}`}>
                      <PaginationEllipsis />
                    </PaginationItem>
                  ) : (
                    <PaginationItem key={p}>
                      <PaginationLink
                        isActive={p === page}
                        onClick={(e) => {
                          e.preventDefault()
                          if (p !== page) onPageChange(p)
                        }}
                      >
                        {p}
                      </PaginationLink>
                    </PaginationItem>
                  )
                )}
                <PaginationItem>
                  <PaginationNext
                    onClick={(e) => {
                      e.preventDefault()
                      if (!nextDisabled) onPageChange(page + 1)
                    }}
                    aria-disabled={nextDisabled}
                    className={
                      nextDisabled ? 'pointer-events-none opacity-50' : ''
                    }
                  />
                </PaginationItem>
              </PaginationContent>
            </Pagination>
          )}
        </>
      )}
    </TitledCard>
  )
}