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
import { Plus, Ticket } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge, type StatusVariant } from '@/components/status-badge'
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
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatTimestamp } from '@/lib/format'

import { buildCustomerInviteLink } from '../lib/invite'
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

/** 一张号在界面上该被看成什么。由状态和用量一起算，见 resolveCodeState。 */
type CodeState = 'available' | 'used' | 'revoked' | 'expired'

/** 列表看哪一档：还能发出去的号，还是连历史一起翻。 */
type CodeScope = 'usable' | 'all'

const CODE_STATE_LABEL: Record<CodeState, string> = {
  available: 'Available',
  used: 'Used',
  revoked: 'Revoked',
  expired: 'Expired',
}

const CODE_STATE_VARIANT: Record<CodeState, StatusVariant> = {
  available: 'success',
  used: 'info',
  revoked: 'neutral',
  expired: 'warning',
}

/**
 * 一张号在界面上该显示成什么。
 *
 * 判断顺序是有讲究的：先看有没有被作废（那是经销商主动做的动作），再看有没有被用掉
 * （一张号只拉一位客户，用掉就是它的归宿），最后才轮到过期。已经被用掉的号之后过了
 * 有效期，它的身份仍然是"某位客户的来路"，不该被显示成一张过期作废的号。
 */
function resolveCodeState(code: CustomerCode, now: number): CodeState {
  if (code.status !== CUSTOMER_CODE_STATUS.ACTIVE) return 'revoked'
  if (code.max_uses > 0 && code.used_count >= code.max_uses) return 'used'
  if (code.expired_at > 0 && code.expired_at <= now) return 'expired'
  return 'available'
}

/**
 * 给客户号面板准备的三个动作。
 *
 * 同一个面板要服务两种视角：平台替某位经销商签号（管理员在用户列表里点开），
 * 和经销商自己签号（他在「我的客户」页里）。两种视角的业务完全一样，
 * 差别只在请求打到哪个地址，所以由调用方把这三个动作组装好传进来。
 */
export interface CustomerCodesSource {
  /** `onlyUsable` 为真时只要还能发出去的号，历史号码不带。 */
  list: (
    page: number,
    onlyUsable: boolean
  ) => Promise<{ items: CustomerCode[]; total: number }>
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
 * 一张号只拉一位客户，用完即废，所以这一块首先要回答的是"手上还有哪些号能发出去"：
 * 默认只列还能用的号，用完的、作废的切到「全部」去翻——数据都在库里，只是不占屏幕。
 *
 * 谁绑了、绑完按什么价算，分别在下面那份客户名单和计价那两条链上；这里只管发和收。
 */
export function CustomerCodesPanel({ source }: Props) {
  const { t } = useTranslation()
  const [codes, setCodes] = useState<CustomerCode[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [scope, setScope] = useState<CodeScope>('usable')
  const [loading, setLoading] = useState(true)
  const [issueOpen, setIssueOpen] = useState(false)
  const [pendingRevoke, setPendingRevoke] = useState<CustomerCode | null>(null)

  const load = useCallback(
    async (targetPage: number, targetScope: CodeScope) => {
      setLoading(true)
      try {
        const result = await source.list(targetPage, targetScope === 'usable')
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
    // 换档位时回到第一页：两档的总数不是一回事，停在第 3 页多半是空页。
    void load(1, scope)
  }, [load, scope])

  /** 方案名以货架为准；方案被停用后货架上没有它，退回显示编号，不留空白。 */
  const planLabel = (id: number) => {
    if (id <= 0) return t('No discount plan')
    const plan = source.plans.find((item) => item.id === id)
    return plan
      ? `${plan.name} · ${plan.base_discount}`
      : t('Plan #{{id}}', { id })
  }

  const handleRevoke = async () => {
    if (!pendingRevoke) return
    try {
      const result = await source.revoke(pendingRevoke.id)
      if (result.success) {
        toast.success(t('Customer code revoked'))
        // 回第一页：作废掉的号会从「还能用」这一档里消失，停在原页可能就空了。
        await load(1, scope)
      } else {
        toast.error(result.message || t('Failed to revoke the customer code'))
      }
    } catch {
      toast.error(t('Failed to revoke the customer code'))
    } finally {
      setPendingRevoke(null)
    }
  }

  const now = Math.floor(Date.now() / 1000)
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className='space-y-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <div className='flex items-center gap-2'>
          <Ticket className='text-muted-foreground size-4' />
          <span className='text-sm font-medium'>{t('Customer Codes')}</span>
        </div>

        <div className='flex items-center gap-2'>
          <div className='flex items-center gap-0.5 rounded-lg border p-0.5'>
            <Button
              size='sm'
              className='h-7'
              variant={scope === 'usable' ? 'default' : 'ghost'}
              onClick={() => setScope('usable')}
            >
              {t('Available')}
            </Button>
            <Button
              size='sm'
              className='h-7'
              variant={scope === 'all' ? 'default' : 'ghost'}
              onClick={() => setScope('all')}
            >
              {t('All')}
            </Button>
          </div>

          <Button size='sm' onClick={() => setIssueOpen(true)}>
            <Plus className='size-4' />
            {t('Issue a customer code')}
          </Button>
        </div>
      </div>

      {loading && <LoadingState size='md' />}

      {!loading && codes.length === 0 && (
        <EmptyState
          icon={Ticket}
          title={
            scope === 'usable'
              ? t('No code left to hand out')
              : t('No customer codes yet')
          }
          description={
            scope === 'usable'
              ? t('Issue one when you have a customer to bring in.')
              : t('Issue a code and hand it to a customer to sign up.')
          }
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
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Customer')}</TableHead>
                  <TableHead>{t('Discount plan')}</TableHead>
                  <TableHead>{t('Valid until')}</TableHead>
                  <TableHead>{t('Remark')}</TableHead>
                  <TableHead className='text-right'>{t('Actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {codes.map((item) => {
                  const state = resolveCodeState(item, now)
                  // 光有号码，客户也不知道拿它干什么：号只有踩在注册链接上才有用，
                  // 所以拷出去的是整段邀请文案，鼠标停上去看到的也是同一段（一个字不差）。
                  const inviteMessage = t(
                    'Dealer customer invitation message',
                    {
                      link: buildCustomerInviteLink(item.code),
                      code: item.code,
                    }
                  )
                  return (
                    <TableRow key={item.id}>
                      <TableCell>
                        <div className='flex items-center gap-1.5'>
                          <span className='font-mono text-xs font-medium'>
                            {item.code}
                          </span>
                          <Tooltip>
                            <TooltipTrigger
                              render={
                                <CopyButton
                                  value={inviteMessage}
                                  aria-label={t('Copy')}
                                />
                              }
                            />
                            <TooltipContent className='max-w-sm items-start'>
                              <p className='text-left whitespace-pre-line'>
                                {inviteMessage}
                              </p>
                            </TooltipContent>
                          </Tooltip>
                        </div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          label={t(CODE_STATE_LABEL[state])}
                          variant={CODE_STATE_VARIANT[state]}
                          copyable={false}
                        />
                      </TableCell>
                      <TableCell className='text-sm'>
                        {item.bound_user_id > 0 ? (
                          item.bound_display_name ||
                          item.bound_username ||
                          `#${item.bound_user_id}`
                        ) : (
                          <span className='text-muted-foreground'>-</span>
                        )}
                      </TableCell>
                      <TableCell className='text-sm'>
                        {planLabel(item.plan_id)}
                      </TableCell>
                      <TableCell className='text-muted-foreground text-sm'>
                        {item.expired_at > 0
                          ? formatTimestamp(item.expired_at)
                          : t('Never expires')}
                      </TableCell>
                      <TableCell className='text-muted-foreground text-sm'>
                        {item.remark || '-'}
                      </TableCell>
                      <TableCell className='text-right'>
                        {state === 'available' ? (
                          <Button
                            variant='ghost'
                            size='sm'
                            className='text-destructive hover:text-destructive h-7 px-2'
                            onClick={() => setPendingRevoke(item)}
                          >
                            {t('Revoke')}
                          </Button>
                        ) : (
                          <span className='text-muted-foreground'>-</span>
                        )}
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
                onClick={() => void load(page - 1, scope)}
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
                onClick={() => void load(page + 1, scope)}
              >
                {t('Next')}
              </Button>
            </div>
          )}
        </div>
      )}

      <CustomerCodeIssueDialog
        open={issueOpen}
        onOpenChange={setIssueOpen}
        source={source}
        onIssued={() => void load(1, scope)}
      />

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

interface IssueDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  source: CustomerCodesSource
  onIssued: () => void
}

/**
 * 签发客户号。
 *
 * 做成弹窗而不是常驻表单：签号是低频动作（签一次用很久），而这一页日常被打开是为了
 * 看客户、改价、发额度——表单常驻会把真正要看的东西挤出屏幕。
 *
 * 这里没有"能用几次"这一项：一张号只拉一位客户，是后端写死的规则，界面上不给旋钮。
 */
function CustomerCodeIssueDialog({
  open,
  onOpenChange,
  source,
  onIssued,
}: IssueDialogProps) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const [count, setCount] = useState('1')
  const [expiredAt, setExpiredAt] = useState('')
  const [remark, setRemark] = useState('')

  // 每次打开都从干净的默认值开始：上一回填了一半的表单不该跟到下一次。
  useEffect(() => {
    if (!open) return
    setCount('1')
    setExpiredAt('')
    setRemark('')
  }, [open])

  const countValue = Number.parseInt(count, 10)
  const countValid =
    Number.isFinite(countValue) && countValue >= 1 && countValue <= MAX_BATCH

  const handleIssue = async () => {
    if (!countValid) return
    // 日期填的是"用到哪天"，所以按当天 23:59:59 截止，不给客户一个当天早上就失效的号。
    const expiresAt = expiredAt
      ? Math.floor(new Date(`${expiredAt}T23:59:59`).getTime() / 1000)
      : 0
    setSubmitting(true)
    try {
      const result = await source.create({
        count: countValue,
        // 号不挂方案：客户绑号只落归属，价格由平台管理员统一配（业务口径 2026-09-14）。
        plan_id: 0,
        expired_at: expiresAt,
        remark,
      })
      if (result.success) {
        toast.success(
          t('{{count}} customer code(s) issued', { count: countValue })
        )
        onOpenChange(false)
        onIssued()
      } else {
        toast.error(result.message || t('Failed to issue customer codes'))
      }
    } catch {
      toast.error(t('Failed to issue customer codes'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Issue a customer code')}
      description={t(
        'One code brings in one customer: he binds it and becomes yours.'
      )}
      contentClassName='sm:max-w-lg'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleIssue} disabled={submitting || !countValid}>
            {submitting ? t('Issuing') : t('Issue')}
          </Button>
        </>
      }
    >
      <div className='space-y-2'>
        <Label htmlFor='code-count'>{t('Quantity')}</Label>
        <Input
          id='code-count'
          value={count}
          inputMode='numeric'
          onChange={(event) => setCount(event.target.value)}
        />
        <p className='text-muted-foreground text-xs'>
          {t('Quantity must be between 1 and {{max}}.', { max: MAX_BATCH })}
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

      <div className='space-y-2'>
        <Label htmlFor='code-remark'>{t('Remark')}</Label>
        <Input
          id='code-remark'
          value={remark}
          maxLength={255}
          onChange={(event) => setRemark(event.target.value)}
        />
      </div>
    </Dialog>
  )
}
