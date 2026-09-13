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
import { Percent } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
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
  createDiscountBinding,
  deleteDiscountBinding,
  getDiscountBindings,
  getDiscountPlans,
} from '@/features/discounts/api'
import {
  DISCOUNT_BINDING_SOURCE,
  DISCOUNT_PLAN_STATUS,
  DISCOUNT_SUBJECT,
  type DiscountBinding,
  type DiscountPlan,
} from '@/features/discounts/types'

import { ERROR_MESSAGES } from '../../constants'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: { id: number; username: string } | null
  onSuccess?: () => void
}

/**
 * 给一个客户配专属价：从用户列表点进来时客户已经选好了，只要挑一套折扣方案绑上。
 *
 * 这跟折扣方案页那个「绑定客户」抽屉是同一件事的两个方向——那边是拿着方案找人，
 * 这边是点着人配方案。绑上之后客户类型列会显示成「企业折扣」（判定见
 * constants.ts 的 resolveCustomerType），解绑即回到官方标价。
 *
 * 这是业务身份，不是权限角色：绑不绑都不影响这个人登录、消费和管理端可见性。
 */
export function SetDiscountDialog({
  open,
  onOpenChange,
  user,
  onSuccess,
}: Props) {
  const { t } = useTranslation()
  const [plans, setPlans] = useState<DiscountPlan[]>([])
  const [bindings, setBindings] = useState<DiscountBinding[]>([])
  const [selectedPlanId, setSelectedPlanId] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [unbindingId, setUnbindingId] = useState<number | null>(null)

  const loadPlans = useCallback(async () => {
    const result = await getDiscountPlans({
      page: 1,
      page_size: 100,
      status: DISCOUNT_PLAN_STATUS.ENABLED,
    })
    setPlans(result.success ? (result.data?.items ?? []) : [])
  }, [])

  const loadBindings = useCallback(async (userId: number) => {
    const result = await getDiscountBindings({
      subject_type: DISCOUNT_SUBJECT.USER,
      subject_id: userId,
      page: 1,
      page_size: 20,
    })
    const items = result.success ? (result.data?.items ?? []) : []
    setBindings(
      items.filter((binding) => binding.status === DISCOUNT_PLAN_STATUS.ENABLED)
    )
  }, [])

  useEffect(() => {
    setSelectedPlanId('')
    if (!open) {
      setBindings([])
      return
    }
    void loadPlans()
    if (user) void loadBindings(user.id)
  }, [open, user, loadPlans, loadBindings])

  /** 方案名以方案列表为准；方案被停用后列表里没有，退回显示编号，不留空白。 */
  const planLabel = (planId: number) =>
    plans.find((plan) => plan.id === planId)?.name ||
    t('Plan #{{id}}', { id: planId })

  const handleBind = async () => {
    const planId = Number.parseInt(selectedPlanId, 10)
    if (!user || !Number.isFinite(planId) || planId <= 0) return
    setSubmitting(true)
    try {
      const result = await createDiscountBinding({
        subject_type: DISCOUNT_SUBJECT.USER,
        subject_id: user.id,
        plan_id: planId,
        effective_from: 0,
        effective_to: 0,
        source: DISCOUNT_BINDING_SOURCE.MANUAL,
      })
      if (result.success) {
        toast.success(
          t('{{username}} follows this plan now', { username: user.username })
        )
        setSelectedPlanId('')
        await loadBindings(user.id)
        onSuccess?.()
        return
      }
      toast.error(result.message || t('Failed to bind the discount plan'))
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setSubmitting(false)
    }
  }

  const handleUnbind = async (binding: DiscountBinding) => {
    if (!user) return
    setUnbindingId(binding.id)
    try {
      const result = await deleteDiscountBinding(binding.id)
      if (result.success) {
        toast.success(
          t('{{username}} is back to the list price', {
            username: user.username,
          })
        )
        await loadBindings(user.id)
        onSuccess?.()
        return
      }
      toast.error(result.message || t('Failed to unbind the discount plan'))
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setUnbindingId(null)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        <>
          <Percent className='h-5 w-5' />
          {t('Set Exclusive Discount')}
        </>
      }
      description={t(
        'Bind a discount plan to this customer. Once bound, their consumption is priced by that plan.'
      )}
      contentClassName='sm:max-w-md'
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
            onClick={handleBind}
            disabled={submitting || !selectedPlanId || !user}
          >
            {t('Bind')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        {user && (
          <p className='text-muted-foreground text-sm'>
            {user.username} (ID: {user.id})
          </p>
        )}

        <div className='space-y-2'>
          <Label>{t('Current plan')}</Label>
          {bindings.length === 0 ? (
            <p className='text-muted-foreground text-sm'>
              {t('No discount plan is bound to this customer.')}
            </p>
          ) : (
            <div className='space-y-2'>
              {bindings.map((binding) => (
                <div
                  key={binding.id}
                  className='flex items-center justify-between rounded-md border px-3 py-2'
                >
                  <span className='text-sm'>{planLabel(binding.plan_id)}</span>
                  <Button
                    variant='ghost'
                    size='sm'
                    className='text-destructive hover:text-destructive h-7 px-2'
                    disabled={unbindingId === binding.id}
                    onClick={() => void handleUnbind(binding)}
                  >
                    {t('Unbind')}
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className='space-y-2'>
          <Label>{t('Select a plan')}</Label>
          <Select
            items={plans.map((plan) => ({
              value: String(plan.id),
              label: plan.name,
            }))}
            value={selectedPlanId}
            onValueChange={(value) => setSelectedPlanId(value ?? '')}
          >
            <SelectTrigger className='w-full'>
              <SelectValue placeholder={t('Select a plan')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {plans.map((plan) => (
                  <SelectItem key={plan.id} value={String(plan.id)}>
                    {plan.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <p className='text-muted-foreground text-xs'>
            {t(
              'Only enabled plans are listed. One customer follows one plan at a time.'
            )}
          </p>
        </div>
      </div>
    </Dialog>
  )
}
