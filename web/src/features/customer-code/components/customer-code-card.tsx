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
import { Loader2, Ticket } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { TitledCard } from '@/components/ui/titled-card'

import { bindCustomerCode, getMyAgentBinding } from '../api'
import type { CustomerBinding } from '../types'

/**
 * 客户号卡片：客户在这里填经销商给他的号，绑完就归到那位经销商名下、按号上的折扣计价。
 *
 * 同时把「我现在挂在谁名下、按什么价」摊开显示——客户号是别人给的，自己看不到归属和
 * 折扣就容易怀疑钱被多扣了。归属一旦落下只能由平台改，所以这里只有绑，没有解绑。
 */
export function CustomerCodeCard() {
  const { t } = useTranslation()
  const [binding, setBinding] = useState<CustomerBinding | null>(null)
  const [loading, setLoading] = useState(true)
  const [code, setCode] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const load = useCallback(async () => {
    try {
      const response = await getMyAgentBinding()
      setBinding(response.success && response.data ? response.data : null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const handleBind = async () => {
    const value = code.trim()
    if (!value) return
    setSubmitting(true)
    try {
      const response = await bindCustomerCode(value)
      if (response.success && response.data) {
        setBinding(response.data)
        setCode('')
        toast.success(t('Customer code bound'))
      } else {
        toast.error(response.message || t('Failed to bind the customer code'))
      }
    } catch {
      toast.error(t('Failed to bind the customer code'))
    } finally {
      setSubmitting(false)
    }
  }

  const hasAgent = (binding?.parent_agent_id ?? 0) > 0
  const hasPlan = (binding?.plan_id ?? 0) > 0

  return (
    <TitledCard
      title={t('Customer code')}
      description={t('Belong to a dealer and use the price he gave you')}
      icon={<Ticket className='h-4 w-4' />}
      iconTone='chart-4'
      disableHoverEffect
    >
      {loading ? (
        <div className='text-muted-foreground flex items-center gap-2 text-sm'>
          <Loader2 className='size-4 animate-spin' />
          {t('Loading')}
        </div>
      ) : (
        <div className='space-y-4'>
          <div className='grid gap-3 sm:grid-cols-2'>
            <div className='rounded-lg border p-3'>
              <div className='text-muted-foreground text-xs'>
                {t('Your dealer')}
              </div>
              <div className='text-sm font-medium'>
                {hasAgent
                  ? binding?.agent_name ||
                    `${t('Dealer')} #${binding?.parent_agent_id}`
                  : t('Direct under the platform')}
              </div>
            </div>
            <div className='rounded-lg border p-3'>
              <div className='text-muted-foreground text-xs'>
                {t('Your discount')}
              </div>
              <div className='text-sm font-medium'>
                {hasPlan
                  ? `${binding?.plan_name} · ${binding?.plan_discount}`
                  : t('Official list price')}
              </div>
            </div>
          </div>

          <div className='space-y-2'>
            <div className='flex items-center gap-2'>
              <Input
                value={code}
                placeholder={t('Customer code')}
                aria-label={t('Customer code')}
                className='max-w-56 font-mono'
                maxLength={32}
                onChange={(event) => setCode(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') void handleBind()
                }}
              />
              <Button
                onClick={handleBind}
                disabled={submitting || code.trim() === ''}
              >
                {submitting ? t('Binding') : t('Bind')}
              </Button>
            </div>
            <p className='text-muted-foreground text-xs'>
              {t(
                'Ask the dealer for a customer code. Binding records you under that dealer and applies the discount on the code; only the platform can change it afterwards.'
              )}
            </p>
          </div>
        </div>
      )}
    </TitledCard>
  )
}
