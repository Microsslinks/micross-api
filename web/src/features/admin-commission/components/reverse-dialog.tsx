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

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import type { AdminCommissionRecord } from '../types'

interface ReverseDialogProps {
  record: AdminCommissionRecord | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: (recordId: number, reason: string) => Promise<void>
  busy?: boolean
}

/**
 * 撤销确认对话框（task-20 §20.8）。
 *
 * - reason 必填：服务端的 service.ReverseCommission 兜底为空字符串，
 *   但运营人写的"风控扫描发现返佣为薅羊毛产出"是 reviewer 必看的审计痕迹，
 *   这里强制非空。
 * - 提交时调 onConfirm(recordId, reason)；父组件负责调 API 并按状态码刷新列表。
 * - 不可撤销的记录 (reversed=true) 在父层就被屏蔽 —— 这里再次防御性检查，
 *   如果用户传进来的是已撤销记录，按钮 disabled + 文案明确告知原因。
 */
export function ReverseDialog({
  record,
  open,
  onOpenChange,
  onConfirm,
  busy = false,
}: ReverseDialogProps) {
  const { t } = useTranslation()
  const [reason, setReason] = useState('')

  // 每次开窗时清空 reason —— 防止上一笔的 reason 串到下一笔。
  useEffect(() => {
    if (open) setReason('')
  }, [open])

  const trimmed = reason.trim()
  const canSubmit = trimmed !== '' && !busy && record != null && !record.reversed

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {t('admin-commission.reverse.title', '撤销返佣记录')}
          </DialogTitle>
          <DialogDescription>
            {t(
              'admin-commission.reverse.description',
              '该操作将从邀请人钱包扣回返佣金额，并写入 commission_reverse ledger 记录。已撤销的记录不可二次撤销。',
            )}
          </DialogDescription>
        </DialogHeader>

        {record && (
          <div className='rounded-md border bg-muted/30 p-3 text-sm'>
            <div className='grid grid-cols-2 gap-2'>
              <span className='text-muted-foreground'>record_id</span>
              <span className='font-mono'>#{record.id}</span>
              <span className='text-muted-foreground'>inviter_id</span>
              <span className='font-mono'>#{record.inviter_id}</span>
              <span className='text-muted-foreground'>invitee_id</span>
              <span className='font-mono'>#{record.invitee_id}</span>
              <span className='text-muted-foreground'>amount</span>
              <span className='font-mono'>{record.amount}</span>
              <span className='text-muted-foreground'>breach</span>
              <span>{record.breach ? 'true' : 'false'}</span>
            </div>
            {record.reversed && (
              <div className='mt-2 text-xs text-destructive'>
                {t(
                  'admin-commission.reverse.already-reversed',
                  '该记录已被撤销：reason =',
                )}{' '}
                {record.reverse_reason || '(empty)'}
              </div>
            )}
          </div>
        )}

        <div className='grid gap-2'>
          <Label htmlFor='reverse-reason'>
            {t('admin-commission.reverse.reason-label', '撤销原因（必填）')}
          </Label>
          <Input
            id='reverse-reason'
            placeholder='e.g. 风控扫描发现返佣为薅羊毛产出'
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            disabled={busy || (record?.reversed ?? false)}
          />
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={busy}
          >
            {t('common.cancel', '取消')}
          </Button>
          <Button
            variant='destructive'
            disabled={!canSubmit}
            onClick={() => record && onConfirm(record.id, trimmed)}
          >
            {busy
              ? t('admin-commission.reverse.submitting', '撤销中...')
              : t('admin-commission.reverse.submit', '确认撤销')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}