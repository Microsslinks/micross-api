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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

import type { DiscountModelList, DiscountModelListPayload } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { DiscountsModelListsDrawer } =
  await import('../discounts-model-lists-drawer')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Model Lists': 'Model Lists',
        'List name': 'List name',
        'New list': 'New list',
        'Saved lists': 'Saved lists',
        'Create the list': 'Create the list',
        'Failed to save the model list': 'Failed to save the model list',
        'The list name is required': 'The list name is required',
        'Model list is required': 'Model list is required',
        Edit: 'Edit',
        Delete: 'Delete',
        'Deleting...': 'Deleting...',
        'Are you sure?': 'Are you sure?',
        'No saved lists yet.': 'No saved lists yet.',
        '{{count}} models': '{{count}} models',
        'Model List': 'Model List',
        Remark: 'Remark',
        'Select models or type one and press Enter':
          'Select models or type one and press Enter',
        'Paste the whole list here: one per line, or separated by commas.':
          'Paste the whole list here: one per line, or separated by commas.',
        'Add to the list': 'Add to the list',
        'No models selected yet': 'No models selected yet',
        'At most {{max}} models.': 'At most {{max}} models.',
        'Clear the list': 'Clear the list',
        'Import from a saved list': 'Import from a saved list',
      },
    },
  },
})

type ApiMethod = (url: string, body?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
  post: ApiMethod
  put: ApiMethod
  delete: ApiMethod
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPost = apiClient.post
const originalPut = apiClient.put
const originalDelete = apiClient.delete

let modelLists: DiscountModelList[] = []
let createCalls: DiscountModelListPayload[] = []
let updateCalls: Array<{ id: number; payload: DiscountModelListPayload }> = []
let deleteCalls: number[] = []
/** 想让后端拒一次（重名之类）时把它设成后端要说的话。 */
let createRejection: string | null = null

function modelList(
  overrides: Partial<DiscountModelList> = {}
): DiscountModelList {
  return {
    id: 1,
    name: '企业VIP标准包',
    remark: '2026-03 签的',
    models: ['gpt-4o', 'claude-3-5-sonnet', 'gemini-2.0-flash'],
    created_at: 1,
    updated_at: 2,
    ...overrides,
  }
}

function mockApi(): void {
  createCalls = []
  updateCalls = []
  deleteCalls = []
  createRejection = null
  apiClient.get = async (url: string) => {
    if (url.includes('/api/discount/admin/model-lists')) {
      return {
        data: {
          success: true,
          data: {
            items: modelLists,
            total: modelLists.length,
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
  apiClient.post = async (url: string, body?: unknown) => {
    if (!url.includes('/api/discount/admin/model-lists')) {
      throw new Error(`unexpected request: ${url}`)
    }
    createCalls.push(body as DiscountModelListPayload)
    if (createRejection) {
      return { data: { success: false, message: createRejection } }
    }
    const created = modelList({
      id: 99,
      ...(body as DiscountModelListPayload),
    })
    modelLists = [created, ...modelLists]
    return { data: { success: true, data: created } }
  }
  apiClient.put = async (url: string, body?: unknown) => {
    const match = /\/model-lists\/(\d+)$/.exec(url)
    if (!match) {
      throw new Error(`unexpected request: ${url}`)
    }
    const id = Number.parseInt(match[1], 10)
    updateCalls.push({ id, payload: body as DiscountModelListPayload })
    return { data: { success: true, data: modelList({ id }) } }
  }
  apiClient.delete = async (url: string) => {
    const match = /\/model-lists\/(\d+)$/.exec(url)
    if (!match) {
      throw new Error(`unexpected request: ${url}`)
    }
    const id = Number.parseInt(match[1], 10)
    deleteCalls.push(id)
    modelLists = modelLists.filter((list) => list.id !== id)
    return { data: { success: true, data: null } }
  }
}

async function renderDrawer(): Promise<void> {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <DiscountsModelListsDrawer open onOpenChange={() => undefined} />
      </I18nextProvider>
    </QueryClientProvider>
  )
  // 等清单落地再操作，免得点和回读互相盖。
  if (modelLists.length > 0) {
    await screen.findByText(modelLists[0].name)
  }
}

function submitForm(): void {
  const form = document.querySelector<HTMLFormElement>(
    '#discount-model-list-form'
  )
  if (!form) throw new Error('Expected the model list form')
  fireEvent.submit(form)
}

/** 走一遍「粘贴名单 → 加进清单」这条运营最常走的路。 */
function pasteModels(raw: string): void {
  fireEvent.input(
    screen.getByPlaceholderText(
      'Paste the whole list here: one per line, or separated by commas.'
    ),
    { target: { value: raw } }
  )
  fireEvent.click(screen.getByText('Add to the list'))
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.post = originalPost
  apiClient.put = originalPut
  apiClient.delete = originalDelete
})

describe('model lists drawer', () => {
  test('lists what is saved, with the model count', async () => {
    modelLists = [
      modelList(),
      modelList({ id: 2, name: '测试小包', remark: '', models: ['gpt-4o'] }),
    ]
    mockApi()
    await renderDrawer()

    expect(screen.getByText('企业VIP标准包')).toBeInTheDocument()
    expect(screen.getByText('测试小包')).toBeInTheDocument()
    expect(screen.getByText('3 models')).toBeInTheDocument()
    expect(screen.getByText('1 models')).toBeInTheDocument()
    // 名单预览：前三个点名列出来
    expect(
      screen.getByText('gpt-4o, claude-3-5-sonnet, gemini-2.0-flash')
    ).toBeInTheDocument()
  })

  test('creates a list and shows it in the table right away', async () => {
    modelLists = []
    mockApi()
    await renderDrawer()

    expect(await screen.findByText('No saved lists yet.')).toBeInTheDocument()

    fireEvent.input(screen.getByLabelText('List name'), {
      target: { value: '  企业VIP标准包  ' },
    })
    fireEvent.input(screen.getByLabelText('Remark'), {
      target: { value: '2026-03 签的' },
    })
    pasteModels('gpt-4o\nclaude-3-5-sonnet\ngpt-4o')
    submitForm()

    await waitFor(() => {
      expect(createCalls).toHaveLength(1)
    })
    // 名称两头空格不要，名单去重、保持录入顺序
    expect(createCalls[0]).toEqual({
      name: '企业VIP标准包',
      remark: '2026-03 签的',
      models: ['gpt-4o', 'claude-3-5-sonnet'],
    })

    // 存完要能在表里看见——不然运营不敢确定到底存进去没有
    expect(await screen.findByText('企业VIP标准包')).toBeInTheDocument()
    expect(screen.getByText('2 models')).toBeInTheDocument()
    // 存完表单回到新建态，可以直接接着录下一份
    expect(screen.getByLabelText('List name')).toHaveValue('')
  })

  test('edits the list picked in the table', async () => {
    modelLists = [
      modelList(),
      modelList({ id: 2, name: '测试小包', models: ['gpt-4o'] }),
    ]
    mockApi()
    await renderDrawer()

    // 点第二行的编辑：表单该切到「测试小包」，而不是第一份
    fireEvent.click(screen.getAllByLabelText('Edit')[1])
    expect(screen.getByText('Editing: 测试小包')).toBeInTheDocument()
    expect(screen.getByLabelText('List name')).toHaveValue('测试小包')

    fireEvent.input(screen.getByLabelText('List name'), {
      target: { value: '测试小包（改）' },
    })
    pasteModels('gemini-2.0-flash')
    submitForm()

    await waitFor(() => {
      expect(updateCalls).toHaveLength(1)
    })
    expect(updateCalls[0]).toEqual({
      id: 2,
      payload: {
        name: '测试小包（改）',
        // 备注跟着这一行的原值带过来，不用人重填
        remark: '2026-03 签的',
        models: ['gpt-4o', 'gemini-2.0-flash'],
      },
    })
    // 改的是第二份，第一份必须原样不动
    expect(screen.getByText('企业VIP标准包')).toBeInTheDocument()
  })

  test('keeps the backend reason when the name is taken', async () => {
    modelLists = [modelList()]
    mockApi()
    createRejection = '已存在同名清单'
    await renderDrawer()

    fireEvent.input(screen.getByLabelText('List name'), {
      target: { value: '企业VIP标准包' },
    })
    pasteModels('gpt-4o')
    submitForm()

    await waitFor(() => {
      expect(createCalls).toHaveLength(1)
    })
    // 后端说的话要原样给运营看，不能换成一句笼统的「保存失败」
    expect(await screen.findByText('已存在同名清单')).toBeInTheDocument()
    expect(
      screen.getByText('Failed to save the model list')
    ).toBeInTheDocument()
    // 输入不能被清掉，改个名就能接着存
    expect(screen.getByLabelText('List name')).toHaveValue('企业VIP标准包')
  })

  test('refuses to save a list with no name or no models', async () => {
    modelLists = []
    mockApi()
    await renderDrawer()

    pasteModels('gpt-4o')
    submitForm()
    expect(
      await screen.findByText('The list name is required')
    ).toBeInTheDocument()

    fireEvent.input(screen.getByLabelText('List name'), {
      target: { value: '空名单' },
    })
    // 把刚加进去的模型清掉，再存：空名单不该进库
    fireEvent.click(screen.getByText('Clear the list'))
    submitForm()
    expect(
      await screen.findByText('Model list is required')
    ).toBeInTheDocument()

    expect(createCalls).toHaveLength(0)
  })

  test('asks before deleting, and deletes the row that was clicked', async () => {
    modelLists = [
      modelList(),
      modelList({ id: 2, name: '测试小包', models: ['gpt-4o'] }),
    ]
    mockApi()
    await renderDrawer()

    fireEvent.click(screen.getAllByLabelText('Delete')[1])
    expect(await screen.findByText('Are you sure?')).toBeInTheDocument()
    // 删之前连名字都要说清，免得删错那一份
    expect(
      screen.getByText(/This deletes the model list 测试小包\./)
    ).toBeInTheDocument()

    fireEvent.click(screen.getByText('Delete'))

    await waitFor(() => {
      expect(deleteCalls).toEqual([2])
    })
    await waitFor(() => {
      expect(screen.queryByText('测试小包')).toBeNull()
    })
    expect(screen.getByText('企业VIP标准包')).toBeInTheDocument()
  })
})
