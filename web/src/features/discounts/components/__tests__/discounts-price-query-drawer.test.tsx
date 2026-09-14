import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
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
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, test } from 'vitest'

import type { CustomerPriceQueryResult } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { DiscountsPriceQueryDrawer } =
  await import('../discounts-price-query-drawer')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Customer Price Check': 'Customer Price Check',
        'Price lookup failed': 'Price lookup failed',
        'Look up prices': 'Look up prices',
        Applied: 'Applied',
        'Show candidates': 'Show candidates',
      },
    },
  },
})

type ApiMethod = (url: string, config?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
  post: ApiMethod
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPost = apiClient.post

let queryCalls: Array<{ userId: number; models: string[] }> = []
let queryPayload: CustomerPriceQueryResult

function priceQueryResult(
  overrides: Partial<CustomerPriceQueryResult> = {}
): CustomerPriceQueryResult {
  return {
    user: { id: 7, username: 'alice', group: 'default' },
    min_margin_ratio: '0.100000',
    price_book: [
      {
        binding_id: 1,
        plan_id: 3,
        plan_name: 'vip plan',
        plan_status: 1,
        source: 'manual',
        billing_mode: 'usage',
        base_discount: '0.950000',
        effective_from: 0,
        effective_to: 0,
        in_window: true,
        window_reason: '',
        rules: [
          {
            scope_type: 'model',
            scope_value: 'gpt-4o',
            discount: '0.800000',
            priority: 10,
            status: 1,
          },
        ],
      },
      {
        binding_id: 2,
        plan_id: 4,
        plan_name: 'dealer plan',
        plan_status: 1,
        source: 'agent',
        billing_mode: 'usage',
        base_discount: '0.900000',
        effective_from: 0,
        effective_to: 0,
        in_window: false,
        window_reason: '尚未到生效时间',
        rules: [],
      },
    ],
    models: [
      {
        model: 'gpt-4o',
        vendor: 'openai',
        discount: '0.800000',
        source: 'model',
        plan: { id: 3, name: 'vip plan', status: 1 },
        matched_rule: {
          id: 11,
          scope_type: 'model',
          scope_value: 'gpt-4o',
          discount: '0.800000',
          priority: 10,
        },
        channel_count: 1,
        usable_count: 1,
        cheapest_cost: '0.700000',
        gross_margin: '0.142857',
        verdict: 'ok',
        verdict_detail: '每条线路都过底线，赚 14.29%',
        candidates: [
          {
            plan_id: 3,
            plan_name: 'vip plan',
            source: 'manual',
            specificity: 'model',
            discount: '0.800000',
            applied: true,
            rejected: false,
            reject_reason: '',
          },
          {
            plan_id: 4,
            plan_name: 'dealer plan',
            source: 'agent',
            specificity: 'plan_base',
            discount: '0.900000',
            applied: false,
            rejected: true,
            reject_reason: '折扣不如已选方案：0.9 比 0.8 贵',
          },
        ],
      },
    ],
    summary: {
      total: 1,
      ok: 1,
      loss: 0,
      unknown_cost: 0,
      no_channel: 0,
      signable: true,
      conclusion: '这份报价可以签',
    },
    warnings: [],
    ...overrides,
  }
}

function mockApi(): void {
  queryCalls = []
  apiClient.get = async (url: string) => {
    if (url.includes('/api/user/search')) {
      return {
        data: {
          success: true,
          data: {
            items: [
              {
                id: 7,
                username: 'alice',
                display_name: 'Alice',
                group: 'default',
              },
            ],
            total: 1,
            page: 1,
            page_size: 50,
          },
        },
      }
    }
    if (url.includes('/api/channel/models_enabled')) {
      return {
        data: {
          success: true,
          data: ['gpt-4o', 'gpt-4.1'],
        },
      }
    }
    throw new Error(`unexpected request: ${url}`)
  }
  apiClient.post = async (url: string, config?: unknown) => {
    if (url.includes('/api/discount/admin/price-query')) {
      const body = config as { user_id: number; models: string[] }
      queryCalls.push({ userId: body.user_id, models: body.models })
      return { data: { success: true, data: queryPayload } }
    }
    throw new Error(`unexpected request: ${url}`)
  }
}

async function pickCustomer(): Promise<void> {
  // 客户下拉在前、模型清单的输入框在后，所以第一个 combobox 是客户。
  fireEvent.click(screen.getAllByRole('combobox')[0])
  const option = await screen.findByRole('option', { name: 'alice (Alice)' })
  fireEvent.click(option)
}

async function renderDrawer(withCustomer = true): Promise<void> {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  render(
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <DiscountsPriceQueryDrawer open onOpenChange={() => undefined} />
      </I18nextProvider>
    </QueryClientProvider>
  )
  if (withCustomer) {
    await pickCustomer()
  }
}

/** 走模型清单零件：整份贴进粘贴框，再点「加入清单」并进已选。 */
function addToList(models: string): void {
  fireEvent.input(
    screen.getByPlaceholderText(
      'Paste the whole list here: one per line, or separated by commas.'
    ),
    { target: { value: models } }
  )
  fireEvent.click(screen.getByRole('button', { name: 'Add to the list' }))
}

function submitLookup(modelList: string | null = 'gpt-4o'): void {
  if (modelList !== null) {
    addToList(modelList)
  }
  const form = document.querySelector<HTMLFormElement>(
    '#discount-price-query-form'
  )
  if (!form) throw new Error('Expected the price query form')
  fireEvent.submit(form)
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.post = originalPost
})

describe('discount price query drawer', () => {
  test('shows the price book as soon as a customer is picked', async () => {
    queryPayload = priceQueryResult()
    mockApi()
    await renderDrawer()

    // 选客户就带着空模型清单问一次：先看身上有几套价，不用先想要问哪个模型。
    await waitFor(() => {
      expect(queryCalls).toEqual([{ userId: 7, models: [] }])
    })
    // 价目本里两套价都在（第二套没生效，也照样列出来）。
    expect((await screen.findAllByText('vip plan')).length).toBeGreaterThan(0)
    expect(screen.getByText('dealer plan')).toBeInTheDocument()
    // 没生效的价也要说清为什么，不然运营没法回答「上周谈的价怎么没生效」。
    expect(screen.getByText('尚未到生效时间')).toBeInTheDocument()
    // 没有模型级/厂商级规则时要说清基础折扣管全部模型，不能留空白让人猜。
    expect(
      screen.getByText('No rules: the base discount covers every model.')
    ).toBeInTheDocument()
  })

  test('lists the candidate that lost together with its reason', async () => {
    queryPayload = priceQueryResult()
    mockApi()
    await renderDrawer()

    // 整份清单贴进粘贴框、点「加入清单」后，模型名以标签形式进已选清单。
    addToList('gpt-4o')
    expect(screen.getAllByText('gpt-4o').length).toBeGreaterThan(0)

    submitLookup(null)

    await waitFor(() => {
      expect(queryCalls.at(-1)).toEqual({ userId: 7, models: ['gpt-4o'] })
    })
    // 现价、逐模型毛利与一句话结论都在。
    expect(await screen.findByText('+14.29%')).toBeInTheDocument()
    expect(screen.getByText('这份报价可以签')).toBeInTheDocument()

    // 展开候选明细：赢的那套打「已生效」，输的那套必须带原因。
    fireEvent.click(screen.getByRole('button', { name: 'Show candidates' }))
    expect(await screen.findByText('Applied')).toBeInTheDocument()
    expect(
      screen.getByText('折扣不如已选方案：0.9 比 0.8 贵')
    ).toBeInTheDocument()
  })

  test('refuses to look up before a customer is picked', async () => {
    queryPayload = priceQueryResult()
    mockApi()
    await renderDrawer(false)

    submitLookup()

    expect(screen.getByText('Price lookup failed')).toBeInTheDocument()
    expect(queryCalls).toHaveLength(0)
  })
})
