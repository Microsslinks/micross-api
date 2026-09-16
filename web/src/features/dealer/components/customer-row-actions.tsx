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
import { Copy, KeyRound, Loader2, MoreHorizontal, ShieldOff } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Label } from '@/components/ui/label'
import { handleServerError } from '@/lib/handle-server-error'
import { USER_STATUS } from '@/features/users/constants'

import {
  disableSelfAgentCustomer,
  resetSelfAgentCustomerPassword,
} from '../api'
import type { AgentCustomer } from '../types'

interface CustomerRowActionsProps {
  customer: AgentCustomer
  /**
   * 任意一个写动作做完后都让父组件重新拉一遍列表（停用会改变 status；
   * 重置密码会让旧 token/session 失效，但列表行没变化——保险起见也刷一次）。
   */
  onRefresh: () => void
}

/**
 * 客户行尾的「更多」菜单：放两个会改变这位客户身份的动作——重置密码、停用。
 *
 * 重置密码：
 *   - 后端给 8 位随机串，明文只在这一次响应里出现：弹窗顶部红字提示必须当面/私聊告知客户。
 *   - 提供复制按钮，把明文放到剪贴板（经销商可以贴到自己的聊天里发给客户）。
 *
 * 停用：
 *   - 二次确认走 ConfirmDialog，destructive 样式，避免误点。
 *   - 幂等：客户已是 disabled 时菜单项直接 disabled（不让重复触发）。
 *   - 同时级联把他的 token 一起 disable，下次请求就拒。
 */
export function CustomerRowActions({
  customer,
  onRefresh,
}: CustomerRowActionsProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  // 重置密码：成功后展示明文；关闭展示窗后 onRefresh 让父组件拉新状态。
  const [resetting, setResetting] = useState(false)
  const [newPassword, setNewPassword] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  // 停用：二次确认 + 真正执行
  const [confirmingDisable, setConfirmingDisable] = useState(false)
  const [disabling, setDisabling] = useState(false)

  const isAlreadyDisabled = customer.status === USER_STATUS.DISABLED
  const name = customer.display_name || customer.username

  const handleReset = async () => {
    setOpen(false) // 关掉菜单，让弹窗独立呈现
    setResetting(true)
    try {
      const result = await resetSelfAgentCustomerPassword(customer.user_id)
      if (result.success && result.data) {
        setNewPassword(result.data.password)
        return
      }
      toast.error(result.message || t('Failed to reset password'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setResetting(false)
    }
  }

  const handleDisable = async () => {
    setDisabling(true)
    try {
      const result = await disableSelfAgentCustomer(customer.user_id)
      if (result.success) {
        toast.success(t('Customer disabled'))
        setConfirmingDisable(false)
        onRefresh()
        return
      }
      toast.error(result.message || t('Failed to disable customer'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setDisabling(false)
    }
  }

  const handleCopyPassword = async () => {
    if (!newPassword) return
    try {
      await navigator.clipboard.writeText(newPassword)
      setCopied(true)
      toast.success(t('Copied'))
      // 2 秒后让按钮文字回到「复制」，避免一直显示「已复制」误导
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error(t('Failed to copy'))
    }
  }

  return (
    <>
      <DropdownMenu open={open} onOpenChange={setOpen}>
        <DropdownMenuTrigger
          render={
            <Button
              variant='ghost'
              size='sm'
              className='h-7 w-7 p-0'
              aria-label={t('More actions')}
            >
              <MoreHorizontal className='h-4 w-4' />
            </Button>
          }
        />
        <DropdownMenuContent align='end'>
          <DropdownMenuItem onClick={handleReset} disabled={resetting}>
            <KeyRound className='mr-2 h-4 w-4' />
            {t('dealer.resetPassword')}
          </DropdownMenuItem>
          <DropdownMenuItem
            variant='destructive'
            disabled={isAlreadyDisabled}
            onClick={() => {
              setOpen(false)
              setConfirmingDisable(true)
            }}
          >
            <ShieldOff className='mr-2 h-4 w-4' />
            {t('dealer.disable')}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* 重置密码后展示明文：顶部红字提醒 + 单色按钮区 + 关闭时刷新。 */}
      <Dialog
        open={newPassword !== null}
        onOpenChange={(next) => {
          if (!next) {
            setNewPassword(null)
            setCopied(false)
            onRefresh()
          }
        }}
        title={t('Password reset for {{name}}', { name })}
        contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
        contentHeight='auto'
        bodyClassName='space-y-4'
        footer={
          <Button onClick={() => setNewPassword(null)}>
            {t('Done')}
          </Button>
        }
      >
        <div className='text-destructive text-sm font-medium'>
          {t('dealer.resetPassword.notify')}
        </div>
        <div className='space-y-2'>
          <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('New password')}
          </Label>
          <div className='flex items-center gap-2'>
            <code className='bg-muted flex-1 break-all rounded px-3 py-2 font-mono text-base'>
              {newPassword}
            </code>
            <Button
              variant='outline'
              size='sm'
              onClick={handleCopyPassword}
              className='shrink-0'
            >
              <Copy className='mr-1 h-4 w-4' />
              {copied ? t('Copied') : t('Copy')}
            </Button>
          </div>
        </div>
      </Dialog>

      {/* 停用二次确认：destructive 样式 + ConfirmDialog 自带 loading 态。 */}
      <ConfirmDialog
        open={confirmingDisable}
        onOpenChange={setConfirmingDisable}
        title={t('dealer.disable')}
        desc={t('dealer.disable.confirm', { name })}
        destructive
        isLoading={disabling}
        handleConfirm={handleDisable}
      />

      {/* 重置中单独 toast 一个 loader，避免菜单关掉后没反馈： */}
      {resetting && (
        <div className='sr-only' role='status' aria-live='polite'>
          <Loader2 className='h-4 w-4 animate-spin' />
          {t('Resetting password...')}
        </div>
      )}
    </>
  )
}