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
import { Bell, BookOpen, Info, LayoutTemplate, Menu } from 'lucide-react'

import type { SiteSettings } from '@/features/system-settings/types'
import { createSectionRegistry } from '@/features/system-settings/utils/section-registry'

import {
  parseHeaderNavModules,
  parseSidebarModulesAdmin,
  serializeHeaderNavModules,
  serializeSidebarModulesAdmin,
} from './sections/config'
import { DocsLinkSection } from './sections/docs-link-section'
import { HeaderNavigationSection } from './sections/header-navigation-section'
import { NoticeSection } from './sections/notice-section'
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
          Footer: settings.Footer,
          About: settings.About,
          HomePageContent: settings.HomePageContent,
          ServerAddress: settings.ServerAddress,
          legal: {
            user_agreement: settings['legal.user_agreement'],
            privacy_policy: settings['legal.privacy_policy'],
          },
        }}
      />
    ),
  },
  {
    id: 'notice',
    titleKey: 'System Notice',
    icon: Bell,
    build: (settings: SiteSettings) => (
      <NoticeSection defaultValue={settings.Notice ?? ''} />
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
    id: 'docs-link',
    titleKey: 'Documentation Link',
    icon: BookOpen,
    build: (settings: SiteSettings) => (
      <DocsLinkSection
        defaultValue={settings['general_setting.docs_link'] ?? ''}
      />
    ),
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
