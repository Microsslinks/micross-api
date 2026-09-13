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
import { Ticket } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { CopyButton } from '@/components/copy-button'
import { EmptyState } from '@/components/empty-state'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestamp } from '@/lib/format'

import type {
  CustomerCode,
  CustomerCodeIssuePayload,
  DealerPlanOption,
} from '../types'

/** 客户号状态：与后端 customer_codes.status 对齐。 */
const CUSTOMER_CODE_STATUS = {
  REVOKED: 0,
  ACTIVE: 1,
} as const

/** 一次最多签多少个，与后端 CustomerCodeMaxBatch 是同一条线。 */
const MAX_BATCH = 50
const PAGE_SIZE = 20

/**
 * 给客户号面板准备的三个动作。
 *
 * 同一个面板要服务两种视角：平台替某位经销商签号（管理员在用户列表里点开），
 * 和经销商自己签号（他在自己的台账页里）。两种视角的业务完全一样，
 * 差别只在请求打到哪个地址，所以由调用方把这三个动作组装好传进来。
 */
export interface CustomerCodesSource {
  list: (page: number) => Promise<{ items: CustomerCode[]; total: number }>
  create: (
    payload: CustomerCodeIssuePayload
  ) => Promise<{ success: boolean; message?: string; items?: CustomerCode[] }>
  revoke: (codeId: number) => Promise<{ success: boolean; message?: string }>
  /** 可绑的折扣方案；空数组时下拉里只剩「不绑折扣」 */
  plans: DealerPlanOption[]
}

interface Props {
  source: CustomerCodesSource
}

/**
 * 客户号面板：签号、看号、作废号。
 *
 * 客户号是经销商把客户拉进来的凭证——客户拿号绑一下，归属和折扣就一起落到他账号上。
 * 这里只管发和收：谁绑了、绑完按什么价算，分别在「我的客户」和计价那两条链上。
 */
export function CustomerCodesPanel({ source }: Props) {
  const { t } = useTranslation()
  const [codes, setCodes] = useState<CustomerCode[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [pendingRevoke, setPendingRevoke] = useState<CustomerCode | null>(null)

  const [count, setCount] = useState('1')
  const [planId, setPlanId] = useState('0')
  const [maxUses, setMaxUses] = useState('1')
  const [expiredAt, setExpiredAt] = useState('')
  const [remark, setRemark] = useState('')

  const load = useCallback(
    async (targetPage: number) => {
      setLoading(true)
      try {
        const result = await source.list(targetPage)
        setCodes(result.items)
        setTotal(result.total)
        setPage(targetPage)
      } finally {
        setLoading(false)
      }
    },
    [source]
  )

  useEffect(() => {
    void load(1)
  }, [load])

  const countValue = Number.parseInt(count, 10)
  const countValid =
    Number.isFinite(countValue) && countValue >= 1 && countValue <= MAX_BATCH
  const maxUsesValue = Number.parseInt(maxUses, 10)
  const maxUsesValid = Number.isFinite(maxUsesValue) && maxUsesValue >= 0

  /** 方案名以货架为准；方案被停用后货架上没有它，退回显示编号，不留空白。 */
  const planLabel = (id: number) => {
    if (id <= 0) return t('No discount plan')
    const plan = source.plans.find((item) => item.id === id)
    return plan
      ? `${plan.name} · ${plan.base_discount}`
      : t('Plan #{{id}}', { id })
  }

  const handleIssue = async () => {
    if (!countValid || !maxUsesValid) return
    // 日期填的是"用到哪天"，所以按当天 23:59:59 截止，不给客户一个当天早上就失效的号。
    const expiresAt = expiredAt
      ? Math.floor(new Date(`${expiredAt}T23:59:59`).getTime() / 1000)
      : 0
    setSubmitting(true)
    try {
      const result = await source.create({
        count: countValue,
        plan_id: Number.parseInt(planId, 10) || 0,
        max_uses: maxUsesValue,
        expired_at: expiresAt,
        remark,
      })
      if (result.success) {
        toast.success(
          t('{{count}} customer code(s) issued', { count: countValue })
        )
        setRemark('')
        await load(1)
      } else {
        toast.error(result.message || t('Failed to issue customer codes'))
      }
    } catch {
      toast.error(t('Failed to issue customer codes'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleRevoke = async () => {
    if (!pendingRevoke) return
    try {
      const result = await source.revoke(pendingRevoke.id)
      if (result.success) {
        toast.success(t('Customer code revoked'))
        await load(page)
      } else {
        toast.error(result.message || t('Failed to revoke the customer code'))
      }
    } catch {
      toast.error(t('Failed to revoke the customer code'))
    } finally {
      setPendingRevoke(null)
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className='space-y-4'>
      <div className='space-y-3 rounded-lg border p-3'>
        <div className='flex items-center gap-2'>
          <Ticket className='text-muted-foreground size-4' />
          <span className='text-sm font-medium'>{t('Customer Codes')}</span>
        </div>

        <div className='grid grid-cols-2 gap-3 sm:grid-cols-3'>
          <div className='space-y-2'>
            <Label htmlFor='code-count'>{t('Quantity')}</Label>
            <Input
              id='code-count'
              value={count}
              inputMode='numeric'
              onChange={(event) => setCount(event.target.value)}
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='code-max-uses'>{t('Usage limit')}</Label>
            <Input
              id='code-max-uses'
              value={maxUses}
              inputMode='numeric'
              onChange={(event) => setMaxUses(event.target.value)}
            />
            <p className='text-muted-foreground text-xs'>
              {t('0 means unlimited')}
            </p>
          </div>

          <div className='space-y-2'>
            <Label htmlFor='code-expired-at'>{t('Valid until')}</Label>
            <Input
              id='code-expired-at'
              type='date'
              value={expiredAt}
              onChange={(event) => setExpiredAt(event.target.value)}
            />
            <p className='text-muted-foreground text-xs'>
              {t('Leave empty to never expire')}
            </p>
          </div>
        </div>

        <div className='space-y-2'>
          <Label>{t('Discount plan')}</Label>
          <Select
            items={[
              { value: '0', label: t('No discount plan') },
              ...source.plans.map((plan) => ({
                value: String(plan.id),
                label: `${plan.name} · ${plan.base_discount}`,
              })),
            ]}
            value={planId}
            onValueChange={(value) => setPlanId(value ?? '0')}
          >
            <SelectTrigger className='w-full'>
              <SelectValue placeholder={t('No discount plan')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                <SelectItem value='0'>{t('No discount plan')}</SelectItem>
                {source.plans.map((plan) => (
                  <SelectItem key={plan.id} value={String(plan.id)}>
                    {`${plan.name} · ${plan.base_discount}`}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <p className='text-muted-foreground text-xs'>
            {t(
              'The customer is priced by this plan once the code is bound. Choose none to only record the dealer.'
            )}
          </p>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='code-remark'>{t('Remark')}</Label>
          <Input
            id='code-remark'
            value={remark}
            maxLength={255}
            onChange={(event) => setRemark(event.target.value)}
          />
        </div>

        <Button
          onClick={handleIssue}
          disabled={submitting || !countValid || !maxUsesValid}
        >
          {submitting ? t('Issuing') : t('Issue')}
        </Button>
        {!countValid && (
          <p className='text-muted-foreground text-xs'>
            {t('Quantity must be between 1 and {{max}}.', { max: MAX_BATCH })}
          </p>
        )}
      </div>

      {loading && <LoadingState size='md' />}

      {!loading && codes.length === 0 && (
        <EmptyState
          icon={Ticket}
          title={t('No customer codes yet')}
          description={t('Issue a code and hand it to a customer to sign up.')}
          size='md'
        />
      )}

      {!loading && codes.length > 0 && (
        <div className='space-y-2'>
          <div className='overflow-hidden rounded-lg border'>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Customer code')}</TableHead>
                  <TableHead>{t('Discount plan')}</TableHead>
                  <TableHead className='text-right'>{t('Used')}</TableHead>
                  <TableHead>{t('Valid until')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Remark')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {codes.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <div className='flex items-center gap-1.5'>
                        <span className='font-mono text-xs font-medium'>
                          {item.code}
                        </span>
                        <CopyButton value={item.code} aria-label={t('Copy')} />
                      </div>
                    </TableCell>
                    <TableCell className='text-sm'>
                      {planLabel(item.plan_id)}
                    </TableCell>
                    <TableCell className='text-right font-mono text-sm tabular-nums'>
                      {item.max_uses > 0
                        ? `${item.used_count} / ${item.max_uses}`
                        : item.used_count}
                    </TableCell>
                    <TableCell className='text-muted-foreground text-sm'>
                      {item.expired_at > 0
                        ? formatTimestamp(item.expired_at)
                        : t('Never expires')}
                    </TableCell>
                    <TableCell>
                      {item.status === CUSTOMER_CODE_STATUS.ACTIVE ? (
                        <StatusBadge label={t('Active')} variant='success' />
                      ) : (
                        <StatusBadge label={t('Revoked')} variant='neutral' />
                      )}
                    </TableCell>
                    <TableCell className='text-muted-foreground text-sm'>
                      {item.remark || '-'}
                    </TableCell>
                    <TableCell className='text-right'>
                      <Button
                        variant='ghost'
                        size='sm'
                        className='text-destructive hover:text-destructive h-7 px-2'
                        disabled={item.status !== CUSTOMER_CODE_STATUS.ACTIVE}
                        onClick={() => setPendingRevoke(item)}
                      >
                        {t('Revoke')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {totalPages > 1 && (
            <div className='flex items-center justify-end gap-2'>
              <Button
                variant='outline'
                size='sm'
                disabled={page <= 1}
                onClick={() => void load(page - 1)}
              >
                {t('Previous')}
              </Button>
              <span className='text-muted-foreground text-xs'>
                {t('Page {{page}} of {{total}}', { page, total: totalPages })}
              </span>
              <Button
                variant='outline'
                size='sm'
                disabled={page >= totalPages}
                onClick={() => void load(page + 1)}
              >
                {t('Next')}
              </Button>
            </div>
          )}
        </div>
      )}

      <ConfirmDialog
        open={pendingRevoke !== null}
        onOpenChange={(next) => {
          if (!next) setPendingRevoke(null)
        }}
        title={t('Revoke')}
        desc={t(
          'Revoke this customer code? Customers who already bound it keep what they got; the code itself stops working.'
        )}
        confirmText={t('Revoke')}
        handleConfirm={handleRevoke}
      />
    </div>
  )
}
