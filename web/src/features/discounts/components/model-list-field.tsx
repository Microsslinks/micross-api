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
import { BookmarkPlus } from 'lucide-react'
import { type KeyboardEvent, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

import { useDiscountModelLists } from '../hooks/use-discount-model-lists'
import { useModelNameOptions } from '../hooks/use-model-name-options'
import { mergeModelNames, splitModelList } from '../lib'
import type { DiscountModelList } from '../types'

type ModelListFieldProps = {
  value: string[]
  onChange: (next: string[]) => void
  /** 一次能用几个模型（报价核算与查价的入参上界）。 */
  max: number
  id?: string
  disabled?: boolean
  /**
   * 是否给「从已存清单带入」这个按钮。
   * 默认给；只有一个地方关掉它——「模型清单」维护页自己的那份表单，
   * 在编辑清单的地方再摆一个「从清单带入」容易让人以为是在复制另一份清单。
   */
  allowImport?: boolean
}

/**
 * 模型清单零件：把「一组模型名」当成一个可复用的输入。
 *
 * 两条路都能走，因为它们各自解决不同的问题：
 * - 上面的标签框：从平台认识的模型里挑，看得见清单里有几个、点掉写错的那个。
 * - 下面的粘贴框：整份清单（换行、逗号、顿号分隔都行）一次贴进来。
 *   `<input>` 会把粘贴的多行压成一行，所以整份粘贴必须有一个多行框接着。
 *
 * 超出上限时截断到上限并在下面写清丢掉了几个——不静默丢，也不假装收下了。
 */
export function ModelListField({
  value,
  onChange,
  max,
  id,
  disabled = false,
  allowImport = true,
}: ModelListFieldProps) {
  const { t } = useTranslation()
  const { options, isCatalogUnavailable } = useModelNameOptions()
  const { lists } = useDiscountModelLists()
  const [draft, setDraft] = useState('')
  const [limitNotice, setLimitNotice] = useState<string | null>(null)
  const [importNotice, setImportNotice] = useState<string | null>(null)

  const catalog = useMemo(
    () => new Set(options.map((option) => option.value)),
    [options]
  )
  // 目录拿不到时不标「不在目录里」——那会把清单里每个模型都标一遍，纯属噪音。
  const unknownNames = catalog.size
    ? value.filter((name) => !catalog.has(name))
    : []

  const commit = (next: string[]) => {
    const merged = mergeModelNames([], next)
    const kept = merged.slice(0, max)
    const dropped = merged.length - kept.length
    onChange(kept)
    setLimitNotice(
      dropped > 0
        ? t('Kept the first {{max}} models; {{dropped}} were dropped.', {
            max,
            dropped,
          })
        : null
    )
  }

  const addDraft = () => {
    const names = splitModelList(draft)
    if (names.length === 0) {
      setDraft('')
      return
    }
    commit(mergeModelNames(value, names))
    setDraft('')
  }

  const handleDraftKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    // 贴完清单习惯性按回车换行，所以整份提交用 Ctrl/Cmd + Enter，避免误触。
    if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
      event.preventDefault()
      addDraft()
    }
  }

  /**
   * 从一份存好的清单往里带：并到现有清单后面，不清空已经挑好的。
   * 已经有的模型不重复加，所以「一个都没加进来」也得说清，
   * 不然会让人以为按钮没反应。
   */
  const importList = (list: DiscountModelList) => {
    const incoming = list.models ?? []
    const merged = mergeModelNames(value, incoming)
    const added = merged.length - value.length
    commit(merged)
    setImportNotice(
      added > 0
        ? t('Imported {{count}} models from "{{name}}".', {
            count: added,
            name: list.name,
          })
        : t('All models in "{{name}}" are already in the list.', {
            name: list.name,
          })
    )
  }

  return (
    <div className='space-y-2'>
      <MultiSelect
        id={id}
        options={options}
        selected={value}
        onChange={commit}
        allowCreate
        disabled={disabled}
        maxVisibleChips={12}
        placeholder={t('Select models or type one and press Enter')}
      />

      {allowImport && lists.length > 0 && (
        <div className='flex flex-wrap items-center gap-2'>
          <Select
            items={lists.map((list) => ({
              value: String(list.id),
              label: list.name,
            }))}
            value=''
            disabled={disabled}
            onValueChange={(value) => {
              const picked = lists.find((list) => String(list.id) === value)
              if (picked) importList(picked)
            }}
          >
            <SelectTrigger
              className='w-[260px]'
              aria-label={t('Import from a saved list')}
            >
              <BookmarkPlus className='h-4 w-4' />
              <SelectValue placeholder={t('Import from a saved list')} />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                {lists.map((list) => (
                  <SelectItem key={list.id} value={String(list.id)}>
                    {list.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {importNotice && (
            <span className='text-muted-foreground text-xs'>
              {importNotice}
            </span>
          )}
        </div>
      )}

      <div className='flex items-start gap-2'>
        <Textarea
          value={draft}
          rows={3}
          disabled={disabled}
          placeholder={t(
            'Paste the whole list here: one per line, or separated by commas.'
          )}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={handleDraftKeyDown}
        />
        <Button
          type='button'
          variant='outline'
          disabled={disabled || draft.trim().length === 0}
          onClick={addDraft}
        >
          {t('Add to the list')}
        </Button>
      </div>

      <div className='text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-xs'>
        <span>
          {value.length === 0
            ? t('No models selected yet')
            : t('{{count}} models', { count: value.length })}
        </span>
        <span>{t('At most {{max}} models.', { max })}</span>
        {value.length > 0 && !disabled && (
          <Button
            type='button'
            variant='link'
            className='h-auto p-0 text-xs'
            onClick={() => commit([])}
          >
            {t('Clear the list')}
          </Button>
        )}
      </div>

      {limitNotice && <p className='text-destructive text-xs'>{limitNotice}</p>}
      {unknownNames.length > 0 && (
        <p className='text-xs text-amber-600 dark:text-amber-500'>
          {t('Not in the model catalog: {{models}}', {
            models: unknownNames.join(', '),
          })}
        </p>
      )}
      {isCatalogUnavailable && (
        <p className='text-muted-foreground text-xs'>
          {t(
            'The model catalog is unavailable; type or paste names as before.'
          )}
        </p>
      )}
    </div>
  )
}
