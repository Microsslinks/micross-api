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

import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
import dayjs from '@/lib/dayjs'
import { formatQuota } from '@/lib/format'

import type { AdminCommissionRecord } from '../types'

interface AdminCommissionTableProps {
  records: AdminCommissionRecord[]
  page: number
  total: number
  pageSize: number
  loading: boolean
  onPageChange: (page: number) => void
  onReverse: (record: AdminCommissionRecord) => void
}

// ============================================================================
// Admin Commission Records Table (task-20 §20.8)
//
// 列：record_id / inviter_id / invitee_id / gross / rate / amount / breach /
// reversed / reversed_at / reversed_by / reverse_reason / 操作(撤销按钮)
//
// 与 user 端 earnings/commission-records-table.tsx 的区别：
//   1. 多展示 reversed / reversed_at / reversed_by / reverse_reason 4 列
//      (task-20 §20.6 加的字段)
//   2. 多了"撤销"按钮 —— 已撤销的记录按钮 disabled + 改文案为"已撤销"
//   3. breach 单独一列显示（user 端是 badge 形式，这里以冗余 badge 让 admin 看清）
// ============================================================================
export function AdminCommissionTable({
  records,
  page,
  total,
  pageSize,
  loading,
  onPageChange,
  onReverse,
}: AdminCommissionTableProps) {
  const { t } = useTranslation()

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const prevDisabled = page <= 1
  const nextDisabled = page >= totalPages

  // 与 user 端同一套短数字分页算法：当前页 + 邻居 + 头尾
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
    } else if (pageNumbers.at(-1) !== 'ellipsis') {
      pageNumbers.push('ellipsis')
    }
  }

  return (
    <div className='space-y-4'>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className='w-16'>#</TableHead>
            <TableHead>inviter</TableHead>
            <TableHead>invitee</TableHead>
            <TableHead className='text-right'>gross</TableHead>
            <TableHead>rate</TableHead>
            <TableHead className='text-right'>amount</TableHead>
            <TableHead className='text-right'>margin</TableHead>
            <TableHead>breach</TableHead>
            <TableHead>reversed</TableHead>
            <TableHead>reason</TableHead>
            <TableHead>settled_at</TableHead>
            <TableHead className='w-24'>{t('common.action', '操作')}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading && (
            <>
              {Array.from({ length: 5 }).map((_, i) => (
                <TableRow key={`skeleton-row-${i.toString()}`}>
                  <TableCell colSpan={12}>
                    <Skeleton className='h-6 w-full' />
                  </TableCell>
                </TableRow>
              ))}
            </>
          )}
          {!loading && records.length === 0 && (
            <TableRow>
              <TableCell colSpan={12}>
                <EmptyState
                  title={t('admin-commission.empty', '没有匹配的返佣记录')}
                />
              </TableCell>
            </TableRow>
          )}
          {!loading &&
              records.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className='font-mono'>#{r.id}</TableCell>
                  <TableCell className='font-mono'>#{r.inviter_id}</TableCell>
                  <TableCell className='font-mono'>#{r.invitee_id}</TableCell>
                  <TableCell className='text-right font-mono'>
                    {formatQuota(r.gross)}
                  </TableCell>
                  <TableCell className='font-mono'>{r.rate}</TableCell>
                  <TableCell className='text-right font-mono'>
                    {formatQuota(r.amount)}
                  </TableCell>
                  <TableCell className='text-right font-mono'>
                    {formatQuota(r.margin)}
                  </TableCell>
                  <TableCell>
                    {r.breach ? (
                      <Badge variant='destructive'>breach</Badge>
                    ) : (
                      <Badge variant='outline'>ok</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {r.reversed ? (
                      <Badge variant='secondary'>
                        {t('admin-commission.reversed', '已撤销')}
                      </Badge>
                    ) : (
                      <Badge variant='outline'>
                        {t('admin-commission.active', '正常')}
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className='max-w-[14rem] truncate text-xs text-muted-foreground'>
                    {r.reverse_reason || '—'}
                  </TableCell>
                  <TableCell className='text-xs'>
                    {dayjs.unix(r.settled_at).format('YYYY-MM-DD HH:mm')}
                  </TableCell>
                  <TableCell>
                    {r.reversed ? (
                      <Button variant='outline' size='sm' disabled>
                        {t('admin-commission.reversed', '已撤销')}
                      </Button>
                    ) : (
                      <Button
                        variant='destructive'
                        size='sm'
                        onClick={() => onReverse(r)}
                      >
                        {t('admin-commission.reverse', '撤销')}
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
        </TableBody>
      </Table>

      <Pagination>
        <PaginationContent>
          <PaginationItem>
            <PaginationPrevious
              onClick={() => onPageChange(Math.max(1, page - 1))}
              className={
                prevDisabled ? 'pointer-events-none opacity-50' : 'cursor-pointer'
                }
            />
          </PaginationItem>
          {pageNumbers.map((p) =>
            p === 'ellipsis' ? (
              <PaginationItem key={`ellipsis-${p}`}>
                <PaginationEllipsis />
              </PaginationItem>
            ) : (
              <PaginationItem key={p}>
                <PaginationLink
                  isActive={p === page}
                  onClick={() => onPageChange(p)}
                  className='cursor-pointer'
                >
                  {p}
                </PaginationLink>
              </PaginationItem>
            ),
          )}
          <PaginationItem>
            <PaginationNext
              onClick={() => onPageChange(Math.min(totalPages, page + 1))}
              className={
                nextDisabled ? 'pointer-events-none opacity-50' : 'cursor-pointer'
                }
            />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    </div>
  )
}