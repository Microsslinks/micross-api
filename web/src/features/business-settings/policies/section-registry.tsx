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
import { Gauge, KeyRound, UserPlus } from 'lucide-react'

import { createSectionRegistry } from '@/features/system-settings/utils/section-registry'

import { BasicAuthSection } from './sections/basic-auth-section'
import { RateLimitSection } from './sections/rate-limit-section'
import { TokenLimitSection } from './sections/token-limit-section'

/**
 * 「业务策略」配置组的字段集合。
 *
 * 这些 section 原先分散在「认证」（basic-auth）与「安全」
 * （rate-limit / token-limits）两个技术分组下，与 OAuth、SSRF
 * 等技术开关混在一起，运营人员难以定位。此处只做归属聚合，
 * 字段读写仍由原 section 组件负责。
 */
export type PolicySettings = {
  PasswordLoginEnabled: boolean
  PasswordRegisterEnabled: boolean
  EmailVerificationEnabled: boolean
  RegisterEnabled: boolean
  EmailDomainRestrictionEnabled: boolean
  EmailAliasRestrictionEnabled: boolean
  EmailDomainWhitelist: string
  ModelRequestRateLimitEnabled: boolean
  ModelRequestRateLimitCount: number
  ModelRequestRateLimitSuccessCount: number
  ModelRequestRateLimitDurationMinutes: number
  ModelRequestRateLimitGroup: string
  'token_setting.max_user_tokens': number
}

const POLICIES_SECTIONS = [
  {
    id: 'registration',
    titleKey: 'Registration & Access',
    icon: UserPlus,
    build: (settings: PolicySettings) => (
      <BasicAuthSection
        defaultValues={{
          PasswordLoginEnabled: settings.PasswordLoginEnabled,
          PasswordRegisterEnabled: settings.PasswordRegisterEnabled,
          EmailVerificationEnabled: settings.EmailVerificationEnabled,
          RegisterEnabled: settings.RegisterEnabled,
          EmailDomainRestrictionEnabled: settings.EmailDomainRestrictionEnabled,
          EmailAliasRestrictionEnabled: settings.EmailAliasRestrictionEnabled,
          EmailDomainWhitelist: settings.EmailDomainWhitelist,
        }}
      />
    ),
  },
  {
    id: 'rate-limit',
    titleKey: 'Rate Limiting',
    icon: Gauge,
    build: (settings: PolicySettings) => (
      <RateLimitSection
        defaultValues={{
          ModelRequestRateLimitEnabled: settings.ModelRequestRateLimitEnabled,
          ModelRequestRateLimitCount: settings.ModelRequestRateLimitCount,
          ModelRequestRateLimitSuccessCount:
            settings.ModelRequestRateLimitSuccessCount,
          ModelRequestRateLimitDurationMinutes:
            settings.ModelRequestRateLimitDurationMinutes,
          ModelRequestRateLimitGroup: settings.ModelRequestRateLimitGroup,
        }}
      />
    ),
  },
  {
    id: 'token-limits',
    titleKey: 'Token Limits',
    icon: KeyRound,
    build: (settings: PolicySettings) => (
      <TokenLimitSection
        defaultValues={{
          'token_setting.max_user_tokens':
            settings['token_setting.max_user_tokens'],
        }}
      />
    ),
  },
] as const

export type PolicySectionId = (typeof POLICIES_SECTIONS)[number]['id']

const policiesRegistry = createSectionRegistry<PolicySectionId, PolicySettings>(
  {
    sections: POLICIES_SECTIONS,
    defaultSection: 'registration',
    basePath: '/business-settings/policies',
    urlStyle: 'path',
  }
)

export const POLICIES_SECTION_IDS = policiesRegistry.sectionIds
export const POLICIES_DEFAULT_SECTION = policiesRegistry.defaultSection
export const getPoliciesSectionNavItems = policiesRegistry.getSectionNavItems
export const getPoliciesSectionContent = policiesRegistry.getSectionContent
export const getPoliciesSectionMeta = policiesRegistry.getSectionMeta
