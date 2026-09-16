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
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import type { UserWalletData } from '@/features/wallet/types'
import { formatQuota } from '@/lib/format'

interface CommissionRewardsCardProps {
  user: UserWalletData | null
  affiliateLink: string
  /**
   * @deprecated commission 永远不可提现（任务文档 §三"资金闭环"），保留
   * props 仅作兼容：wallet/index.tsx 的 AffiliateRewardsCard 调用方仍会传入，
   * 内部永不消费。Phase 3 之后可与 affiliate-rewards-card 一起删除。
   */
  onTransfer?: () => void
  /**
   * @deprecated commission 没有转账操作故无需合规检查；保留 props 仅作兼容。
   */
  complianceConfirmed?: boolean
  loading?: boolean
}

/**
 * 「佣金收益」板块（task-10 P4 佣金核心；task-11 UI 重命名）
 *
 * 原 affiliate-rewards-card.tsx 的 affiliate（一次性注册返利）改为
 * commission（按消费返佣）口径——数据结构、文案、按钮均替换：
 *   1. 数据源：user.commission_balance（独立钱包）+ commission_history（可选）；
 *      aff_count 仍展示（邀请人数）。
 *   2. 文案：Commission Earnings / Pending Commission / Total Commission Earned。
 *   3. 删 Transfer to Balance 按钮——commission 端点永远返回
 *      "commission is not withdrawable"，前端不再展示转账入口。
 *   4. 卡片底部固定提示 "Commission is not withdrawable"。
 *
 * 与 affiliate-rewards-card 的关系：后者已被改写为兼容重导出别名
 * （export { CommissionRewardsCard as AffiliateRewardsCard }），
 * 保证 wallet/index.tsx 老 import 路径仍可工作。
 */
export function CommissionRewardsCard({
  user,
  affiliateLink,
  loading,
}: CommissionRewardsCardProps) {
  const { t } = useTranslation()

  return (
    <TitledCard
      title={t('Commission Earnings')}
      description={t(
        'Earn commission when your invitees consume API quota. The commission wallet is non-withdrawable and can only be spent through API calls.'
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
                [
                  t('Pending Commission'),
                  formatQuota(user?.commission_balance ?? 0),
                ],
                [
                  t('Total Commission Earned'),
                  // commission_history 后端暂未实现，fallback 到 0；
                  // 后续若后端补字段，前端自动生效（Phase 2 types.ts 已声明）
                  formatQuota(user?.commission_history ?? 0),
                ],
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
              {/* 不再有 Transfer to Balance 按钮——commission 永远不可提现。
                  后端端点 /api/user/aff/commission/transfer 永远返回
                  "commission is not withdrawable"，前端不再渲染入口。 */}
            </div>
          </div>

          <p className='text-muted-foreground mt-3 text-xs'>
            {t('Commission is not withdrawable')}
          </p>
        </>
      )}
    </TitledCard>
  )
}