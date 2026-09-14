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
export {
  RATIO_UNKNOWN_TEXT,
  parseRatioText,
  formatRatioText,
  formatMarginText,
} from './format'
export { normalizeRatioInput, ratioTextToInput } from './ratio'
export { mergeModelNames, splitModelList } from './model-list'
export {
  DISCOUNT_PLAN_FORM_DEFAULT_VALUES,
  DISCOUNT_PLAN_NAME_MAX_LENGTH,
  DISCOUNT_PLAN_REMARK_MAX_LENGTH,
  buildDiscountPlanPayload,
  getDiscountPlanFormSchema,
  transformDiscountPlanToFormDefaults,
  type DiscountPlanFormValues,
} from './plan-form'
export {
  DISCOUNT_RULE_FORM_DEFAULT_VALUES,
  DISCOUNT_RULE_SCOPE_VALUE_MAX_LENGTH,
  buildDiscountRulePayload,
  getDiscountRuleFormSchema,
  transformDiscountRuleToFormDefaults,
  type DiscountRuleFormValues,
} from './rule-form'
