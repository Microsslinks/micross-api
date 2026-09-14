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
import { Loader2, Plus, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemOptions } from '@/features/system-settings/hooks/use-system-options'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import { safeJsonParseWithValidation } from '@/features/system-settings/utils/json-parser'
import { isArray } from '@/features/system-settings/utils/json-validators'

const OPTION_KEY = 'payment_setting.amount_options'

type AmountOptionsManageDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Called after a successful save so the wallet page can refresh presets. */
  onSaved?: () => void
}

/**
 * 「充值金额管理」对话框（仅管理员）
 *
 * CRUD 列表维护 payment_setting.amount_options（预设充值额度）：
 * 每行一个额度、右侧删除、底部添加行；保存后通知钱包页刷新快捷充值卡片。
 */
export function AmountOptionsManageDialog({
  open,
  onOpenChange,
  onSaved,
}: AmountOptionsManageDialogProps) {
  const { t } = useTranslation()
  const { data: options, isLoading } = useSystemOptions()
  const updateOption = useUpdateOption()
  const [draft, setDraft] = useState('[]')
  const [newAmount, setNewAmount] = useState('')

  const serverValue =
    options?.data?.find((option) => option.key === OPTION_KEY)?.value ?? '[]'

  // Only resync the draft when the dialog opens, never mid-edit.
  const wasOpen = useRef(false)
  useEffect(() => {
    if (open && !wasOpen.current) {
      setDraft(serverValue)
      setNewAmount('')
    }
    wasOpen.current = open
  }, [open, serverValue])

  const amounts = useMemo(() => {
    const parsed = safeJsonParseWithValidation<unknown[]>(draft, {
      fallback: [],
      validator: isArray,
      validatorMessage: t('Amount options must be a JSON array'),
      context: 'amount options',
    })

    return parsed
      .filter((item) => typeof item === 'number' || !isNaN(Number(item)))
      .map(Number)
      .sort((a, b) => a - b)
  }, [draft, t])

  const commit = (next: number[]) => {
    setDraft(JSON.stringify([...next].sort((a, b) => a - b), null, 2))
  }

  const newAmountNum = parseFloat(newAmount)
  const canAdd =
    !isNaN(newAmountNum) && newAmountNum > 0 && !amounts.includes(newAmountNum)

  const handleAdd = () => {
    if (!canAdd) return
    commit([...amounts, newAmountNum])
    setNewAmount('')
  }

  const handleRemove = (amount: number) => {
    commit(amounts.filter((a) => a !== amount))
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      handleAdd()
    }
  }

  const handleSave = () => {
    updateOption.mutate(
      { key: OPTION_KEY, value: draft },
      {
        onSuccess: (data) => {
          if (data.success) {
            onSaved?.()
            onOpenChange(false)
          }
        },
      }
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Manage top-up amounts')}
      contentClassName='sm:max-w-[440px]'
      contentHeight='auto'
      bodyClassName='space-y-2'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={handleSave}
            disabled={updateOption.isPending || isLoading}
          >
            {updateOption.isPending && (
              <Loader2 className='mr-2 h-4 w-4 animate-spin' />
            )}
            {t('Save')}
          </Button>
        </>
      }
    >
      {isLoading ? (
        <Skeleton className='h-40 w-full' />
      ) : (
        <>
          {amounts.length === 0 && (
            <EmptyState
              title={t(
                'No amount options configured. Add amounts below to get started.'
              )}
              className='rounded-lg border border-dashed'
              size='sm'
            />
          )}
          {amounts.map((amount) => (
            <div
              key={amount}
              className='flex items-center justify-between rounded-lg border px-4 py-2.5'
            >
              <span className='font-mono text-base font-semibold'>
                ${amount}
              </span>
              <Button
                type='button'
                variant='ghost'
                size='icon-sm'
                aria-label={t('Remove ${{amount}}', { amount })}
                onClick={() => handleRemove(amount)}
              >
                <Trash2 className='text-muted-foreground h-4 w-4' />
              </Button>
            </div>
          ))}
          <div className='flex gap-2 pt-1'>
            <Input
              type='number'
              step='0.01'
              min='0'
              placeholder={t('e.g., 100')}
              value={newAmount}
              onChange={(e) => setNewAmount(e.target.value)}
              onKeyDown={handleKeyDown}
            />
            <Button
              type='button'
              onClick={handleAdd}
              disabled={!canAdd}
              className='shrink-0'
            >
              <Plus className='h-4 w-4' />
              <span className='ml-1'>{t('Add')}</span>
            </Button>
          </div>
        </>
      )}
    </Dialog>
  )
}
