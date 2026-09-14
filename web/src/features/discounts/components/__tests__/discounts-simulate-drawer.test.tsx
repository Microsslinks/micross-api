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

import type { DiscountSimulateResult } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { DiscountsSimulateDrawer } = await import('../discounts-simulate-drawer')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Cost is unknown': 'Cost is unknown',
        'Discount Simulation': 'Discount Simulation',
        'Enter a model name to simulate': 'Enter a model name to simulate',
        'Simulation failed': 'Simulation failed',
        Simulate: 'Simulate',
      },
    },
  },
})

type ApiMethod = (url: string, config?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get

let simulateCalls = 0
let simulatePayload: DiscountSimulateResult

function simulateResult(
  overrides: Partial<DiscountSimulateResult> = {}
): DiscountSimulateResult {
  return {
    user: { id: 7, username: 'alice', group: 'default' },
    model: 'gpt-4o',
    vendor: 'openai',
    plan: { id: 3, name: 'vip', status: 1 },
    resolution: {
      discount: '0.300000',
      source: 'model',
      matched_rule: {
        id: 11,
        scope_type: 'model',
        scope_value: 'gpt-4o',
        discount: '0.300000',
        priority: 10,
      },
    },
    candidates: [],
    cost_known: true,
    min_margin_ratio: '0.100000',
    channels: [
      {
        channel_id: 1,
        channel_name: 'primary',
        cost_ratio: '0.270000',
        gross_margin: '0.111111',
        passes_floor: true,
      },
    ],
    warnings: [],
    ...overrides,
  }
}

function mockApi(): void {
  simulateCalls = 0
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
    if (url.includes('/api/discount/admin/simulate')) {
      simulateCalls += 1
      return { data: { success: true, data: simulatePayload } }
    }
    throw new Error(`unexpected request: ${url}`)
  }
}

async function pickCustomer(): Promise<void> {
  fireEvent.click(screen.getByRole('combobox'))
  const option = await screen.findByRole('option', { name: 'alice (Alice)' })
  fireEvent.click(option)
}

async function renderDrawer(withCustomer = true): Promise<void> {
  render(
    <I18nextProvider i18n={i18n}>
      <DiscountsSimulateDrawer open onOpenChange={() => undefined} />
    </I18nextProvider>
  )
  if (withCustomer) {
    await pickCustomer()
  }
}

function submitSimulation(modelName: string | null = 'gpt-4o'): void {
  if (modelName !== null) {
    fireEvent.input(
      screen.getByPlaceholderText('Enter a model name to simulate'),
      { target: { value: modelName } }
    )
  }
  const form = document.querySelector<HTMLFormElement>(
    '#discount-simulate-form'
  )
  if (!form) throw new Error('Expected the simulation form')
  fireEvent.submit(form)
}

afterEach(() => {
  apiClient.get = originalGet
})

describe('discount simulation drawer', () => {
  test('keeps a missing cost visible instead of reporting zero profit', async () => {
    simulatePayload = simulateResult({
      cost_known: false,
      channels: [
        {
          channel_id: 2,
          channel_name: 'backup',
          cost_ratio: null,
          gross_margin: null,
          passes_floor: null,
        },
      ],
    })
    mockApi()
    await renderDrawer()

    submitSimulation()

    await waitFor(() => {
      expect(screen.getByText('Cost is unknown')).toBeInTheDocument()
    })
    // 进货折扣和毛利两处都必须是「—」，不能变成 0；达标判断另说一句
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText('Not recorded')).toBeInTheDocument()
    expect(screen.queryByText('0.00%')).toBeNull()
  })

  test('renders each route margin and whether it clears the floor', async () => {
    simulatePayload = simulateResult({
      channels: [
        {
          channel_id: 1,
          channel_name: 'primary',
          cost_ratio: '0.270000',
          gross_margin: '0.111111',
          passes_floor: true,
        },
        {
          channel_id: 9,
          channel_name: 'expensive',
          cost_ratio: '0.900000',
          gross_margin: '-0.666667',
          passes_floor: false,
        },
      ],
    })
    mockApi()
    await renderDrawer()

    submitSimulation()

    await waitFor(() => {
      expect(screen.getByText('+11.11%')).toBeInTheDocument()
    })
    expect(screen.getByText('-66.67%')).toBeInTheDocument()
    expect(screen.getByText('Break-even or better')).toBeInTheDocument()
    expect(screen.getByText('Below floor')).toBeInTheDocument()
    expect(screen.queryByText('Cost is unknown')).toBeNull()
  })

  test('shows the backend warnings as written', async () => {
    const warning = '线路「primary」未录入进货折扣，无法判断是否赔本'
    simulatePayload = simulateResult({ warnings: [warning] })
    mockApi()
    await renderDrawer()

    submitSimulation()

    await waitFor(() => {
      expect(screen.getByText(warning)).toBeInTheDocument()
    })
  })

  test('refuses to simulate before a customer is picked', async () => {
    simulatePayload = simulateResult()
    mockApi()
    await renderDrawer(false)

    submitSimulation()

    expect(screen.getByText('Simulation failed')).toBeInTheDocument()
    expect(simulateCalls).toBe(0)
  })
})
