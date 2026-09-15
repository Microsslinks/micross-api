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
import {
  CircleHelp,
  Image,
  LayoutDashboard,
  Link,
  Megaphone,
} from 'lucide-react'

import type { ContentSettings } from '@/features/system-settings/types'
import { createSectionRegistry } from '@/features/system-settings/utils/section-registry'

import { AnnouncementsSection } from './announcements-section'
import { ApiInfoSection } from './api-info-section'
import { DashboardSection } from './dashboard-section'
import { DrawingSettingsSection } from './drawing-settings-section'
import { FAQSection } from './faq-section'

/**
 * Validate and coerce DataExportDefaultTime to a safe value
 */
function validateDataExportDefaultTime(value: string): 'week' | 'hour' | 'day' {
  if (value === 'week' || value === 'hour' || value === 'day') {
    return value
  }
  // Default to 'hour' if value is unexpected
  return 'hour'
}

const CONTENT_SECTIONS = [
  {
    id: 'dashboard',
    titleKey: 'Data Dashboard',
    icon: LayoutDashboard,
    build: (settings: ContentSettings) => (
      <DashboardSection
        defaultValues={{
          DataExportEnabled: settings.DataExportEnabled,
          DataExportInterval: settings.DataExportInterval,
          DataExportDefaultTime: validateDataExportDefaultTime(
            settings.DataExportDefaultTime
          ),
        }}
      />
    ),
  },
  {
    id: 'announcements',
    titleKey: 'Business Announcements',
    icon: Megaphone,
    build: (settings: ContentSettings) => (
      <AnnouncementsSection
        enabled={settings['console_setting.announcements_enabled']}
        data={settings['console_setting.announcements']}
      />
    ),
  },
  {
    id: 'api-info',
    titleKey: 'API Address Settings',
    icon: Link,
    build: (settings: ContentSettings) => (
      <ApiInfoSection
        enabled={settings['console_setting.api_info_enabled']}
        data={settings['console_setting.api_info']}
      />
    ),
  },
  {
    id: 'faq',
    titleKey: 'FAQ',
    icon: CircleHelp,
    build: (settings: ContentSettings) => (
      <FAQSection
        enabled={settings['console_setting.faq_enabled']}
        data={settings['console_setting.faq']}
      />
    ),
  },
  {
    id: 'drawing',
    titleKey: 'Image Feature Settings',
    icon: Image,
    build: (settings: ContentSettings) => (
      <DrawingSettingsSection
        defaultValues={{
          DrawingEnabled: settings.DrawingEnabled,
          MjNotifyEnabled: settings.MjNotifyEnabled,
          MjAccountFilterEnabled: settings.MjAccountFilterEnabled,
          MjForwardUrlEnabled: settings.MjForwardUrlEnabled,
          MjModeClearEnabled: settings.MjModeClearEnabled,
          MjActionCheckSuccessEnabled: settings.MjActionCheckSuccessEnabled,
        }}
      />
    ),
  },
] as const

export type ContentSectionId = (typeof CONTENT_SECTIONS)[number]['id']

const contentRegistry = createSectionRegistry<
  ContentSectionId,
  ContentSettings
>({
  sections: CONTENT_SECTIONS,
  defaultSection: 'dashboard',
  basePath: '/business-settings/content',
  urlStyle: 'path',
})

export const CONTENT_SECTION_IDS = contentRegistry.sectionIds
export const CONTENT_DEFAULT_SECTION = contentRegistry.defaultSection
export const getContentSectionNavItems = contentRegistry.getSectionNavItems
export const getContentSectionContent = contentRegistry.getSectionContent
export const getContentSectionMeta = contentRegistry.getSectionMeta
