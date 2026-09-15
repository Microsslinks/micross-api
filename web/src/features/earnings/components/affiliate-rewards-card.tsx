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
import { TrendingUp } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import type { UserWalletData } from '@/features/wallet/types'
import { formatQuota } from '@/lib/format'

interface AffiliateRewardsCardProps {
  user: UserWalletData | null
  affiliateLink: string
  onTransfer: () => void
  complianceConfirmed?: boolean
  loading?: boolean
}

/**
 * 「推荐收益」板块
 *
 * 原先是独立的 /earnings 页面，但内容只有一行推广链接，
 * 现在挂在钱包页最下面，与余额 / 充值共用同一页。
 */
export function AffiliateRewardsCard({
  user,
  affiliateLink,
  onTransfer,
  complianceConfirmed = true,
  loading,
}: AffiliateRewardsCardProps) {
  const { t } = useTranslation()

  const hasRewards = (user?.aff_quota ?? 0) > 0

  return (
    <TitledCard
      title={t('Referral Earnings')}
      description={t(
        'Earn rewards when users join through your referral link. Transfer accumulated rewards to your balance anytime.'
      )}
      icon={<TrendingUp className='h-4 w-4' />}
      iconTone='chart-3'
      disableHoverEffect
    >
      {loading ? (
        <div className='flex flex-col gap-3 lg:flex-row lg:items-center'>
          <Skeleton className='h-10 flex-1' />
          <Skeleton className='h-9 w-72' />
        </div>
      ) : (
        <>
          <div className='flex flex-col gap-3 lg:flex-row lg:items-center'>
            <div className='grid grid-cols-3 gap-1.5 text-center lg:w-72 lg:shrink-0'>
              {[
                [t('Pending'), formatQuota(user?.aff_quota ?? 0)],
                [t('Total Earned'), formatQuota(user?.aff_history_quota ?? 0)],
                [t('Invites'), String(user?.aff_count ?? 0)],
              ].map(([label, value]) => (
                <div key={label}>
                  <div className='text-muted-foreground truncate text-[10px] font-medium tracking-wider uppercase'>
                    {label}
                  </div>
                  <div className='mt-0.5 truncate text-sm font-semibold tabular-nums'>
                    {value}
                  </div>
                </div>
              ))}
            </div>

            <div className='flex min-w-0 flex-1 items-center gap-2'>
              <Input
                value={affiliateLink}
                readOnly
                className='border-muted bg-background/70 h-9 min-w-0 flex-1 font-mono text-xs'
              />
              <CopyButton
                value={affiliateLink}
                variant='outline'
                className='bg-background size-9 shrink-0'
                iconClassName='size-4'
                tooltip={t('Copy referral link')}
                aria-label={t('Copy referral link')}
              />
              {hasRewards && (
                <Button
                  onClick={onTransfer}
                  disabled={!complianceConfirmed}
                  className='h-9 shrink-0 px-3'
                  size='sm'
                >
                  {t('Transfer to Balance')}
                </Button>
              )}
            </div>
          </div>
          {!complianceConfirmed ? (
            <p className='text-muted-foreground mt-3 text-xs'>
              {t(
                'Referral reward transfer is disabled until the administrator confirms compliance terms.'
              )}
            </p>
          ) : null}
        </>
      )}
    </TitledCard>
  )
}
