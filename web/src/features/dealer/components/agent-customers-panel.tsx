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
import type { TFunction } from 'i18next'
import { Loader2, Plus, Users } from 'lucide-react'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getDiscountBindingSourceLabel } from '@/features/discounts/constants'
import { formatRatioText } from '@/features/discounts/lib/format'
import { parseRatioText } from '@/features/discounts/lib/format'
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
} from '../api'
import type { AgentCustomer, AgentCustomerBinding } from '../types'
import { CustomerRowActions } from './customer-row-actions'
import { RegisterCustomerDialog } from './register-customer-dialog'

const PAGE_SIZE = 20

/** 一条绑定的显示名：方案名缺失（方案被删）时退回编号，不留空白。 */
function bindingLabel(t: TFunction, binding: AgentCustomerBinding) {
  const name = binding.plan_name || t('Plan #{{id}}', { id: binding.plan_id })
  const discount = formatRatioText(binding.plan_discount)
  return `${name} · ${discount}`
}

/**
 * 我的客户：挂在当前经销商名下的客户名单。
 *
 * 「谁是我的客户」不看客户号，只看归属：客户绑了他的号，就归到他名下。所以号被作废、
 * 被用满都不影响这份名单，人还是他的。
 *
 * 这里只做一件事：从经销商自己的余额里给客户发额度。价格不在这里定——折扣方案由
 * 平台管理员统一挂载（业务口径 2026-09-14），经销商在「查看价格」里能看到客户此刻挂着
 * 的每一套方案和它的来路，但改不了。
 */
export function AgentCustomersPanel() {
  const { t } = useTranslation()
  const [customers, setCustomers] = useState<AgentCustomer[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [quotaTarget, setQuotaTarget] = useState<AgentCustomer | null>(null)
  const [detailTarget, setDetailTarget] = useState<AgentCustomer | null>(null)
  const [registerOpen, setRegisterOpen] = useState(false)

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

  /**
   * 一个客户可以挂多套方案，所以这一列只有一个价可写的情况才把价写出来：
   * 没挂方案 = 官方标价，挂一套 = 方案名 · 折扣，挂多套 = 只说套数（明细在「查看价格」里）。
   */
  const priceLabel = (item: AgentCustomer) => {
    const bindings = item.bindings ?? []
    if (bindings.length === 0) return t('Official price')
    if (bindings.length === 1) return bindingLabel(t, bindings[0])
    return t('{{plans}} plans', { plans: bindings.length })
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
          <span className='text-muted-foreground text-xs tabular-nums'>
            {total}
          </span>
          <div className='ml-auto'>
            <Button
              size='sm'
              onClick={() => setRegisterOpen(true)}
            >
              <Plus className='size-4' />
              {t('dealer.registerCustomer')}
            </Button>
          </div>
        </div>
        <EmptyState
          icon={Users}
          title={t('No customers yet')}
          description={t(
            'Hand a customer code to a customer, or register one for them; they show up here once bound.'
          )}
          size='md'
        />
        <RegisterCustomerDialog
          open={registerOpen}
          onOpenChange={setRegisterOpen}
          onSuccess={(created) => {
            // 直接把新行插到列表顶部，省一次刷新；同时刷新台账金额等
            setCustomers([created])
            setTotal((value) => value + 1)
            void load(1)
          }}
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
        <div className='ml-auto'>
          <Button
            size='sm'
            onClick={() => setRegisterOpen(true)}
          >
            <Plus className='size-4' />
            {t('dealer.registerCustomer')}
          </Button>
        </div>
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
              // 来源标记只在"就一套"时写：多套方案的来源各不相同，挑一条显示反而是误导。
              const soleSource =
                item.bindings?.length === 1 ? item.bindings[0].source : ''
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
                      {/* 平台手工定价的来源标记（「手动」）对经销商没有信息量，
                          他要的答案挪进了「查看价格」对话框，这里不再显示。 */}
                      {soleSource &&
                        soleSource !== DISCOUNT_BINDING_SOURCE.MANUAL && (
                          <span className='text-muted-foreground text-xs'>
                            {getDiscountBindingSourceLabel(t, soleSource)}
                          </span>
                        )}
                    </div>
                  </TableCell>
                  <TableCell className='text-right whitespace-nowrap'>
                    <Button
                      variant='ghost'
                      size='sm'
                      className='h-7 px-2'
                      onClick={() => setDetailTarget(item)}
                    >
                      {t('View pricing')}
                    </Button>
                    <Button
                      variant='ghost'
                      size='sm'
                      className='h-7 px-2'
                      onClick={() => setQuotaTarget(item)}
                    >
                      {t('Issue quota')}
                    </Button>
                    <CustomerRowActions
                      customer={item}
                      onRefresh={() => void load(page)}
                    />
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
      <PricingDetailDialog
        customer={detailTarget}
        onOpenChange={(open) => {
          if (!open) setDetailTarget(null)
        }}
      />
      <RegisterCustomerDialog
        open={registerOpen}
        onOpenChange={setRegisterOpen}
        onSuccess={(created) => {
          // 把新行插到列表顶部让经销商立即看到，再异步刷一次拿服务端最终态
          setCustomers((previous) => [created, ...previous])
          setTotal((value) => value + 1)
          void load(1)
        }}
      />
    </div>
  )
}

interface PricingDetailDialogProps {
  /** null 表示关着：对话框的开关就是「有没有选中一行」 */
  customer: AgentCustomer | null
  onOpenChange: (open: boolean) => void
}

/**
 * 价格详情：这位客户此刻挂着的每一套方案、每套是谁定的。
 *
 * 一个客户可以同时挂多套（平台后台逐条挂），所以这里必须全部列出来——只列一套
 * 会让经销商以为客户只有那一份价。多套之间由计价规则决定哪个模型用哪套，
 * 所以底下那句话不能写成"第一套就是生效的价"。
 *
 * 价格由平台管理员统一挂载，经销商在这里只有看的份。
 */
function PricingDetailDialog({
  customer,
  onOpenChange,
}: PricingDetailDialogProps) {
  const { t } = useTranslation()
  const name = customer?.display_name || customer?.username || ''
  const bindings = customer?.bindings ?? []

  return (
    <Dialog
      open={customer !== null}
      onOpenChange={onOpenChange}
      title={t('Pricing for {{name}}', { name })}
      description={t(
        'Pricing is managed by the platform. Contact the platform administrator to adjust it.'
      )}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      contentHeight='auto'
      bodyClassName='space-y-4'
    >
      {bindings.length === 0 ? (
        <p className='text-sm'>
          {t('No discount plan is bound to this customer.')}
        </p>
      ) : (
        <div className='overflow-hidden rounded-md border'>
          <div className='bg-muted/40 flex items-center justify-between gap-3 px-3 py-1.5'>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Discount plan')}
            </Label>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Base discount')}
            </Label>
          </div>
          <div className='divide-y'>
            {bindings.map((binding) => (
              <div
                key={binding.binding_id}
                className='flex items-center justify-between gap-3 px-3 py-2'
              >
                <div className='flex min-w-0 flex-col'>
                  <span className='truncate text-sm font-medium'>
                    {binding.plan_name ||
                      t('Plan #{{id}}', { id: binding.plan_id })}
                  </span>
                  <span className='text-muted-foreground text-xs'>
                    {getDiscountBindingSourceLabel(t, binding.source)}
                  </span>
                </div>
                <span className='text-sm font-medium tabular-nums'>
                  {formatRatioText(binding.plan_discount)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
      <p className='text-muted-foreground text-xs'>
        {t('Individual models in the plan may carry their own discounts.')}
      </p>
      {bindings.length > 1 && (
        <p className='text-muted-foreground text-xs'>
          {t(
            'When several plans are bound, the pricing rules pick one for each model.'
          )}
        </p>
      )}
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
  // 折算预览：客户主方案的 TopupConversionRate（task-09）。空字符串走 1.0 兜底，
  // 与后端 resolveAgentTopupCost 同口径。预览按 ceil(quota × rate) 估算，与后端实扣一致。
  const topupRate = customer ? (parseRatioText(customer.topup_conversion_rate) ?? 1) : 1
  const agentCostPreview = Math.ceil(quota * topupRate)
  const showTopupPreview = customer !== null && quota > 0 && topupRate < 1
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

      {/*
       * task-09：折算预览。客户主方案配了 < 1 的比例时显示「面值 X 实扣 Y」+ 比例百分比。
       * 后端仍按自己的 TopupConversionRate 实扣（不依赖前端传的 agentCost），
       * 这里只是给经销商看个估算，让他发之前心里有数。
       */}
      {showTopupPreview && (
        <div className='text-muted-foreground space-y-1 text-xs'>
          <p>
            {t('Face value: {{face}}', {
              face: formatQuota(quota),
            })}
          </p>
          <p>
            {t('Actually deducted: {{actual}}', {
              actual: formatQuota(agentCostPreview),
            })}
            <span className='ml-1'>
              ({formatRatioText(customer?.topup_conversion_rate)})
            </span>
          </p>
        </div>
      )}
    </Dialog>
  )
}
