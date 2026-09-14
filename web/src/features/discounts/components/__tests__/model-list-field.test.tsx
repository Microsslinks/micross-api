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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, test } from 'vitest'

import type { DiscountModelList } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { ModelListField } = await import('../model-list-field')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        '{{count}} models': '{{count}} models',
        'No models selected yet': 'No models selected yet',
        'At most {{max}} models.': 'At most {{max}} models.',
        'Clear the list': 'Clear the list',
        'Add to the list': 'Add to the list',
        'Import from a saved list': 'Import from a saved list',
        'Select models or type one and press Enter':
          'Select models or type one and press Enter',
        'Paste the whole list here: one per line, or separated by commas.':
          'Paste the whole list here: one per line, or separated by commas.',
        'Kept the first {{max}} models; {{dropped}} were dropped.':
          'Kept the first {{max}} models; {{dropped}} were dropped.',
        'Imported {{count}} models from "{{name}}".':
          'Imported {{count}} models from "{{name}}".',
        'All models in "{{name}}" are already in the list.':
          'All models in "{{name}}" are already in the list.',
      },
    },
  },
})

type ApiMethod = (url: string, config?: unknown) => Promise<{ data: unknown }>
type MockableApi = { get: ApiMethod }

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get

let savedLists: DiscountModelList[] = []

function modelList(
  overrides: Partial<DiscountModelList> = {}
): DiscountModelList {
  return {
    id: 1,
    name: '企业VIP标准包',
    remark: '',
    models: ['gpt-4o', 'claude-3-5-sonnet', 'gemini-2.0-flash'],
    created_at: 1,
    updated_at: 2,
    ...overrides,
  }
}

function mockApi(): void {
  apiClient.get = async (url: string) => {
    if (url.includes('/api/discount/admin/model-lists')) {
      return {
        data: {
          success: true,
          data: {
            items: savedLists,
            total: savedLists.length,
            page: 1,
            page_size: 100,
          },
        },
      }
    }
    if (url.includes('/api/channel/models_enabled')) {
      return { data: { success: true, data: ['gpt-4o'] } }
    }
    throw new Error(`unexpected request: ${url}`)
  }
}

/** 把当前清单揉进界面：断言的是运营眼睛看到的东西，不是内部 state。 */
function Harness({ initial }: { initial: string[] }) {
  const [value, setValue] = useState(initial)
  return <ModelListField value={value} onChange={setValue} max={100} />
}

async function renderField(initial: string[] = []): Promise<void> {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <Harness initial={initial} />
      </I18nextProvider>
    </QueryClientProvider>
  )
  if (savedLists.length > 0) {
    // 等清单读完再操作：不然「带入」那个下拉可能还没出现
    await screen.findByText('Import from a saved list')
  }
}

/** 打开「从已存清单带入」并挑一份。 */
async function importFrom(name: string): Promise<void> {
  fireEvent.click(
    screen.getByRole('combobox', { name: 'Import from a saved list' })
  )
  const option = await screen.findByRole('option', { name })
  fireEvent.click(option)
}

afterEach(() => {
  apiClient.get = originalGet
})

describe('model list field', () => {
  test('offers no import control when nothing is saved', async () => {
    savedLists = []
    mockApi()
    await renderField()

    // 一份清单都没存过时不该出现这个下拉——点开是空的只会让人以为坏了
    expect(
      screen.queryByRole('combobox', { name: 'Import from a saved list' })
    ).toBeNull()
    expect(screen.getByText('No models selected yet')).toBeInTheDocument()
  })

  test('imports a saved list on top of what is already picked', async () => {
    savedLists = [modelList()]
    mockApi()
    await renderField(['gpt-4o'])

    await importFrom('企业VIP标准包')

    // 已经在清单里的 gpt-4o 不该再来一遍，新来的两个并到后面
    expect(screen.getByText('3 models')).toBeInTheDocument()
    expect(
      screen.getByText('Imported 2 models from "企业VIP标准包".')
    ).toBeInTheDocument()
  })

  test('says so when the saved list brought nothing new', async () => {
    savedLists = [modelList({ models: ['gpt-4o'] })]
    mockApi()
    await renderField(['gpt-4o'])

    await importFrom('企业VIP标准包')

    // 一个都没加进来也必须说清，不然会以为按钮没反应
    expect(
      screen.getByText('All models in "企业VIP标准包" are already in the list.')
    ).toBeInTheDocument()
    expect(screen.getByText('1 models')).toBeInTheDocument()
  })

  test('still truncates to the cap and says how many were dropped', async () => {
    savedLists = [modelList({ models: ['a', 'b', 'c'] })]
    mockApi()
    // 上限 2 个：带入 3 个必须留下「丢了几个」的交代
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <ModelListField value={[]} onChange={() => undefined} max={2} />
        </I18nextProvider>
      </QueryClientProvider>
    )
    await screen.findByText('Import from a saved list')

    await importFrom('企业VIP标准包')

    expect(
      screen.getByText('Kept the first 2 models; 1 were dropped.')
    ).toBeInTheDocument()
  })
})
