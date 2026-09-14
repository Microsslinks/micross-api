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
import { z } from 'zod'

import {
  DISCOUNT_PLAN_STATUS,
  DISCOUNT_SCOPE,
  type DiscountRule,
  type DiscountRulePayload,
} from '../types'

export type DiscountRuleFormValues = {
  scope_type: string
  scope_value: string
  priority: number
  status: number
}

export const DISCOUNT_RULE_SCOPE_VALUE_MAX_LENGTH = 128

export const DISCOUNT_RULE_FORM_DEFAULT_VALUES: DiscountRuleFormValues = {
  scope_type: DISCOUNT_SCOPE.MODEL,
  scope_value: '',
  priority: 0,
  status: DISCOUNT_PLAN_STATUS.ENABLED,
}

export function getDiscountRuleFormSchema(t: TFunction) {
  return z.object({
    scope_type: z.enum([DISCOUNT_SCOPE.MODEL, DISCOUNT_SCOPE.VENDOR]),
    scope_value: z
      .string()
      .trim()
      .min(1, t('Scope value is required'))
      .max(
        DISCOUNT_RULE_SCOPE_VALUE_MAX_LENGTH,
        t('Scope value must be at most 128 characters')
      ),
    // rule.Discount 字段已废弃：规则改为纯范围标记，命中时按方案基础折扣出价。
    // 表单不再接收折扣输入，命中率仍然由 priority 决定。
    priority: z
      .number()
      .int()
      .min(0, t('Priority must be a non-negative whole number')),
    status: z.union([
      z.literal(DISCOUNT_PLAN_STATUS.ENABLED),
      z.literal(DISCOUNT_PLAN_STATUS.DISABLED),
    ]),
  })
}

export function buildDiscountRulePayload(
  values: DiscountRuleFormValues
): DiscountRulePayload {
  return {
    scope_type: values.scope_type,
    scope_value: values.scope_value.trim(),
    priority: values.priority,
    status: values.status,
  }
}

export function transformDiscountRuleToFormDefaults(
  rule: DiscountRule
): DiscountRuleFormValues {
  return {
    scope_type: rule.scope_type || DISCOUNT_SCOPE.MODEL,
    scope_value: rule.scope_value ?? '',
    priority: rule.priority ?? 0,
    status: rule.status,
  }
}