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
import { Activity, BarChart3, Key, Percent, WalletCards } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { SectionPageLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { StatusBadge } from '@/components/status-badge'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { API_KEY_STATUSES } from '@/features/keys/constants'
import {
  formatCompactNumber,
  formatPercent,
  formatQuota,
  formatTimestamp,
} from '@/lib/format'

import { getAgentLedger } from './api'
import type { AgentLedger } from './types'

/**
 * 经销商台账 —— 只给经销商看。
 *
 * 看的是他自己这一侧的账：钱包还剩多少，发出去的每个 Key 用了多少、花了多少。
 * 「花了多少」是平台按拿货价从他钱包里扣掉的额度，不是他卖给客户收了多少钱——
 * 那件事平台不参与也不知道，所以这一页没有那个数。
 *
 * 数据来自 GET /api/user/self/agent/ledger（自己看自己，不用管理员权限）；
 * 菜单项由 use-sidebar-data.ts 按 subject_type 决定是否出现，路由上还挡了一道。
 */
export function DealerBilling() {
  const { t } = useTranslation()
  const [ledger, setLedger] = useState<AgentLedger | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchLedger = useCallback(async () => {
    try {
      const response = await getAgentLedger()
      if (response.success && response.data) {
        setLedger(response.data)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchLedger()
  }, [fetchLedger])

  // 加价率是乘数（1.1 = 在他拿货价上加 10%），换成百分数才看得懂。
  // 值缺了或不是数字时 formatPercent 会给 '-'。
  const markupPercent = formatPercent(
    (Number.parseFloat(ledger?.markup_ratio ?? '') - 1) * 100
  )

  const stats: {
    label: string
    value: string
    description: string
    icon: typeof WalletCards
    tone: IconBadgeTone
  }[] = [
    {
      label: t('Current Balance'),
      value: formatQuota(ledger?.quota ?? 0),
      description: t('Remaining quota'),
      icon: WalletCards,
      tone: 'success',
    },
    {
      label: t('Total Usage'),
      value: formatQuota(ledger?.used_quota ?? 0),
      description: t('Total consumed quota'),
      icon: BarChart3,
      tone: 'info',
    },
    {
      label: t('API Requests'),
      value: (ledger?.request_count ?? 0).toLocaleString(),
      description: t('Total requests made'),
      icon: Activity,
      tone: 'chart-4',
    },
  ]

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Dealer Billing')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {loading ? (
          <LoadingState size='lg' />
        ) : !ledger ? (
          <EmptyState size='lg' />
        ) : (
          <div className='flex w-full flex-col gap-4'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Your balance, and what each of your keys has used and spent.'
              )}
            </p>

            <div className='text-muted-foreground flex items-center gap-1.5 text-sm'>
              <Percent className='size-3.5' />
              <span>{t('Platform markup')}</span>
              <span className='text-foreground font-medium tabular-nums'>
                {markupPercent}
              </span>
            </div>

            <div className='grid grid-cols-3 divide-x rounded-lg border'>
              {stats.map((item) => (
                <div
                  key={item.label}
                  className='min-w-0 px-2.5 py-2.5 sm:px-5 sm:py-4'
                >
                  <div className='flex items-center gap-1.5 sm:gap-2.5'>
                    <IconBadge tone={item.tone} size='stat'>
                      <item.icon />
                    </IconBadge>
                    <div className='text-muted-foreground truncate text-[11px] font-medium tracking-wider uppercase sm:text-xs'>
                      {item.label}
                    </div>
                  </div>

                  <div className='text-foreground mt-1.5 font-mono text-sm font-bold tracking-tight break-all tabular-nums sm:mt-2.5 sm:text-2xl'>
                    {item.value}
                  </div>
                  <div className='text-muted-foreground/60 mt-1 hidden text-xs md:block'>
                    {item.description}
                  </div>
                </div>
              ))}
            </div>

            {ledger.keys.length === 0 ? (
              <EmptyState
                icon={Key}
                title={t('No API Keys Found')}
                description={t(
                  'No API keys available. Create your first API key to get started.'
                )}
                size='lg'
              />
            ) : (
              <div className='overflow-hidden rounded-lg border'>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('Name')}</TableHead>
                      <TableHead>{t('Status')}</TableHead>
                      <TableHead className='text-right'>
                        {t('Used Quota')}
                      </TableHead>
                      <TableHead className='text-right'>
                        {t('Requests')}
                      </TableHead>
                      <TableHead className='text-right'>
                        {t('Tokens')}
                      </TableHead>
                      <TableHead className='text-right'>
                        {t('Last Used')}
                      </TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {ledger.keys.map((item) => {
                      const status = API_KEY_STATUSES[item.status]
                      return (
                        <TableRow key={item.token_id}>
                          <TableCell className='font-medium'>
                            {item.name || `#${item.token_id}`}
                          </TableCell>
                          <TableCell>
                            {item.removed ? (
                              <StatusBadge
                                label={t('This key was removed')}
                                variant='neutral'
                              />
                            ) : status ? (
                              <StatusBadge
                                label={t(status.label)}
                                variant={status.variant}
                              />
                            ) : (
                              '-'
                            )}
                          </TableCell>
                          <TableCell className='text-right font-mono tabular-nums'>
                            {formatQuota(item.used_quota)}
                          </TableCell>
                          <TableCell className='text-right font-mono tabular-nums'>
                            {formatCompactNumber(item.request_count)}
                          </TableCell>
                          <TableCell className='text-right font-mono tabular-nums'>
                            {formatCompactNumber(item.total_tokens)}
                          </TableCell>
                          <TableCell className='text-muted-foreground text-right'>
                            {formatTimestamp(item.last_used_at)}
                          </TableCell>
                        </TableRow>
                      )
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        )}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
