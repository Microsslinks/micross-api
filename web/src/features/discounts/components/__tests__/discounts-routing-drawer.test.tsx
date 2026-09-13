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

import type { DiscountRouting, DiscountRoutingPayload } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { DiscountsRoutingDrawer } =
  await import('../discounts-routing-drawer')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        Allowed: 'Allowed',
        'Allow cost breach': 'Allow cost breach',
        'Highest margin first': 'Highest margin first',
        'Not allowed': 'Not allowed',
        'Reset to default': 'Reset to default',
        'Routing Policy': 'Routing Policy',
        'Routing Strategy': 'Routing Strategy',
        Save: 'Save',
        'Select a customer': 'Select a customer',
        'Setting the routing policy failed': 'Setting the routing policy failed',
        'Stable route first': 'Stable route first',
        'This customer has its own routing policy; other customers use the default.':
          'This customer has its own routing policy; other customers use the default.',
        'This customer uses the default routing policy and has no row of its own.':
          'This customer uses the default routing policy and has no row of its own.',
        'Why this customer is special, e.g. the ticket number':
          'Why this customer is special, e.g. the ticket number',
      },
    },
  },
})

type ApiMethod = (url: string, body?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  get: ApiMethod
  put: ApiMethod
  delete: ApiMethod
}

const apiClient = api as unknown as MockableApi
const originalGet = apiClient.get
const originalPut = apiClient.put
const originalDelete = apiClient.delete

let routingReads = 0
let saveCalls = 0
let resetCalls = 0
let lastSavePayload: DiscountRoutingPayload | null = null
let routingPayload: DiscountRouting

function routing(overrides: Partial<DiscountRouting> = {}): DiscountRouting {
  return {
    user_id: 7,
    username: 'alice',
    routing_strategy: 'margin',
    allow_cost_breach: false,
    remark: '',
    updated_by: 0,
    updated_at: 0,
    configured: false,
    ...overrides,
  }
}

function mockApi(): void {
  routingReads = 0
  saveCalls = 0
  resetCalls = 0
  lastSavePayload = null
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
    if (url.includes('/api/discount/admin/routing')) {
      routingReads += 1
      return { data: { success: true, data: routingPayload } }
    }
    throw new Error(`unexpected request: ${url}`)
  }
  apiClient.put = async (url: string, body?: unknown) => {
    if (!url.includes('/api/discount/admin/routing')) {
      throw new Error(`unexpected request: ${url}`)
    }
    saveCalls += 1
    lastSavePayload = body as DiscountRoutingPayload
    return { data: { success: true, data: null } }
  }
  apiClient.delete = async (url: string) => {
    if (!url.includes('/api/discount/admin/routing')) {
      throw new Error(`unexpected request: ${url}`)
    }
    resetCalls += 1
    return { data: { success: true, data: null } }
  }
}

/** 这一颗药丸现在是不是"选中"状态。断言的是管理员眼睛看到的东西，不是内部 state。 */
function isPressed(label: string): boolean {
  const button = screen.getByText(label).closest('button')
  if (!button) throw new Error(`Expected a toggle button labelled ${label}`)
  return (
    button.getAttribute('aria-pressed') === 'true' ||
    button.hasAttribute('data-pressed')
  )
}

function resetButton(): HTMLElement {
  const button = screen.getByText('Reset to default').closest('button')
  if (!button) throw new Error('Expected the reset button')
  return button
}

async function pickCustomer(): Promise<void> {
  fireEvent.click(screen.getByRole('combobox'))
  const option = await screen.findByRole('option', { name: 'alice (Alice)' })
  fireEvent.click(option)
  // 选完客户会去读他当前的口径；等它落地再操作，免得点和回读互相盖。
  await waitFor(() => {
    expect(routingReads).toBe(1)
  })
}

async function renderDrawer(withCustomer = true): Promise<void> {
  render(
    <I18nextProvider i18n={i18n}>
      <DiscountsRoutingDrawer open onOpenChange={() => undefined} />
    </I18nextProvider>
  )
  if (withCustomer) {
    await pickCustomer()
  }
}

function submitForm(): void {
  const form = document.querySelector<HTMLFormElement>(
    '#discount-routing-form'
  )
  if (!form) throw new Error('Expected the routing form')
  fireEvent.submit(form)
}

afterEach(() => {
  apiClient.get = originalGet
  apiClient.put = originalPut
  apiClient.delete = originalDelete
})

describe('discount routing drawer', () => {
  test('sends the loss switch as a boolean and defaults to not allowed', async () => {
    routingPayload = routing()
    mockApi()
    await renderDrawer()

    fireEvent.input(
      screen.getByPlaceholderText(
        'Why this customer is special, e.g. the ticket number'
      ),
      { target: { value: '  ticket 123  ' } }
    )
    submitForm()

    await waitFor(() => {
      expect(saveCalls).toBe(1)
    })
    // 「没配过」= 按默认口径存：毛利优先 + 不允许走亏损线路
    expect(lastSavePayload).toEqual({
      user_id: 7,
      routing_strategy: 'margin',
      allow_cost_breach: false,
      remark: 'ticket 123',
    })

    fireEvent.click(screen.getByText('Allowed'))
    submitForm()

    await waitFor(() => {
      expect(saveCalls).toBe(2)
    })
    // 这个开关是能让人赔钱的，落库必须是布尔 true，不能变成字符串或空值
    expect(lastSavePayload?.allow_cost_breach).toBe(true)
  })

  test('shows the policy the backend stored, not the one just submitted', async () => {
    routingPayload = routing({ configured: true })
    mockApi()
    await renderDrawer()

    fireEvent.click(screen.getByText('Allowed'))
    await waitFor(() => {
      expect(isPressed('Allowed')).toBe(true)
    })
    submitForm()

    await waitFor(() => {
      expect(saveCalls).toBe(1)
    })
    expect(lastSavePayload?.allow_cost_breach).toBe(true)

    // 后端没认这个开关（回读仍是 false），界面必须跟着回到「不允许」，
    // 否则管理员会以为放行了，实际没放行。
    await waitFor(() => {
      expect(routingReads).toBe(2)
    })
    await waitFor(() => {
      expect(isPressed('Not allowed')).toBe(true)
    })
    expect(isPressed('Allowed')).toBe(false)

    submitForm()

    await waitFor(() => {
      expect(saveCalls).toBe(2)
    })
    expect(lastSavePayload?.allow_cost_breach).toBe(false)
  })

  test('refuses to save before a customer is picked', async () => {
    routingPayload = routing()
    mockApi()
    await renderDrawer(false)

    // 没选客户就不该出现策略开关：这时候它调的是谁的口径都说不清
    expect(screen.queryByText('Routing Strategy')).toBeNull()
    expect(screen.queryByText('Allow cost breach')).toBeNull()
    expect(resetButton()).toBeDisabled()

    submitForm()

    expect(screen.getByText('Setting the routing policy failed')).toBeInTheDocument()
    const alert = document.querySelector('[role="alert"]')
    expect(alert?.textContent ?? '').toContain('Select a customer')
    expect(saveCalls).toBe(0)
  })

  test('leaves reset disabled and says so when the customer has no row of its own', async () => {
    routingPayload = routing()
    mockApi()
    await renderDrawer()

    expect(resetButton()).toBeDisabled()
    expect(
      screen.getByText(
        'This customer uses the default routing policy and has no row of its own.'
      )
    ).toBeInTheDocument()
  })

  test('resets a configured customer back to the default policy', async () => {
    routingPayload = routing({
      configured: true,
      routing_strategy: 'priority',
      allow_cost_breach: true,
      updated_at: 1700000000,
    })
    mockApi()
    await renderDrawer()

    await waitFor(() => {
      expect(isPressed('Stable route first')).toBe(true)
    })
    expect(isPressed('Allowed')).toBe(true)
    expect(resetButton()).toBeEnabled()

    // 管理员点了重置：后端把这一行删掉，界面必须回到默认口径
    routingPayload = routing()
    fireEvent.click(resetButton())

    await waitFor(() => {
      expect(resetCalls).toBe(1)
    })
    await waitFor(() => {
      expect(
        screen.getByText(
          'This customer uses the default routing policy and has no row of its own.'
        )
      ).toBeInTheDocument()
    })
    expect(isPressed('Highest margin first')).toBe(true)
    expect(isPressed('Not allowed')).toBe(true)
    expect(resetButton()).toBeDisabled()
  })

  test('keeps the strategy when the pressed option is clicked again', async () => {
    routingPayload = routing({ configured: true, routing_strategy: 'priority' })
    mockApi()
    await renderDrawer()

    await waitFor(() => {
      expect(isPressed('Stable route first')).toBe(true)
    })

    // 再点一次已选中的那项：单选组不该被点空，也不该偷偷换成另一项
    fireEvent.click(screen.getByText('Stable route first'))
    submitForm()

    await waitFor(() => {
      expect(saveCalls).toBe(1)
    })
    expect(lastSavePayload?.routing_strategy).toBe('priority')

    fireEvent.click(screen.getByText('Highest margin first'))
    submitForm()

    await waitFor(() => {
      expect(saveCalls).toBe(2)
    })
    expect(lastSavePayload?.routing_strategy).toBe('margin')
  })
})
