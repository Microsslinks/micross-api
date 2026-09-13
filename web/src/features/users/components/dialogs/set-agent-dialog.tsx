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
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { getAgentWholesaleQuote, setUserAsAgent } from '../../api'
import { ERROR_MESSAGES } from '../../constants'
import type { AgentWholesaleQuoteItem } from '../../types'

/** 界面上的三档毛利。填哪个都行，也可以直接改数字，但不得低于下限。 */
const MARKUP_PRESETS = [5, 10, 20]
/** 平台毛利的下限，与后端 NormalizeAgentMarkup 里的 1.05 是同一条线。 */
const MIN_MARKUP_PERCENT = 5
const DEFAULT_MARKUP_PERCENT = 10
/** 改一次数字就拉一次价目表，防抖一下，别每敲一个字符都打后端。 */
const QUOTE_DEBOUNCE_MS = 300

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
 * 价格只填一档毛利。逐模型填折扣那条路走不通：上游给我们的价每个模型都不一样
 * （同一个模型还分好几条线路），填一个统一的折扣数字只能表达一个现实中不存在的价。
 * 所以这里只定「在成本上加几个点」，每个模型的拿货价由系统各自算出来，
 * 下面那张清单就是把算出来的结果摊开给他看——他要拿这些数字去给客户报价。
 */
export function SetAgentDialog({ open, onOpenChange, user, onSuccess }: Props) {
  const { t } = useTranslation()
  const [markupPercent, setMarkupPercent] = useState(
    String(DEFAULT_MARKUP_PERCENT)
  )
  const [remark, setRemark] = useState('')
  const [items, setItems] = useState<AgentWholesaleQuoteItem[]>([])
  const [quoting, setQuoting] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (open) {
      setMarkupPercent(String(DEFAULT_MARKUP_PERCENT))
      setRemark('')
      setItems([])
    }
  }, [open])

  const markupPercentValue = Number(markupPercent)
  const markupValid =
    markupPercent.trim() !== '' &&
    Number.isFinite(markupPercentValue) &&
    markupPercentValue >= MIN_MARKUP_PERCENT
  const markupRatio = markupValid
    ? (1 + markupPercentValue / 100).toFixed(6)
    : ''

  useEffect(() => {
    if (!open || !user || !markupRatio) {
      setItems([])
      return
    }
    let cancelled = false
    setQuoting(true)
    const timer = setTimeout(async () => {
      try {
        const result = await getAgentWholesaleQuote(user.id, markupRatio)
        if (cancelled) return
        setItems(result.success ? (result.data?.items ?? []) : [])
      } catch {
        if (!cancelled) setItems([])
      } finally {
        if (!cancelled) setQuoting(false)
      }
    }, QUOTE_DEBOUNCE_MS)
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [open, user, markupRatio])

  const handleSubmit = async () => {
    if (!user || !markupValid) return
    setSubmitting(true)
    try {
      const result = await setUserAsAgent(user.id, {
        markup_ratio: markupRatio,
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

  const chips = useMemo(
    () =>
      items.map((item) => (
        <span
          key={item.model_name}
          title={`${item.channel_name} · ${t('Cost')} ${item.cost_ratio}`}
          className='bg-muted inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs'
        >
          <span className='font-medium'>{item.model_name}</span>
          <span className='text-muted-foreground'>{item.discount}</span>
        </span>
      )),
    [items, t]
  )

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
      contentClassName='sm:max-w-lg'
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
          <Button
            onClick={handleSubmit}
            disabled={submitting || !user || !markupValid}
          >
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
          <Label htmlFor='agent-markup'>{t('Platform markup')}</Label>
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
              id='agent-markup'
              className='w-24'
              value={markupPercent}
              onChange={(e) => setMarkupPercent(e.target.value)}
              inputMode='decimal'
              placeholder={String(DEFAULT_MARKUP_PERCENT)}
            />
            <span className='text-muted-foreground text-sm'>%</span>
          </div>
          <p className='text-muted-foreground text-xs'>
            {markupValid
              ? t(
                  'What the platform adds on top of its own purchase cost for this dealer. Each model is priced from its own cost, so the result differs per model.'
                )
              : t('Markup cannot be lower than 5%.')}
          </p>
        </div>

        <div className='space-y-2'>
          <Label>{t('His purchase price by model')}</Label>
          {quoting ? (
            <p className='text-muted-foreground text-xs'>{t('Loading')}</p>
          ) : chips.length === 0 ? (
            <p className='text-muted-foreground text-xs'>
              {t(
                'No model can be priced right now. Record the purchase discount on the upstream channels first.'
              )}
            </p>
          ) : (
            <div className='flex max-h-40 flex-wrap gap-1.5 overflow-y-auto rounded-md border p-2'>
              {chips}
            </div>
          )}
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
