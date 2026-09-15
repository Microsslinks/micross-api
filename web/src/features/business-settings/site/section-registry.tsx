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
import { Info, LayoutTemplate, Menu } from 'lucide-react'

import type { SiteSettings } from '@/features/system-settings/types'
import { createSectionRegistry } from '@/features/system-settings/utils/section-registry'

import {
  parseHeaderNavModules,
  parseSidebarModulesAdmin,
  serializeHeaderNavModules,
  serializeSidebarModulesAdmin,
} from './sections/config'
import { HeaderNavigationSection } from './sections/header-navigation-section'
import { SidebarModulesSection } from './sections/sidebar-modules-section'
import { SystemInfoSection } from './sections/system-info-section'

const SITE_SECTIONS = [
  {
    id: 'system-info',
    titleKey: 'Site Information',
    icon: Info,
    build: (settings: SiteSettings) => (
      <SystemInfoSection
        defaultValues={{
          SystemName: settings.SystemName,
          Logo: settings.Logo,
          Icp: settings.Icp,
          ServerAddress: settings.ServerAddress,
          general_setting: {
            docs_link: settings['general_setting.docs_link'],
          },
          site_contact: {
            company_name: settings['site_contact.company_name'],
            phone: settings['site_contact.phone'],
            email: settings['site_contact.email'],
            address: settings['site_contact.address'],
            service_hours: settings['site_contact.service_hours'],
          },
          site_social: {
            wechat_qrcode: settings['site_social.wechat_qrcode'],
            qq_group: settings['site_social.qq_group'],
            telegram: settings['site_social.telegram'],
            discord: settings['site_social.discord'],
            github: settings['site_social.github'],
            twitter: settings['site_social.twitter'],
            youtube: settings['site_social.youtube'],
            bilibili: settings['site_social.bilibili'],
            whatsapp: settings['site_social.whatsapp'],
            custom_links: settings['site_social.custom_links'],
          },
          site_content: {
            home_page_content_enabled:
              settings['site_content.home_page_content_enabled'],
            about_enabled: settings['site_content.about_enabled'],
            footer_enabled: settings['site_content.footer_enabled'],
            user_agreement_enabled:
              settings['site_content.user_agreement_enabled'],
            privacy_policy_enabled:
              settings['site_content.privacy_policy_enabled'],
          },
        }}
        contentValues={{
          HomePageContent: settings.HomePageContent,
          About: settings.About,
          Footer: settings.Footer,
          'legal.user_agreement': settings['legal.user_agreement'],
          'legal.privacy_policy': settings['legal.privacy_policy'],
        }}
      />
    ),
  },
  {
    id: 'header-navigation',
    titleKey: 'Header navigation',
    icon: Menu,
    build: (settings: SiteSettings) => {
      const headerNavConfig = parseHeaderNavModules(settings.HeaderNavModules)
      const headerNavSerialized = serializeHeaderNavModules(headerNavConfig)
      return (
        <HeaderNavigationSection
          config={headerNavConfig}
          initialSerialized={headerNavSerialized}
        />
      )
    },
  },
  {
    id: 'sidebar-modules',
    titleKey: 'Sidebar modules',
    icon: LayoutTemplate,
    build: (settings: SiteSettings) => {
      const sidebarConfig = parseSidebarModulesAdmin(
        settings.SidebarModulesAdmin
      )
      const sidebarSerialized = serializeSidebarModulesAdmin(sidebarConfig)
      return (
        <SidebarModulesSection
          config={sidebarConfig}
          initialSerialized={sidebarSerialized}
        />
      )
    },
  },
] as const

export type SiteSectionId = (typeof SITE_SECTIONS)[number]['id']

const siteRegistry = createSectionRegistry<SiteSectionId, SiteSettings>({
  sections: SITE_SECTIONS,
  defaultSection: 'system-info',
  basePath: '/business-settings/site',
  urlStyle: 'path',
})

export const SITE_SECTION_IDS = siteRegistry.sectionIds
export const SITE_DEFAULT_SECTION = siteRegistry.defaultSection
export const getSiteSectionNavItems = siteRegistry.getSectionNavItems
export const getSiteSectionContent = siteRegistry.getSectionContent
export const getSiteSectionMeta = siteRegistry.getSectionMeta
