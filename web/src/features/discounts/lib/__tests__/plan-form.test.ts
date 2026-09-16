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
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import { DISCOUNT_OWNER, type DiscountPlan } from '../../types'
import {
  buildDiscountPlanPayload,
  getDiscountPlanFormSchema,
  transformDiscountPlanToFormDefaults,
  type DiscountPlanFormValues,
} from '../plan-form'
import { normalizeRatioInput, ratioTextToInput } from '../ratio'

// 只测「拦不拦、拦哪一项」，所以直接把 key 当文案用，不搭 i18n。
const t = ((key: string) => key) as unknown as TFunction

function formValues(
  overrides: Partial<DiscountPlanFormValues> = {}
): DiscountPlanFormValues {
  return {
    name: 'Acme annual contract',
    owner_type: DISCOUNT_OWNER.PLATFORM,
    owner_id: 0,
    base_discount: '0.9',
    min_discount: '0.5',
    billing_mode: 'usage',
    commission_ratio: '0',
    topup_conversion_rate: '1',
    status: 1,
    remark: '',
    ...overrides,
  }
}

function issuePaths(values: DiscountPlanFormValues): string[] {
  const parsed = getDiscountPlanFormSchema(t).safeParse(values)
  if (parsed.success) return []
  return parsed.error.issues.map((issue) => String(issue.path[0] ?? ''))
}

describe('normalizeRatioInput', () => {
  test('turns what the operator typed into the six decimal string the backend stores', () => {
    expect(normalizeRatioInput('0.3')).toBe('0.300000')
    expect(normalizeRatioInput(' 1 ')).toBe('1.000000')
    expect(normalizeRatioInput('0.1234567')).toBe('0.123457')
  })

  test('refuses what the backend would refuse instead of guessing a number', () => {
    // 折扣 0 等于免费、>1 等于加价，都不能被悄悄放行或被改成别的数
    expect(normalizeRatioInput('0')).toBeNull()
    expect(normalizeRatioInput('1.2')).toBeNull()
    expect(normalizeRatioInput('-0.5')).toBeNull()
    expect(normalizeRatioInput('')).toBeNull()
    expect(normalizeRatioInput('abc')).toBeNull()
    expect(normalizeRatioInput('0', { allowZero: true })).toBe('0.000000')
  })

  test('shows a stored ratio back without trailing zeros', () => {
    expect(ratioTextToInput('1.000000')).toBe('1')
    expect(ratioTextToInput('0.300000')).toBe('0.3')
    expect(ratioTextToInput(null)).toBe('')
  })
})

describe('discount plan form', () => {
  test('rejects a minimum discount higher than the base discount', () => {
    expect(
      issuePaths(formValues({ base_discount: '0.5', min_discount: '0.8' }))
    ).toContain('min_discount')
  })

  test('rejects a zero base discount and a name that is only spaces', () => {
    expect(issuePaths(formValues({ base_discount: '0' }))).toContain(
      'base_discount'
    )
    expect(issuePaths(formValues({ name: '   ' }))).toContain('name')
  })

  test('an agent owned plan needs the agent user id', () => {
    expect(
      issuePaths(formValues({ owner_type: DISCOUNT_OWNER.AGENT, owner_id: 0 }))
    ).toContain('owner_id')
    expect(
      issuePaths(formValues({ owner_type: DISCOUNT_OWNER.AGENT, owner_id: 9 }))
    ).toEqual([])
  })

  test('builds the payload with six decimals and clears the agent id for platform plans', () => {
    expect(
      buildDiscountPlanPayload(
        formValues({ name: '  Acme  ', owner_id: 12, min_discount: '0' })
      )
    ).toEqual({
      name: 'Acme',
      owner_type: DISCOUNT_OWNER.PLATFORM,
      owner_id: 0,
      base_discount: '0.900000',
      min_discount: '0.000000',
      billing_mode: 'usage',
      commission_ratio: '0.000000',
      status: 1,
      remark: '',
    })
  })

  test('fills the form from a saved plan', () => {
    const plan: DiscountPlan = {
      id: 3,
      name: 'Acme annual contract',
      owner_type: DISCOUNT_OWNER.AGENT,
      owner_id: 9,
      base_discount: '1.000000',
      min_discount: '0.500000',
      billing_mode: 'subscription',
      commission_ratio: '0.100000',
      topup_conversion_rate: '1.000000',
      status: 1,
      remark: 'asked by sales',
      created_at: 0,
      updated_at: 0,
    }

    expect(transformDiscountPlanToFormDefaults(plan)).toEqual({
      name: 'Acme annual contract',
      owner_type: DISCOUNT_OWNER.AGENT,
      owner_id: 9,
      base_discount: '1',
      min_discount: '0.5',
      billing_mode: 'subscription',
      commission_ratio: '0.1',
      status: 1,
      remark: 'asked by sales',
    })
  })
})
