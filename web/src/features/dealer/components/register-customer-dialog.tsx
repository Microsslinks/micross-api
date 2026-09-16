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
import { Loader2, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { handleServerError } from '@/lib/handle-server-error'

import { registerSelfAgentCustomer } from '../api'
import type { AgentCustomer } from '../types'

interface RegisterCustomerDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /**
   * 成功后回调：把新行回给父组件直接插进列表，省一次为了刷新再发的查询。
   * 父组件没接的话也无所谓，关弹窗 + toast 已经够了。
   */
  onSuccess?: (created: AgentCustomer) => void
}

/**
 * 经销商代注册客户弹窗。
 *
 * 提交后账号立刻归到当前经销商名下（parent_agent_id 写死成调用方 id），
 * 后端给客户默认送 QuotaForNewUser，无需经销商手动充额度。
 *
 * 用户名唯一性 / 邮箱冲突由后端兜住：失败时把后端 message 直接 toast 出来，
 * 让经销商看到"用户名已存在"那一类具体说法，而不是干瘪的"注册失败"。
 */
export function RegisterCustomerDialog({
  open,
  onOpenChange,
  onSuccess,
}: RegisterCustomerDialogProps) {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // 关掉时清空表单：下次打开不该看见上一次的输入；尤其是密码这种字段。
  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setUsername('')
      setPassword('')
      setDisplayName('')
    }
    onOpenChange(next)
  }

  const canSubmit =
    username.trim() !== '' && password.length >= 8 && !submitting

  const handleSubmit = async () => {
    if (!canSubmit) return
    setSubmitting(true)
    try {
      const result = await registerSelfAgentCustomer({
        username: username.trim(),
        password,
        display_name: displayName.trim() || undefined,
      })
      if (result.success && result.data) {
        toast.success(t('Customer registered'))
        onSuccess?.(result.data)
        handleOpenChange(false)
        return
      }
      toast.error(result.message || t('Failed to register customer'))
    } catch (error) {
      handleServerError(error)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      title={t('dealer.registerCustomer.title')}
      description={t(
        'Register a new customer under your account. They can log in right away.'
      )}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button
            variant='outline'
            onClick={() => handleOpenChange(false)}
            disabled={submitting}
          >
            {t('Cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={!canSubmit}>
            {submitting ? (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            ) : (
              <UserPlus className='mr-2 h-4 w-4' />
            )}
            {t('dealer.registerCustomer')}
          </Button>
        </>
      }
    >
      <div className='space-y-2'>
        <Label htmlFor='register-customer-username'>
          {t('Username')} *
        </Label>
        <Input
          id='register-customer-username'
          value={username}
          autoComplete='off'
          onChange={(event) => setUsername(event.target.value)}
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='register-customer-password'>
          {t('Password')} *
        </Label>
        <Input
          id='register-customer-password'
          type='password'
          value={password}
          autoComplete='new-password'
          onChange={(event) => setPassword(event.target.value)}
        />
        <p className='text-muted-foreground text-xs'>
          {t('At least 8 characters.')}
        </p>
      </div>
      <div className='space-y-2'>
        <Label htmlFor='register-customer-display-name'>
          {t('Display name')}
        </Label>
        <Input
          id='register-customer-display-name'
          value={displayName}
          autoComplete='off'
          onChange={(event) => setDisplayName(event.target.value)}
        />
      </div>
    </Dialog>
  )
}