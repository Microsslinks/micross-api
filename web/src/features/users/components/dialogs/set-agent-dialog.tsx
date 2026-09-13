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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { setUserAsAgent } from '../../api'
import { ERROR_MESSAGES } from '../../constants'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: { id: number; username: string } | null
  onSuccess?: () => void
}

/**
 * 设为经销商。
 *
 * 经销商是叠在用户身上的业务身份，不是权限角色：设完之后这个人还是原来的角色、
 * 还能照常登录消费，只是多了一份经营档案。所以这里刻意没有角色选项。
 *
 * 两个折扣由平台手动设定（业务方案 03-agent-and-commission.md §5）：
 * 批发折扣是平台卖给这位经销商的价格，最低折扣是给他自己定价的地板价。
 */
export function SetAgentDialog({ open, onOpenChange, user, onSuccess }: Props) {
  const { t } = useTranslation()
  const [wholesaleDiscount, setWholesaleDiscount] = useState('1')
  const [minDiscount, setMinDiscount] = useState('0')
  const [remark, setRemark] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      // 默认按官方标价进货，不填也不会算错钱。
      setWholesaleDiscount('1')
      setMinDiscount('0')
      setRemark('')
    }
  }, [open])

  const handleSubmit = async () => {
    if (!user) return
    setSubmitting(true)
    try {
      const result = await setUserAsAgent(user.id, {
        wholesale_discount: wholesaleDiscount,
        min_discount: minDiscount,
        remark,
      })
      if (result.success) {
        toast.success(
          t('{{username}} is now a dealer', { username: user.username })
        )
        onOpenChange(false)
        onSuccess?.()
      } else {
        toast.error(result.message || t('Failed to set dealer'))
      }
    } catch {
      toast.error(t(ERROR_MESSAGES.UNEXPECTED))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        <>
          <Store className='h-5 w-5' />
          {t('Set as Dealer')}
        </>
      }
      description={t(
        'A dealer keeps the current role and can still log in and consume as before. Only the business identity is added.'
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
            {t('Cancel')}
          </Button>
          <Button onClick={handleSubmit} disabled={submitting || !user}>
            {t('Confirm')}
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
          <Label htmlFor='agent-wholesale-discount'>
            {t('Wholesale Discount')}
          </Label>
          <Input
            id='agent-wholesale-discount'
            value={wholesaleDiscount}
            onChange={(e) => setWholesaleDiscount(e.target.value)}
            inputMode='decimal'
            placeholder='1'
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'What the platform charges this dealer. 0.7 means the dealer buys at 70% of the list price.'
            )}
          </p>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='agent-min-discount'>
            {t('Minimum discount')}
          </Label>
          <Input
            id='agent-min-discount'
            value={minDiscount}
            onChange={(e) => setMinDiscount(e.target.value)}
            inputMode='decimal'
            placeholder='0'
          />
          <p className='text-muted-foreground text-xs'>
            {t(
              'The lowest discount this dealer may offer their own customers. It cannot be higher than the wholesale discount.'
            )}
          </p>
        </div>

        <div className='space-y-2'>
          <Label htmlFor='agent-remark'>{t('Remark')}</Label>
          <Input
            id='agent-remark'
            value={remark}
            onChange={(e) => setRemark(e.target.value)}
            maxLength={255}
          />
        </div>
      </div>
    </Dialog>
  )
}
