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
import { useEffect, useState } from 'react'
import { afterEach, describe, expect, test } from 'vitest'

import type { DiscountPlan, DiscountValidateResult } from '../../types'

const i18n = (await import('i18next')).default
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { api } = await import('@/lib/api')
const { DiscountsProvider, useDiscounts } =
  await import('../discounts-provider')
const { DiscountsPlanMutateDrawer } =
  await import('../discounts-plan-mutate-drawer')

await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Discount is below the plan minimum':
          'Discount is below the plan minimum',
        'Edit plan': 'Edit plan',
        'Profit floor:': 'Profit floor:',
        Save: 'Save',
        'Save anyway': 'Save anyway',
        'Saving is not blocked. Fix the problems below, or save anyway if the loss is intentional.':
          'Saving is not blocked.',
        'This plan may lose money': 'This plan may lose money',
      },
    },
  },
})

type ApiMethod = (url: string, body?: unknown) => Promise<{ data: unknown }>
type MockableApi = {
  post: ApiMethod
  put: ApiMethod
}

const apiClient = api as unknown as MockableApi
const originalPost = apiClient.post
const originalPut = apiClient.put

const plan: DiscountPlan = {
  id: 3,
  name: 'Acme annual contract',
  owner_type: 'platform',
  owner_id: 0,
  base_discount: '0.900000',
  min_discount: '0.500000',
  billing_mode: 'usage',
  commission_ratio: '0.000000',
  topup_conversion_rate: '1.000000',
  status: 1,
  remark: '',
  created_at: 0,
  updated_at: 0,
}

let validateCalls = 0
let updateCalls = 0
let validatePayload: DiscountValidateResult

function validateResult(
  overrides: Partial<DiscountValidateResult> = {}
): DiscountValidateResult {
  return {
    passed: true,
    min_margin_ratio: '0.100000',
    violations: [],
    warnings: [],
    ...overrides,
  }
}

function mockApi(): void {
  validateCalls = 0
  updateCalls = 0
  apiClient.post = async (url: string) => {
    if (url.includes('/validate')) {
      validateCalls += 1
      return { data: { success: true, data: validatePayload } }
    }
    throw new Error(`unexpected request: ${url}`)
  }
  apiClient.put = async () => {
    updateCalls += 1
    return { data: { success: true, data: plan } }
  }
}

/** 抽屉的「当前方案」来自页面 Provider，测试里也用同一条路把它塞进去。 */
function Harness() {
  const { setCurrentRow } = useDiscounts()
  const [ready, setReady] = useState(false)

  useEffect(() => {
    setCurrentRow(plan)
    setReady(true)
  }, [setCurrentRow])

  if (!ready) return null

  return (
    <DiscountsPlanMutateDrawer open isUpdate onOpenChange={() => undefined} />
  )
}

async function renderDrawer(): Promise<void> {
  render(
    <I18nextProvider i18n={i18n}>
      <DiscountsProvider>
        <Harness />
      </DiscountsProvider>
    </I18nextProvider>
  )
  await waitFor(() => {
    expect(document.querySelector('#discount-plan-form')).not.toBeNull()
  })
}

function submitForm(): void {
  const form = document.querySelector<HTMLFormElement>('#discount-plan-form')
  if (!form) throw new Error('Expected the plan form')
  fireEvent.submit(form)
}

afterEach(() => {
  apiClient.post = originalPost
  apiClient.put = originalPut
})

describe('discount plan drawer', () => {
  test('shows the loss before saving and only saves on a second click', async () => {
    // 「规则折扣低于方案最低折扣」已随 rule.Discount 一起废弃——
    // 校验不再逐条规则比一遍，只剩 cost_breach / no_channel / min_above_base 三档。
    // 这里用 cost_breach 演示「先看到亏损提示，确认后再保存」的二次确认流程。
    validatePayload = validateResult({
      passed: false,
      violations: [
        {
          scope_type: 'model',
          scope_value: 'gpt-4o',
          discount: '0.900000',
          reason: 'cost_breach',
          detail: '最低可用进货折扣 0.950000，高于毛利底线 0.900000',
          available_channels: ['primary'],
        },
      ],
    })
    mockApi()
    await renderDrawer()

    submitForm()

    await waitFor(() => {
      expect(screen.getByText('This plan may lose money')).toBeInTheDocument()
    })
    // 第一次点保存只做检查，不许直接写库
    expect(validateCalls).toBe(1)
    expect(updateCalls).toBe(0)
    expect(
      screen.getByText('Cost exceeds the profit floor')
    ).toBeInTheDocument()
    expect(
      screen.getByText('最低可用进货折扣 0.950000，高于毛利底线 0.900000')
    ).toBeInTheDocument()
    expect(screen.getByText('Save anyway')).toBeInTheDocument()

    submitForm()

    await waitFor(() => {
      expect(updateCalls).toBe(1)
    })
  })

  test('saves straight away when the check finds nothing', async () => {
    validatePayload = validateResult()
    mockApi()
    await renderDrawer()

    submitForm()

    await waitFor(() => {
      expect(updateCalls).toBe(1)
    })
    expect(validateCalls).toBe(1)
    expect(screen.queryByText('This plan may lose money')).toBeNull()
  })
})
