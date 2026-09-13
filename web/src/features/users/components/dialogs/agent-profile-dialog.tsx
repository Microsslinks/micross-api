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
import { Store } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { CustomerCodesPanel } from '@/features/dealer/components/customer-codes-panel'
import type { DealerPlanOption } from '@/features/dealer/types'
import { getDiscountPlans } from '@/features/discounts/api'
import { DISCOUNT_PLAN_STATUS } from '@/features/discounts/types'

import {
  getAgentCustomerCodes,
  getAgentProfile,
  issueAgentCustomerCodes,
  revokeAgentCustomerCode,
  updateAgentProfile,
} from '../../api'
import { ERROR_MESSAGES } from '../../constants'

/** 界面上的三档毛利，与「设为经销商」那个弹窗同一套预设。 */
const MARKUP_PRESETS = [5, 10, 20]
/** 平台毛利的下限，与后端 NormalizeAgentMarkup 里的 1.05 是同一条线。 */
const MIN_MARKUP_PERCENT = 5

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: { id: number; username: string } | null
  onSuccess?: () => void
}

/**
 * 经销商设置。
 *
 * 一个人已经是经销商之后，改价和下号都在这里：上面是他的经营参数（加价率、
 * 给客户的零售折扣下限、能不能给下属发额度、备注），下面是他手里的客户号。
 * 这两件事本来就一起发生——客户号上的折扣方案来自他货架上那些方案，
 * 方案又受他的折扣下限约束，所以放在同一处而不是拆两个入口。
 *
 * 加价率填的是百分数（10 = 加价 10%），存的是乘数（1.100000）；折扣下限填的是
 * 小数（0.8 = 八折），与折扣方案页看到的写法一致。0 表示平台不给下限。
 */
export function AgentProfileDialog({ open, onOpenChange, user, onSuccess }: Props) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [markupPercent, setMarkupPercent] = useState('')
  const [minDiscount, setMinDiscount] = useState('0')
  const [issueQuotaEnabled, setIssueQuotaEnabled] = useState(true)
  const [remark, setRemark] = useState('')
  const [plans, setPlans] = useState<DealerPlanOption[]>([])

  const loadPlans = useCallback(async () => {
    const result = await getDiscountPlans({
      page: 1,
      page_size: 100,
      status: DISCOUNT_PLAN_STATUS.ENABLED,
    })
    const items = result.success ? (result.data?.items ?? []) : []
    setPlans(
      items.map((plan) => ({
        id: plan.id,
        name: plan.name,
        base_discount: plan.base_discount,
        billing_mode: plan.billing_mode,
        remark: plan.remark ?? '',
      }))
    )
  }, [])

  useEffect(() => {
    if (!open || !user) return
    let cancelled = false
    setLoading(true)
    void (async () => {
      try {
        const result = await getAgentProfile(user.id)
        if (cancelled || !result.data) return
        setMarkupPercent(
          String(Math.round((Number.parseFloat(result.data.markup_ratio) - 1) * 100))
        )
        setMinDiscount(result.data.min_discount)
        setIssueQuotaEnabled(result.data.issue_quota_enabled === 1)
        setRemark(result.data.remark)
      } catch {
        if (!cancelled) toast.error(t('Failed to load the dealer profile'))
      } finally {
        if (!cancelled) setLoading(false)
      }
      await loadPlans()
    })()
    return () => {
      cancelled = true
    }
  }, [open, user, loadPlans, t])

  const markupPercentValue = Number.parseFloat(markupPercent)
  const markupValid =
    markupPercent.trim() !== '' &&
    Number.isFinite(markupPercentValue) &&
    markupPercentValue >= MIN_MARKUP_PERCENT
  const minDiscountValue = Number.parseFloat(minDiscount)
  const minDiscountValid =
    minDiscount.trim() !== '' &&
    Number.isFinite(minDiscountValue) &&
    minDiscountValue >= 0 &&
    minDiscountValue <= 1

  const handleSave = async () => {
    if (!user || !markupValid || !minDiscountValid) return
    setSubmitting(true)
    try {
      const result = await updateAgentProfile(user.id, {
        markup_ratio: (1 + markupPercentValue / 100).toFixed(6),
        min_discount: minDiscountValue.toFixed(6),
        issue_quota_enabled: issueQuotaEnabled ? 1 : 0,
        remark,
      })
      if (result.success) {
        toast.success(t('Dealer settings saved'))
        onSuccess?.()
      } else {
        toast.error(result.message || t('Failed to save dealer settings'))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setSubmitting(false)
    }
  }

  // 客户号那三个动作：平台替这位经销商签，走的是带 :id 的管理员路径。
  // 包在 useMemo 里是为了让面板的加载只跟着这个人变，不跟着父组件重渲染。
  const codesSource = useMemo(() => {
    const agentId = user?.id ?? 0
    return {
      list: async (page: number) => {
        const result = await getAgentCustomerCodes(agentId, page)
        return {
          items: result.data?.items ?? [],
          total: result.data?.total ?? 0,
        }
      },
      create: async (payload: Parameters<typeof issueAgentCustomerCodes>[1]) => {
        const result = await issueAgentCustomerCodes(agentId, payload)
        return {
          success: result.success,
          message: result.message,
          items: result.data?.items,
        }
      },
      revoke: async (codeId: number) => {
        const result = await revokeAgentCustomerCode(agentId, codeId)
        return { success: result.success, message: result.message }
      },
      plans,
    }
  }, [user?.id, plans])

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        <>
          <Store className='h-5 w-5' />
          {t('Dealer Settings')}
        </>
      }
      description={t(
        'His operating parameters and the customer codes he hands out.'
      )}
      contentClassName='sm:max-w-2xl'
      titleClassName='flex items-center gap-2'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            {t('Close')}
          </Button>
          <Button
            onClick={handleSave}
            disabled={submitting || !user || !markupValid || !minDiscountValid}
          >
            {t('Save')}
          </Button>
        </>
      }
    >
      {loading ? (
        <p className='text-muted-foreground text-sm'>{t('Loading')}</p>
      ) : (
        <div className='space-y-5'>
          {user && (
            <p className='text-muted-foreground text-sm'>
              {user.username} (ID: {user.id})
            </p>
          )}

          <div className='space-y-2'>
            <Label htmlFor='profile-markup'>{t('Platform markup')}</Label>
            <div className='flex items-center gap-2'>
              {MARKUP_PRESETS.map((preset) => (
                <Button
                  key={preset}
                  type='button'
                  size='sm'
                  variant={
                    markupPercent === String(preset) ? 'default' : 'outline'
                  }
                  onClick={() => setMarkupPercent(String(preset))}
                >
                  {preset}%
                </Button>
              ))}
              <Input
                id='profile-markup'
                className='w-24'
                value={markupPercent}
                inputMode='decimal'
                onChange={(event) => setMarkupPercent(event.target.value)}
              />
              <span className='text-muted-foreground text-sm'>%</span>
            </div>
            <p className='text-muted-foreground text-xs'>
              {markupValid
                ? t(
                    'What the platform adds on top of its own purchase cost for this dealer.'
                  )
                : t('Markup cannot be lower than 5%.')}
            </p>
          </div>

          <div className='space-y-2'>
            <Label htmlFor='profile-min-discount'>
              {t('Lowest selling discount')}
            </Label>
            <Input
              id='profile-min-discount'
              className='w-32'
              value={minDiscount}
              inputMode='decimal'
              onChange={(event) => setMinDiscount(event.target.value)}
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'The floor for the prices this dealer gives customers. 0 means no limit.'
              )}
            </p>
          </div>

          <div className='flex items-center justify-between gap-4'>
            <div className='space-y-0.5'>
              <Label htmlFor='profile-issue-quota'>
                {t('Allow issuing quota to customers')}
              </Label>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'When off, this dealer can no longer hand quota to his customers.'
                )}
              </p>
            </div>
            <Switch
              id='profile-issue-quota'
              checked={issueQuotaEnabled}
              onCheckedChange={(checked) => setIssueQuotaEnabled(checked)}
            />
          </div>

          <div className='space-y-2'>
            <Label htmlFor='profile-remark'>{t('Remark')}</Label>
            <Input
              id='profile-remark'
              value={remark}
              maxLength={255}
              onChange={(event) => setRemark(event.target.value)}
            />
          </div>

          {user && <CustomerCodesPanel source={codesSource} />}
        </div>
      )}
    </Dialog>
  )
}
