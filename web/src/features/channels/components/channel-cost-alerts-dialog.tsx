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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTimestampToDate } from '@/lib/format'

import { getStaleChannelCosts } from '../api'
import {
  COST_STALE_DAYS,
  channelCostQueryKeys,
  formatCostRatio,
  handleBatchSetCost,
} from '../lib'
import type { ChannelCostOverviewItem } from '../types'

interface ChannelCostAlertsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * 「进货折扣待补」清单：哪条线路还没录进价、哪条录了太久该核对。
 * 就地补录，省得为改一条线路来回开关编辑抽屉。
 */
export function ChannelCostAlertsDialog({
  open,
  onOpenChange,
}: ChannelCostAlertsDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [drafts, setDrafts] = useState<Record<number, string>>({})

  const alertsQuery = useQuery({
    queryKey: channelCostQueryKeys.stale(COST_STALE_DAYS),
    queryFn: () => getStaleChannelCosts(COST_STALE_DAYS),
    enabled: open,
  })

  const items = alertsQuery.data?.data?.items ?? []
  const days = alertsQuery.data?.data?.days ?? COST_STALE_DAYS

  const handleSave = (channelId: number) => {
    const value = (drafts[channelId] ?? '').trim()
    if (!value) return

    handleBatchSetCost([channelId], value, queryClient, () => {
      setDrafts((previous) => {
        const next = { ...previous }
        delete next[channelId]
        return next
      })
    })
  }

  const renderUpdatedAt = (item: ChannelCostOverviewItem) => {
    if (item.reason === 'unconfigured' || !item.cost_updated_at) {
      return <span className='text-muted-foreground text-xs'>-</span>
    }

    return (
      <span className='text-xs'>
        {formatTimestampToDate(item.cost_updated_at)}
      </span>
    )
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Cost ratio needs attention')}
      description={t(
        'Channels without a cost ratio are skipped when serving customers on a discount, and a cost ratio left unchanged for over {{days}} days may no longer match the upstream price.',
        { days }
      )}
      contentHeight='auto'
      footer={
        <Button variant='outline' onClick={() => onOpenChange(false)}>
          {t('Close')}
        </Button>
      }
    >
      {alertsQuery.isLoading ? (
        <div className='text-muted-foreground flex items-center justify-center gap-2 py-8 text-sm'>
          <Loader2 className='h-4 w-4 animate-spin' />
          {t('Loading...')}
        </div>
      ) : items.length === 0 ? (
        <div className='text-muted-foreground py-8 text-center text-sm'>
          {t('Every channel has a cost ratio on record.')}
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('Name')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Cost Ratio')}</TableHead>
              <TableHead>{t('Last updated')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((item) => (
              <TableRow key={item.id}>
                <TableCell className='max-w-48 truncate'>
                  <span className='font-medium'>{item.name}</span>
                </TableCell>
                <TableCell>
                  {item.reason === 'unconfigured' ? (
                    <StatusBadge
                      label={t('Not set')}
                      variant='danger'
                      size='sm'
                      copyable={false}
                    />
                  ) : (
                    <StatusBadge
                      label={t('Outdated')}
                      variant='warning'
                      size='sm'
                      copyable={false}
                    />
                  )}
                </TableCell>
                <TableCell>
                  {item.cost_ratio ? (
                    <span className='font-mono text-xs'>
                      {formatCostRatio(item.cost_ratio)}
                    </span>
                  ) : (
                    <span className='text-muted-foreground text-xs'>-</span>
                  )}
                </TableCell>
                <TableCell>{renderUpdatedAt(item)}</TableCell>
                <TableCell>
                  <div className='flex items-center justify-end gap-2'>
                    <Input
                      className='h-8 w-24'
                      type='number'
                      min='0'
                      step='0.01'
                      placeholder={t('Not set')}
                      value={drafts[item.id] ?? ''}
                      onChange={(event) =>
                        setDrafts((previous) => ({
                          ...previous,
                          [item.id]: event.target.value,
                        }))
                      }
                    />
                    <Button
                      size='sm'
                      disabled={!(drafts[item.id] ?? '').trim()}
                      onClick={() => handleSave(item.id)}
                    >
                      {t('Save')}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Dialog>
  )
}
