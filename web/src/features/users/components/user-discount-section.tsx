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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ExternalLink, Link2Off, RefreshCw, UserCog } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  SideDrawerSection,
  SideDrawerSectionHeader,
} from '@/components/drawer-layout'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Spinner } from '@/components/ui/spinner'
import { StatusBadge } from '@/components/status-badge'
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  deleteDiscountBinding,
  getDiscountBindings,
} from '@/features/discounts/api'
import { getDiscountBindingSourceLabel } from '@/features/discounts/constants'
import { type DiscountBinding } from '@/features/discounts/types'

import { resetUserInviter } from '../api'
import { type User } from '../types'

// UserDiscountSection：用户详情页的「折扣」标签区（task-15）。
//
// 它在 UsersMutateDrawer 里以 SideDrawerSection 形式出现，呈现：
//   1) 邀请人 + 邀请码 + 邀请人数 + 佣金余额 + 邀请历史额度
//   2) 业务身份 + 归属经销商
//   3) 当前生效的 discount_bindings（plan_id + 来源 + 有效期 + 解绑）
//
// 不做的事（task-15 README §二「不做」清单 + scope 控制）：
//   - 不在用户详情页加 binding —— 跳到 /discounts 抽屉，由那边的 CRUD 处理
//   - 不做「重置邀请人」—— 仅超管可见 + 二次确认 + 留痕，下版本再说
//   - 不动用户列表页（已有「客户类型」列 + 行菜单）
//
// 数据流：
//   - 邀请人/码/人数/佣金/归属直接从 currentRow 读（controller GET /api/user/:id
//     在 task-15 这次补全了 discount_plan_id + parent_agent_id 输出）
//   - bindings 走 React Query 拉 /api/discount/admin/bindings，仅在 isUpdate 时启用
type UserDiscountSectionProps = {
  user: User
}

export function UserDiscountSection({ user }: UserDiscountSectionProps) {
  const { t } = useTranslation()
  const currentUser = useAuthStore((s) => s.auth.user)
  const queryClient = useQueryClient()
  const isSuperAdmin = currentUser?.role === ROLE.SUPER_ADMIN
  const [unbindTarget, setUnbindTarget] = useState<DiscountBinding | null>(null)
  const [isUnbinding, setIsUnbinding] = useState(false)

  // task-16：重置邀请人对话框状态。仅 isSuperAdmin=true 时才显示按钮，
  // 但 state 永远声明（hooks 顺序规则），只是 UI 不会渲染按钮入口。
  const [resetOpen, setResetOpen] = useState(false)
  const [newInviterId, setNewInviterId] = useState('')
  const [resetReason, setResetReason] = useState('')
  const [isResetting, setIsResetting] = useState(false)

  const parsedInviterId = useMemo(() => {
    const n = Number.parseInt(newInviterId, 10)
    return Number.isFinite(n) && n >= 0 ? n : null
  }, [newInviterId])
  const reasonLen = resetReason.length
  const reasonValid = reasonLen >= 10 && reasonLen <= 200
  const resetFormValid = parsedInviterId !== null && reasonValid

  const submitReset = useCallback(async () => {
    if (parsedInviterId === null || !reasonValid) return
    setIsResetting(true)
    try {
      const result = await resetUserInviter(user.id, {
        inviter_id: parsedInviterId,
        reason: resetReason,
      })
      if (result.success) {
        toast.success(t('Inviter reset successfully'))
        setResetOpen(false)
        setNewInviterId('')
        setResetReason('')
        // 刷新详情页 + bindings 两路查询（inviter_id 改了，前端 user 对象也得换）
        queryClient.invalidateQueries({ queryKey: ['user', user.id] })
        queryClient.invalidateQueries({ queryKey: ['user-discount-bindings', user.id] })
        return
      }
      toast.error(result.message || t('Failed to reset inviter'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsResetting(false)
    }
  }, [
    parsedInviterId,
    reasonValid,
    resetReason,
    user.id,
    queryClient,
    t,
  ])

  const bindingsQuery = useQuery({
    queryKey: ['user-discount-bindings', user.id],
    queryFn: async () => {
      const result = await getDiscountBindings({
        subject_id: user.id,
        subject_type: 'user',
        status: 1,
        page: 1,
        page_size: 50,
      })
      return result
    },
    enabled: !!user.id,
    staleTime: 30 * 1000,
  })

  const bindings = bindingsQuery.data?.data?.items ?? []
  const isLoading = bindingsQuery.isLoading

  const refresh = useCallback(() => {
    void bindingsQuery.refetch()
  }, [bindingsQuery])

  const handleUnbind = useCallback(async () => {
    if (!unbindTarget) return
    setIsUnbinding(true)
    try {
      const result = await deleteDiscountBinding(unbindTarget.id)
      if (result.success) {
        toast.success(t('Binding removed'))
        setUnbindTarget(null)
        refresh()
        return
      }
      toast.error(result.message || t('Failed to remove binding'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsUnbinding(false)
    }
  }, [unbindTarget, refresh, t])

  const hasInviter = (user.inviter_id ?? 0) > 0
  const hasAffCode = !!(user.aff_code && user.aff_code.length > 0)
  const affCount = user.aff_count ?? 0
  const affQuota = user.aff_quota ?? 0
  const affHistoryQuota = user.aff_history_quota ?? 0
  const isAgent = user.subject_type === 'agent'
  const hasParent = (user.parent_agent_id ?? 0) > 0

  return (
    <>
      <SideDrawerSection>
        <SideDrawerSectionHeader
          title={t('Discount')}
          description={t(
            'Discount plans, bindings, invitation chain, and commission balance for this user.'
          )}
        />

        {/* 行 1：业务身份 + 归属经销商 */}
        <div className='grid grid-cols-2 gap-3'>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Customer type')}
            </Label>
            <div className='mt-1'>
              <StatusBadge
                variant={isAgent ? 'pink' : hasParent ? 'info' : 'neutral'}
              >
                {isAgent
                  ? t('Agent')
                  : hasParent
                    ? t('Enterprise Discount')
                    : t('Common User')}
              </StatusBadge>
            </div>
          </div>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Parent agent')}
            </Label>
            <p className='mt-1 text-sm'>
              {hasParent
                ? t('Agent #{{id}}', { id: user.parent_agent_id })
                : '—'}
            </p>
          </div>
        </div>

        {/* task-16：仅超管可见的「重置邀请人」入口。
            放在归属行右侧——inviter_id 才是邀请积分链的核心，归属行 + 邀请行
            之间最自然的位置。普通管理员（role=10）即使能管用户也无权修改
            inviter_id（后端守卫会再次校验）。 */}
        {isSuperAdmin && (
          <div className='flex justify-end'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => setResetOpen(true)}
              className='border-amber-300 text-amber-700 hover:bg-amber-50 dark:text-amber-300 dark:hover:bg-amber-950'
            >
              <UserCog className='mr-2 h-4 w-4' />
              {t('Reset inviter')}
            </Button>
          </div>
        )}

        {/* 行 2：邀请码 + 邀请人 + 邀请人数 */}
        <div className='grid grid-cols-3 gap-3'>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Invite code')}
            </Label>
            <p className='mt-1 font-mono text-sm'>
              {hasAffCode ? user.aff_code : '—'}
            </p>
          </div>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Inviter')}
            </Label>
            <p className='mt-1 text-sm'>
              {hasInviter
                ? t('User #{{id}}', { id: user.inviter_id })
                : '—'}
            </p>
          </div>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Invited count')}
            </Label>
            <p className='mt-1 text-sm'>{affCount}</p>
          </div>
        </div>

        {/* 行 3：佣金余额 + 邀请历史额度 */}
        <div className='grid grid-cols-2 gap-3'>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Commission balance')}
            </Label>
            <p className='mt-1 text-sm'>
              {(affQuota / 500000).toFixed(2)}
            </p>
          </div>
          <div>
            <Label className='text-muted-foreground text-xs'>
              {t('Commission history')}
            </Label>
            <p className='mt-1 text-sm'>
              {(affHistoryQuota / 500000).toFixed(2)}
            </p>
          </div>
        </div>

        {/* 行 4：当前默认折扣方案（fast path 字段） */}
        <div>
          <Label className='text-muted-foreground text-xs'>
            {t('Default discount plan')}
          </Label>
          <p className='mt-1 text-sm'>
            {(user.discount_plan_id ?? 0) > 0
              ? t('Plan #{{id}}', { id: user.discount_plan_id })
              : '—'}
          </p>
        </div>
      </SideDrawerSection>

      <SideDrawerSection>
        <div className='flex items-center justify-between gap-2'>
          <SideDrawerSectionHeader
            title={t('Active bindings')}
            description={t(
              'Each binding attaches one discount plan to this user. The pricing rules pick which plan applies per model.'
            )}
          />
          <Button
            type='button'
            variant='ghost'
            size='icon-sm'
            aria-label={t('Refresh')}
            onClick={refresh}
          >
            <RefreshCw />
          </Button>
        </div>

        <div className='flex items-center justify-between gap-2'>
          <p className='text-muted-foreground text-xs'>
            {isLoading
              ? t('Loading...')
              : t('{{count}} active', { count: bindings.length })}
          </p>
          <Button
            type='button'
            variant='outline'
            size='sm'
            render={<Link to='/discounts' />}
          >
            <ExternalLink className='h-4 w-4' />
            {t('Manage in Discounts')}
          </Button>
        </div>

        {bindings.length === 0 && !isLoading && (
          <p className='text-muted-foreground text-sm'>
            {t('No active discount binding for this user.')}
          </p>
        )}

        {bindings.length > 0 && (
          <ul className='border-border/60 divide-border/60 divide-y rounded-md border'>
            {bindings.map((binding) => (
              <li
                key={binding.id}
                className='flex items-center justify-between gap-3 px-3 py-2 text-sm'
              >
                <div className='min-w-0'>
                  <p className='truncate font-medium'>
                    {t('Plan #{{id}}', { id: binding.plan_id })}
                  </p>
                  <p className='text-muted-foreground text-xs'>
                    {getDiscountBindingSourceLabel(t, binding.source)} ·{' '}
                    {binding.effective_from > 0
                      ? formatTimestampToDate(binding.effective_from)
                      : t('Unlimited')}{' '}
                    →{' '}
                    {binding.effective_to > 0
                      ? formatTimestampToDate(binding.effective_to)
                      : t('Unlimited')}
                  </p>
                </div>
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-sm'
                  aria-label={t('Unbind')}
                  onClick={() => setUnbindTarget(binding)}
                >
                  <Link2Off />
                </Button>
              </li>
            ))}
          </ul>
        )}

        {isLoading && (
          <div className='text-muted-foreground flex items-center gap-2 text-xs'>
            <Spinner className='h-3 w-3' />
            {t('Loading bindings...')}
          </div>
        )}
      </SideDrawerSection>

      <AlertDialog
        open={unbindTarget !== null}
        onOpenChange={(nextOpen) => !nextOpen && setUnbindTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Unbind discount plan?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {unbindTarget
                ? t(
                    'This user falls back to the official price as soon as Plan #{{id}} is unbound.',
                    { id: unbindTarget.plan_id }
                  )
                : ''}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isUnbinding}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleUnbind}
              disabled={isUnbinding}
              variant='destructive'
            >
              {isUnbinding ? t('Unbinding...') : t('Unbind')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* task-16：重置邀请人对话框（仅超管）。 */}
      <AlertDialog open={resetOpen} onOpenChange={setResetOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Reset inviter?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'This will overwrite the user\'s current inviter. Historical commission and quota are NOT recalculated. The action is recorded in the audit log with your reason.'
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className='space-y-3'>
            <div>
              <Label htmlFor='new-inviter-id' className='text-sm'>
                {t('New inviter user ID (0 to clear)')}
              </Label>
              <Input
                id='new-inviter-id'
                type='number'
                min={0}
                value={newInviterId}
                onChange={(e) => setNewInviterId(e.target.value)}
                placeholder={user.inviter_id ? String(user.inviter_id) : '0'}
                className='mt-1'
              />
              <p className='text-muted-foreground mt-1 text-xs'>
                {t('Current inviter')}: {user.inviter_id || '—'}
              </p>
            </div>
            <div>
              <Label htmlFor='reset-reason' className='text-sm'>
                {t('Reason (10–200 characters)')}
              </Label>
              <Input
                id='reset-reason'
                value={resetReason}
                onChange={(e) => setResetReason(e.target.value)}
                placeholder={t('Why are you resetting the inviter?')}
                className='mt-1'
              />
              <p
                className={`mt-1 text-xs ${
                  reasonValid
                    ? 'text-muted-foreground'
                    : 'text-rose-500'
                }`}
              >
                {reasonLen} / 200
              </p>
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isResetting}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={submitReset}
              disabled={!resetFormValid || isResetting}
              variant='destructive'
            >
              {isResetting ? (
                <>
                  <Spinner className='mr-2 h-4 w-4' />
                  {t('Resetting...')}
                </>
              ) : (
                t('Reset')
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}