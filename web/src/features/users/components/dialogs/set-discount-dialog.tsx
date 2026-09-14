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
import { DISCOUNT_PLAN_LIMITS } from '@/features/discounts/constants'
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
  // 对话框开着期间改过绑定就记一笔，等用户自己关掉时再统一刷新用户列表：
  // 绑一次刷一次会让表格在对话框背后闪，而且可能把还没做完的连续绑定打断。
  const [dirty, setDirty] = useState(false)

  const loadPlans = useCallback(async () => {
    const result = await getDiscountPlans({
      page: 1,
      page_size: 100,
      status: DISCOUNT_PLAN_STATUS.ENABLED,
    })
    setPlans(result.success ? (result.data?.items ?? []) : [])
  }, [])

  const loadBindings = useCallback(async (userId: number) => {
    // status 交给后端过滤：解绑过的绑定是停用状态留在库里的历史记录，
    // 拉回来自己筛的话，攒够一页历史记录就会把还在生效的那几条挤出去，
    // 界面会显示成"这个客户没有方案"。
    const result = await getDiscountBindings({
      subject_type: DISCOUNT_SUBJECT.USER,
      subject_id: userId,
      status: DISCOUNT_PLAN_STATUS.ENABLED,
      page: 1,
      page_size: 100,
    })
    setBindings(result.success ? (result.data?.items ?? []) : [])
  }, [])

  // 只认 user.id：父组件每次渲染都会传一个新对象，按对象本身依赖会让
  // 每次刷新列表都把刚选好的方案清空。
  const userId = user?.id ?? 0

  useEffect(() => {
    setSelectedPlanId('')
    if (!open) {
      setBindings([])
      return
    }
    void loadPlans()
    if (userId > 0) void loadBindings(userId)
  }, [open, userId, loadPlans, loadBindings])

  /** 关闭对话框是唯一刷新用户列表的时机：开着的时候连续绑，关了再一起刷新。 */
  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && dirty) {
      setDirty(false)
      onSuccess?.()
    }
    onOpenChange(nextOpen)
  }

  /** 方案名以方案列表为准；方案被停用后列表里没有，退回显示编号，不留空白。 */
  const planLabel = (planId: number) =>
    plans.find((plan) => plan.id === planId)?.name ||
    t('Plan #{{id}}', { id: planId })

  // 后端也会拦（同一个上限），这里先拦是为了让按钮当场变灰、把原因写在旁边，
  // 而不是让他点了才收到一句"已达上限"。
  const atPlanLimit =
    bindings.length >= DISCOUNT_PLAN_LIMITS.MAX_BINDINGS_PER_SUBJECT

  const handleBind = async () => {
    const planId = Number.parseInt(selectedPlanId, 10)
    if (!user || !Number.isFinite(planId) || planId <= 0 || atPlanLimit) return
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
        setDirty(true)
        await loadBindings(user.id)
        // 不关对话框：一位客户可以同时挂多套方案，绑完让他自己关。
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
        setDirty(true)
        await loadBindings(user.id)
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
      onOpenChange={handleOpenChange}
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
            disabled={submitting || !selectedPlanId || !user || atPlanLimit}
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
          {atPlanLimit ? (
            <p className='text-destructive text-xs'>
              {t(
                'One customer can follow at most {{plans}} plans. Unbind one before binding another.',
                { plans: DISCOUNT_PLAN_LIMITS.MAX_BINDINGS_PER_SUBJECT }
              )}
            </p>
          ) : (
            <p className='text-muted-foreground text-xs'>
              {t(
                'Only enabled plans are listed. You can bind more than one plan; the pricing rules decide which one applies.'
              )}
            </p>
          )}
        </div>
      </div>
    </Dialog>
  )
}
