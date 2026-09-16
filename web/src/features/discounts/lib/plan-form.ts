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
  DISCOUNT_BILLING_MODE,
  DISCOUNT_OWNER,
  DISCOUNT_PLAN_STATUS,
  type DiscountPlan,
  type DiscountPlanPayload,
} from '../types'
import { normalizeRatioInput, ratioTextToInput } from './ratio'

/**
 * 方案表单用的都是字符串：折扣在界面上就是「0.3」这样手填的文字，
 * 交给 `normalizeRatioInput` 判合格与否，不合格就停在红字上，
 * 不能在前端悄悄替运营改成别的数。
 */
export type DiscountPlanFormValues = {
  name: string
  owner_type: string
  owner_id: number
  base_discount: string
  min_discount: string
  billing_mode: string
  commission_ratio: string
  /**
   * 经销商给客户发额度时的折算比例（task-09）；输入框里填的就是「0.875」这种。
   * 校验与 base_discount 同口径：>0 且 ≤1，0 等于白送、>1 等于倒贴。
   * 空值视为 1.0（保持 1:1，向后兼容）。
   */
  topup_conversion_rate: string
  status: number
  remark: string
}

export const DISCOUNT_PLAN_NAME_MAX_LENGTH = 64
export const DISCOUNT_PLAN_REMARK_MAX_LENGTH = 255

export const DISCOUNT_PLAN_FORM_DEFAULT_VALUES: DiscountPlanFormValues = {
  name: '',
  owner_type: DISCOUNT_OWNER.PLATFORM,
  owner_id: 0,
  // 后端缺省口径：基础折扣留空即「无折扣」（1.000000），最低折扣 0。
  base_discount: '1',
  min_discount: '0',
  billing_mode: DISCOUNT_BILLING_MODE.USAGE,
  commission_ratio: '0',
  // 经销商发额度折算比例缺省 1.0（保持 1:1，task-09）。空值也走 1.0，向后兼容。
  topup_conversion_rate: '1',
  status: DISCOUNT_PLAN_STATUS.ENABLED,
  remark: '',
}

/** 折扣本身：必须大于 0 且不超过 1（与后端 NormalizeDiscount 同一口径）。 */
function isDiscountInput(raw: string): boolean {
  return normalizeRatioInput(raw) !== null
}

/** 比例类（最低折扣、分成比例）：允许 0，上限 1。 */
function isRatioInput(raw: string): boolean {
  return normalizeRatioInput(raw, { allowZero: true }) !== null
}

export function getDiscountPlanFormSchema(t: TFunction) {
  return z
    .object({
      name: z
        .string()
        .trim()
        .min(1, t('Plan name is required'))
        .max(
          DISCOUNT_PLAN_NAME_MAX_LENGTH,
          t('Plan name must be at most 64 characters')
        ),
      owner_type: z.enum([DISCOUNT_OWNER.PLATFORM, DISCOUNT_OWNER.AGENT]),
      owner_id: z.number().int().min(0),
      base_discount: z
        .string()
        .refine(
          isDiscountInput,
          t('Discount must be a number greater than 0 and at most 1')
        ),
      min_discount: z
        .string()
        .refine(
          isRatioInput,
          t('Minimum discount must be a number between 0 and 1')
        ),
      billing_mode: z.enum([
        DISCOUNT_BILLING_MODE.USAGE,
        DISCOUNT_BILLING_MODE.SUBSCRIPTION,
        DISCOUNT_BILLING_MODE.FREE,
      ]),
      commission_ratio: z
        .string()
        .refine(
          isRatioInput,
          t('Commission ratio must be a number between 0 and 1')
        ),
      // 发放折算比例（task-09）：与 base_discount 同口径校验——不允许 0 也不允许 > 1。
      // 0 是白送、>1 是倒贴，两者都会赔钱，必须让填的人看见红字自己改。
      topup_conversion_rate: z
        .string()
        .refine(
          isDiscountInput,
          t('Topup conversion rate must be greater than 0 and at most 1')
        ),
      status: z.union([
        z.literal(DISCOUNT_PLAN_STATUS.ENABLED),
        z.literal(DISCOUNT_PLAN_STATUS.DISABLED),
      ]),
      remark: z
        .string()
        .max(
          DISCOUNT_PLAN_REMARK_MAX_LENGTH,
          t('Remark must be at most 255 characters')
        ),
    })
    .superRefine((values, ctx) => {
      if (values.owner_type === DISCOUNT_OWNER.AGENT && values.owner_id <= 0) {
        ctx.addIssue({
          code: 'custom',
          path: ['owner_id'],
          message: t('An agent-owned plan needs the agent user ID'),
        })
      }

      const base = normalizeRatioInput(values.base_discount)
      const min = normalizeRatioInput(values.min_discount, { allowZero: true })
      // 底价高于成交价没有意义，后端也会拦（ValidateDiscountFloor），这里先一步说清楚。
      if (base !== null && min !== null && Number(min) > Number(base)) {
        ctx.addIssue({
          code: 'custom',
          path: ['min_discount'],
          message: t(
            'Minimum discount cannot be higher than the base discount'
          ),
        })
      }
    })
}

/** 表单值 → 提交给后端的载荷；折扣在这里统一凑成 6 位小数。 */
export function buildDiscountPlanPayload(
  values: DiscountPlanFormValues
): DiscountPlanPayload {
  const ownerId =
    values.owner_type === DISCOUNT_OWNER.AGENT ? values.owner_id : 0
  return {
    name: values.name.trim(),
    owner_type: values.owner_type,
    owner_id: ownerId,
    base_discount: normalizeRatioInput(values.base_discount) ?? '1.000000',
    min_discount:
      normalizeRatioInput(values.min_discount, { allowZero: true }) ??
      '0.000000',
    billing_mode: values.billing_mode,
    commission_ratio:
      normalizeRatioInput(values.commission_ratio, { allowZero: true }) ??
      '0.000000',
    // topup_conversion_rate：空值兜底 1.000000（与后端 DiscountTopupConversionDefault 同口径）。
    // 这里走 isDiscountInput 路径而不是 isRatioInput——前者不允许 0，后者允许。
    topup_conversion_rate:
      normalizeRatioInput(values.topup_conversion_rate) ?? '1.000000',
    status: values.status,
    remark: values.remark.trim(),
  }
}

/** 已有方案 → 表单初值；折扣字段去掉多余的 0，方便直接改。 */
export function transformDiscountPlanToFormDefaults(
  plan: DiscountPlan
): DiscountPlanFormValues {
  return {
    name: plan.name,
    owner_type: plan.owner_type || DISCOUNT_OWNER.PLATFORM,
    owner_id: plan.owner_id ?? 0,
    base_discount: ratioTextToInput(plan.base_discount),
    min_discount: ratioTextToInput(plan.min_discount),
    billing_mode: plan.billing_mode || DISCOUNT_BILLING_MODE.USAGE,
    commission_ratio: ratioTextToInput(plan.commission_ratio),
    // 老库升级上来的方案可能没有 topup_conversion_rate 字段：兜底成 1.0，与
    // DiscountTopupConversionDefault 同口径。ratioTextToInput 解析失败时
    // 会返回空串，再交给 buildDiscountPlanPayload 用 normalizeRatioInput 兜回 1.000000。
    topup_conversion_rate:
      plan.topup_conversion_rate && plan.topup_conversion_rate.length > 0
        ? ratioTextToInput(plan.topup_conversion_rate)
        : '1',
    status: plan.status,
    remark: plan.remark ?? '',
  }
}
