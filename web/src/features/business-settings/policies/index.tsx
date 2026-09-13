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
import { SettingsPage } from '@/features/system-settings/components/settings-page'

import {
  POLICIES_DEFAULT_SECTION,
  getPoliciesSectionContent,
  getPoliciesSectionMeta,
  type PolicySettings,
} from './section-registry.tsx'

const defaultPolicySettings: PolicySettings = {
  PasswordLoginEnabled: true,
  PasswordRegisterEnabled: true,
  EmailVerificationEnabled: false,
  RegisterEnabled: true,
  EmailDomainRestrictionEnabled: false,
  EmailAliasRestrictionEnabled: false,
  EmailDomainWhitelist: '',
  ModelRequestRateLimitEnabled: false,
  ModelRequestRateLimitCount: 0,
  ModelRequestRateLimitSuccessCount: 1000,
  ModelRequestRateLimitDurationMinutes: 1,
  ModelRequestRateLimitGroup: '',
  'token_setting.max_user_tokens': 1000,
}

export function PoliciesSettings() {
  return (
    <SettingsPage
      routePath='/_authenticated/business-settings/policies/$section'
      defaultSettings={defaultPolicySettings}
      defaultSection={POLICIES_DEFAULT_SECTION}
      getSectionContent={getPoliciesSectionContent}
      getSectionMeta={getPoliciesSectionMeta}
      loadingMessage='Loading business policies...'
    />
  )
}
