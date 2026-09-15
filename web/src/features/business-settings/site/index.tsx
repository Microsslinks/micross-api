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
import type { SiteSettings } from '@/features/system-settings/types'

import {
  SITE_DEFAULT_SECTION,
  getSiteSectionContent,
  getSiteSectionMeta,
} from './section-registry.tsx'

const defaultSiteSettings: SiteSettings = {
  Notice: '',
  SystemName: 'Microsslink',
  Logo: '',
  Icp: '',
  Footer: '',
  About: '',
  HomePageContent: '',
  ServerAddress: '',
  'legal.user_agreement': '',
  'legal.privacy_policy': '',
  HeaderNavModules: '',
  SidebarModulesAdmin: '',
  'general_setting.docs_link': '',
  'site_contact.company_name': '',
  'site_contact.phone': '',
  'site_contact.email': '',
  'site_contact.address': '',
  'site_contact.service_hours': '',
  'site_social.wechat_qrcode': '',
  'site_social.qq_group': '',
  'site_social.telegram': '',
  'site_social.discord': '',
  'site_social.github': '',
  'site_social.twitter': '',
  'site_social.youtube': '',
  'site_social.bilibili': '',
  'site_social.whatsapp': '',
  'site_social.custom_links': '',
  'site_content.home_page_content_enabled': true,
  'site_content.about_enabled': true,
  'site_content.footer_enabled': true,
  'site_content.user_agreement_enabled': true,
  'site_content.privacy_policy_enabled': true,
}

export function SiteSettings() {
  return (
    <SettingsPage
      routePath='/_authenticated/business-settings/site/$section'
      defaultSettings={defaultSiteSettings}
      defaultSection={SITE_DEFAULT_SECTION}
      getSectionContent={getSiteSectionContent}
      getSectionMeta={getSiteSectionMeta}
    />
  )
}
