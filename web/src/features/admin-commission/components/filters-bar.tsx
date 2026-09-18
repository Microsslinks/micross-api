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

import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export type ReversedFilter = 'all' | 'yes' | 'no'
export type BreachFilter = 'all' | 'yes' | 'no'

interface FiltersBarProps {
  inviterId: string
  inviteeId: string
  reversed: ReversedFilter
  breach: BreachFilter
  onChange: (next: {
    inviterId: string
    inviteeId: string
    reversed: ReversedFilter
    breach: BreachFilter
  }) => void
  onApply: () => void
  onReset: () => void
}

// ============================================================================
// Filters Bar (task-20 §20.8)
//
// 过滤维度对应 controller/commission.go:AdminListCommissionRecords 接收的
// query string：inviter_id / invitee_id / reversed / breach。
//
// 设计要点：
//   - 用户输入是 string (空 = 不过滤)；提交时 onApply 负责转 number / boolean。
//   - reversed / breach 是 Select，三态 'all' / 'yes' / 'no'。
//   - "应用" 显式触发（不是 onChange 即时刷新），避免输入到一半的非法值触发
//     后端 500；同时减少不必要的接口调用。
// ============================================================================
export function FiltersBar({
  inviterId,
  inviteeId,
  reversed,
  breach,
  onChange,
  onApply,
  onReset,
}: FiltersBarProps) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-wrap items-end gap-3 rounded-md border bg-muted/30 p-3'>
      <div className='flex flex-col gap-1'>
        <span className='text-xs text-muted-foreground'>
          {t('admin-commission.filter.inviter-id', 'inviter_id')}
        </span>
        <Input
          className='w-32'
          placeholder='e.g. 1001'
          value={inviterId}
          onChange={(e) => onChange({ inviterId: e.target.value, inviteeId, reversed, breach })}
        />
      </div>

      <div className='flex flex-col gap-1'>
        <span className='text-xs text-muted-foreground'>
          {t('admin-commission.filter.invitee-id', 'invitee_id')}
        </span>
        <Input
          className='w-32'
          placeholder='e.g. 2002'
          value={inviteeId}
          onChange={(e) => onChange({ inviterId, inviteeId: e.target.value, reversed, breach })}
        />
      </div>

      <div className='flex flex-col gap-1'>
        <span className='text-xs text-muted-foreground'>
          {t('admin-commission.filter.reversed', 'reversed')}
        </span>
        <Select
          value={reversed}
          onValueChange={(v) =>
            onChange({
              inviterId,
              inviteeId,
              reversed: v as ReversedFilter,
              breach,
            })
          }
        >
          <SelectTrigger className='w-32'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='all'>
              {t('admin-commission.filter.all', '全部')}
            </SelectItem>
            <SelectItem value='yes'>
              {t('admin-commission.filter.yes', '已撤销')}
            </SelectItem>
            <SelectItem value='no'>
              {t('admin-commission.filter.no', '未撤销')}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className='flex flex-col gap-1'>
        <span className='text-xs text-muted-foreground'>
          {t('admin-commission.filter.breach', 'breach')}
        </span>
        <Select
          value={breach}
          onValueChange={(v) =>
            onChange({
              inviterId,
              inviteeId,
              reversed,
              breach: v as BreachFilter,
            })
          }
        >
          <SelectTrigger className='w-32'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value='all'>
              {t('admin-commission.filter.all', '全部')}
            </SelectItem>
            <SelectItem value='yes'>breach</SelectItem>
            <SelectItem value='no'>ok</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div className='ml-auto flex gap-2'>
        <Button variant='outline' onClick={onReset}>
          {t('common.reset', '重置')}
        </Button>
        <Button onClick={onApply}>
          {t('common.apply', '应用')}
        </Button>
      </div>
    </div>
  )
}