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
import { Loader2, Users } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
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
import { getDiscountBindingSourceLabel } from '@/features/discounts/constants'
import { DISCOUNT_BINDING_SOURCE } from '@/features/discounts/types'
import { USER_STATUSES } from '@/features/users/constants'
import {
  formatQuota,
  parseQuotaFromDollars,
  quotaUnitsToDollars,
} from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import {
  getAgentLedger,
  getSelfAgentCustomers,
  issueSelfAgentCustomerQuota,
  setSelfAgentCustomerDiscount,
} from '../api'
import type { AgentCustomer, DealerPlanOption } from '../types'

const PAGE_SIZE = 20

interface Props {
  /** 货架：能给定下的价都在这里；平台还没给他可卖的方案时是空数组 */
  plans: DealerPlanOption[]
}

/**
 * 我的客户：挂在当前经销商名下的客户名单。
 *
 * 「谁是我的客户」不看客户号，只看归属：客户绑了他的号，就归到他名下。所以号被作废、
 * 被用满都不影响这份名单，人还是他的。
 *
 * 两件事在这里做：给某位客户定价（覆盖客户号带的价），以及从自己的余额里给他发额度。
 * 平台已经单独定过价的客户这里改不动——那是平台的决定，后端会拒，界面直接把按钮关掉。
 */
export function AgentCustomersPanel({ plans }: Props) {
  const { t } = useTranslation()
  const [customers, setCustomers] = useState<AgentCustomer[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [pricingTarget, setPricingTarget] = useState<AgentCustomer | null>(null)
  const [quotaTarget, setQuotaTarget] = useState<AgentCustomer | null>(null)

  const load = useCallback(async (targetPage: number) => {
    setLoading(true)
    try {
      const result = await getSelfAgentCustomers(targetPage, PAGE_SIZE)
      setCustomers(result.data?.items ?? [])
      setTotal(result.data?.total ?? 0)
      setPage(targetPage)
    } catch (error) {
      handleServerError(error)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load(1)
  }, [load])

  /** 改价/发额度之后只换那一行：整页重新拉一遍会让表格闪一下。 */
  const replaceRow = (row: AgentCustomer) => {
    setCustomers((previous) =>
      previous.map((item) => (item.user_id === row.user_id ? row : item))
    )
  }

  /** 没有方案或方案已停用时退回显示编号，不留空白。 */
  const priceLabel = (item: AgentCustomer) => {
    if (item.plan_id <= 0) return t('Official price')
    const name = item.plan_name || t('Plan #{{id}}', { id: item.plan_id })
    return item.plan_discount ? `${name} · ${item.plan_discount}` : name
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  if (loading) {
    return <LoadingState size='md' />
  }

  if (customers.length === 0) {
    return (
      <div className='space-y-3'>
        <div className='flex items-center gap-2'>
          <Users className='text-muted-foreground size-4' />
          <span className='text-sm font-medium'>{t('Customers')}</span>
        </div>
        <EmptyState
          icon={Users}
          title={t('No customers yet')}
          description={t(
            'Hand a customer code to a customer; they show up here once they register or bind it.'
          )}
          size='md'
        />
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      <div className='flex items-center gap-2'>
        <Users className='text-muted-foreground size-4' />
        <span className='text-sm font-medium'>{t('Customers')}</span>
        <span className='text-muted-foreground text-xs tabular-nums'>
          {total}
        </span>
      </div>

      <div className='overflow-hidden rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Customer')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead className='text-right'>
                {t('Current Balance')}
              </TableHead>
              <TableHead className='text-right'>{t('Used Quota')}</TableHead>
              <TableHead>{t('Current price')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {customers.map((item) => {
              const statusConfig =
                USER_STATUSES[item.status as keyof typeof USER_STATUSES]
              return (
                <TableRow key={item.user_id}>
                  <TableCell>
                    <div className='flex flex-col'>
                      <span className='font-medium'>
                        {item.display_name || item.username}
                      </span>
                      <span className='text-muted-foreground font-mono text-xs'>
                        #{item.user_id}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    {statusConfig ? (
                      <StatusBadge
                        label={t(statusConfig.labelKey)}
                        variant={statusConfig.variant}
                      />
                    ) : (
                      '-'
                    )}
                  </TableCell>
                  <TableCell className='text-right font-mono tabular-nums'>
                    {formatQuota(item.quota)}
                  </TableCell>
                  <TableCell className='text-right font-mono tabular-nums'>
                    {formatQuota(item.used_quota)}
                  </TableCell>
                  <TableCell>
                    <div className='flex flex-col'>
                      <span className='text-sm'>{priceLabel(item)}</span>
                      {item.binding_source && (
                        <span className='text-muted-foreground text-xs'>
                          {getDiscountBindingSourceLabel(
                            t,
                            item.binding_source
                          )}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className='text-right whitespace-nowrap'>
                    {item.binding_source === DISCOUNT_BINDING_SOURCE.MANUAL ? (
                      <span className='text-muted-foreground text-xs'>
                        {t('Priced by the platform')}
                      </span>
                    ) : (
                      <Button
                        variant='ghost'
                        size='sm'
                        className='h-7 px-2'
                        onClick={() => setPricingTarget(item)}
                      >
                        {t('Set price')}
                      </Button>
                    )}
                    <Button
                      variant='ghost'
                      size='sm'
                      className='h-7 px-2'
                      onClick={() => setQuotaTarget(item)}
                    >
                      {t('Issue quota')}
                    </Button>
                  </TableCell>
                </TableRow>
              )
            })}
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
            {t('Previous page')}
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
            {t('Next page')}
          </Button>
        </div>
      )}

      <PricingDialog
        customer={pricingTarget}
        plans={plans}
        onOpenChange={(open) => {
          if (!open) setPricingTarget(null)
        }}
        onSaved={(row) => {
          replaceRow(row)
          setPricingTarget(null)
        }}
      />
      <QuotaDialog
        customer={quotaTarget}
        onOpenChange={(open) => {
          if (!open) setQuotaTarget(null)
        }}
        onSaved={(row) => {
          replaceRow(row)
          setQuotaTarget(null)
        }}
      />
    </div>
  )
}

interface PricingDialogProps {
  /** null 表示关着：对话框的开关就是「有没有选中一行」 */
  customer: AgentCustomer | null
  plans: DealerPlanOption[]
  onOpenChange: (open: boolean) => void
  onSaved: (row: AgentCustomer) => void
}

/** 给一位客户定价：从自己的货架上挑一套方案，或撤掉自己上次定的价。 */
function PricingDialog({
  customer,
  plans,
  onOpenChange,
  onSaved,
}: PricingDialogProps) {
  const { t } = useTranslation()
  const [planId, setPlanId] = useState('')
  const [saving, setSaving] = useState(false)

  // 打开时回到他此刻生效的价：如果那个价本来就是经销商自己定的，选中它；
  // 否则留空，逼他自己挑一套——不预选任何方案，避免手一滑把别人的价顶掉。
  useEffect(() => {
    if (!customer) return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setPlanId(customer.priced_by_me ? String(customer.plan_id) : '')
  }, [customer])

  const name = customer?.display_name || customer?.username || ''
  const canSave = customer !== null && planId !== '' && !saving

  const handleSave = async () => {
    if (!customer || planId === '') return
    setSaving(true)
    try {
      const result = await setSelfAgentCustomerDiscount(
        customer.user_id,
        Number.parseInt(planId, 10)
      )
      if (result.success && result.data) {
        toast.success(t('Price updated'))
        onSaved(result.data)
        return
      }
      toast.error(result.message || t('Failed to update the price'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog
      open={customer !== null}
      onOpenChange={onOpenChange}
      title={t('Set a price for {{name}}', { name })}
      description={t(
        'Your price replaces the one on the customer code, and the customer is billed by it.'
      )}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleSave} disabled={!canSave}>
            {saving && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Save')}
          </Button>
        </>
      }
    >
      <div className='space-y-2'>
        <Label>{t('Discount plan')}</Label>
        <Select
          items={[
            ...plans.map((plan) => ({
              value: String(plan.id),
              label: `${plan.name} · ${plan.base_discount}`,
            })),
            ...(customer?.priced_by_me
              ? [{ value: '0', label: t('Remove my price') }]
              : []),
          ]}
          value={planId}
          onValueChange={(value) => setPlanId(value ?? '')}
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder={t('Choose a plan')} />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              {plans.map((plan) => (
                <SelectItem key={plan.id} value={String(plan.id)}>
                  {`${plan.name} · ${plan.base_discount}`}
                </SelectItem>
              ))}
              {customer?.priced_by_me && (
                <SelectItem value='0'>{t('Remove my price')}</SelectItem>
              )}
            </SelectGroup>
          </SelectContent>
        </Select>
        <p className='text-muted-foreground text-xs'>
          {t(
            'Only the plans on your shelf are listed. Choosing none of them keeps the current price.'
          )}
        </p>
      </div>
    </Dialog>
  )
}

interface QuotaDialogProps {
  customer: AgentCustomer | null
  onOpenChange: (open: boolean) => void
  onSaved: (row: AgentCustomer) => void
}

/** 给一位客户发额度：钱从经销商自己的余额里出，不是平台补贴。 */
function QuotaDialog({ customer, onOpenChange, onSaved }: QuotaDialogProps) {
  const { t } = useTranslation()
  const currencyConfig = useSystemConfigStore((state) => state.config.currency)
  const [amount, setAmount] = useState('')
  const [available, setAvailable] = useState<number | null>(null)
  const [sending, setSending] = useState(false)

  // 一次至少发一个显示单位（quotaPerUnit 个额度单位），否则发过去是个零头。
  const minimumQuota = Math.ceil(
    currencyConfig.quotaPerUnit > 0
      ? currencyConfig.quotaPerUnit
      : DEFAULT_CURRENCY_CONFIG.quotaPerUnit
  )
  const minimumAmount = quotaUnitsToDollars(minimumQuota)

  // 打开时现拉一次自己的余额，而不是沿用页面上的旧数：连着发几笔之后那个数就不准了。
  useEffect(() => {
    if (!customer) return
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setAmount(String(minimumAmount))
    void getAgentLedger()
      .then((result) =>
        setAvailable(result.success ? (result.data?.quota ?? 0) : 0)
      )
      .catch(() => setAvailable(null))
  }, [customer, minimumAmount])

  const name = customer?.display_name || customer?.username || ''
  const quota = parseQuotaFromDollars(Number.parseFloat(amount) || 0)
  const exceeding = available !== null && quota > available
  const canSend =
    customer !== null && quota >= minimumQuota && !exceeding && !sending

  const handleSend = async () => {
    if (!customer || !canSend) return
    setSending(true)
    try {
      const result = await issueSelfAgentCustomerQuota(customer.user_id, quota)
      if (result.success && result.data) {
        toast.success(t('Quota issued'))
        onSaved(result.data)
        return
      }
      toast.error(result.message || t('Failed to issue quota'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setSending(false)
    }
  }

  return (
    <Dialog
      open={customer !== null}
      onOpenChange={onOpenChange}
      title={t('Issue quota to {{name}}', { name })}
      description={t('The quota comes out of your own balance.')}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={sending}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleSend} disabled={!canSend}>
            {sending && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Issue quota')}
          </Button>
        </>
      }
    >
      <div className='grid grid-cols-2 gap-3 py-1'>
        <div className='space-y-1'>
          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('Your balance')}
          </Label>
          <div className='font-mono text-lg font-semibold tabular-nums'>
            {available === null ? '-' : formatQuota(available)}
          </div>
        </div>
        <div className='space-y-1'>
          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('Current Balance')}
          </Label>
          <div className='font-mono text-lg font-semibold tabular-nums'>
            {formatQuota(customer?.quota ?? 0)}
          </div>
        </div>
      </div>

      <div className='space-y-2'>
        <Label htmlFor='customer-quota-amount'>{t('Amount')}</Label>
        <Input
          id='customer-quota-amount'
          type='number'
          value={amount}
          min={minimumAmount}
          step={minimumAmount}
          className='font-mono text-lg'
          onChange={(event) => setAmount(event.target.value)}
        />
        {exceeding ? (
          <p className='text-destructive text-xs'>
            {t('More than your own balance.')}
          </p>
        ) : (
          <p className='text-muted-foreground text-xs'>
            {t('Minimum: {{amount}}', {
              amount: formatQuota(minimumQuota),
            })}
          </p>
        )}
      </div>
    </Dialog>
  )
}
