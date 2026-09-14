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
import { Loader2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { AmountDiscountVisualEditor } from '@/features/business-settings/billing/sections/payment/amount-discount-visual-editor'
import { useSystemOptions } from '@/features/system-settings/hooks/use-system-options'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'

const OPTION_KEY = 'payment_setting.amount_discount'

type AmountDiscountManageDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Called after a successful save so the wallet page can refresh presets. */
  onSaved?: () => void
}

/**
 * 「充值金额折扣管理」对话框（仅管理员）
 *
 * 编辑 payment_setting.amount_discount（满额折扣等级），
 * 保存后通知钱包页刷新快捷充值卡片。
 */
export function AmountDiscountManageDialog({
  open,
  onOpenChange,
  onSaved,
}: AmountDiscountManageDialogProps) {
  const { t } = useTranslation()
  const { data: options, isLoading } = useSystemOptions()
  const updateOption = useUpdateOption()
  const [draft, setDraft] = useState('{}')

  const serverValue =
    options?.data?.find((option) => option.key === OPTION_KEY)?.value ?? '{}'

  // Only resync the draft when the dialog opens, never mid-edit.
  const wasOpen = useRef(false)
  useEffect(() => {
    if (open && !wasOpen.current) {
      setDraft(serverValue)
    }
    wasOpen.current = open
  }, [open, serverValue])

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
      title={t('Top-up amount discount management')}
      description={t(
        'Discount tiers based on the recharge amount, applied to the wallet quick top-up cards.'
      )}
      contentClassName='sm:max-w-[640px]'
      contentHeight='auto'
      bodyClassName='space-y-4'
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
        <AmountDiscountVisualEditor value={draft} onChange={setDraft} />
      )}
    </Dialog>
  )
}
