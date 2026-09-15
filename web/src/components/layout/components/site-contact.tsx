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
  Building2,
  Clock,
  Link2,
  Mail,
  MapPin,
  Phone,
  type LucideIcon,
} from 'lucide-react'
import type { ComponentType } from 'react'
import {
  SiBilibili,
  SiDiscord,
  SiGithub,
  SiQq,
  SiTelegram,
  SiWechat,
  SiWhatsapp,
  SiX,
  SiYoutube,
} from 'react-icons/si'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import { isHttpUrl } from '@/lib/content-format'
import type { SiteContactInfo, SiteSocialInfo } from '@/features/auth/types'

import { cn } from '@/lib/utils'

interface CustomLink {
  label: string
  url: string
}

function parseCustomLinks(raw?: string): CustomLink[] {
  if (!raw?.trim()) return []
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter(
        (item): item is CustomLink =>
          !!item &&
          typeof item.label === 'string' &&
          typeof item.url === 'string' &&
          item.label.trim() !== '' &&
          item.url.trim() !== ''
      )
      .map((item) => ({ label: item.label, url: item.url }))
  } catch {
    return []
  }
}

/** Fields managed in 业务设置 → 站点 → 站点信息 */
function useSiteContactData() {
  const { status } = useStatus()
  const contact = status?.site_contact ?? {}
  const social = status?.site_social ?? {}
  return { contact, social }
}

/** True when at least one contact/social field is configured */
function hasAnyContactInfo(
  contact: SiteContactInfo,
  social: SiteSocialInfo
): boolean {
  const contactKeys: (keyof SiteContactInfo)[] = [
    'company_name',
    'phone',
    'email',
    'address',
    'service_hours',
  ]
  const socialKeys: (keyof SiteSocialInfo)[] = [
    'wechat_qrcode',
    'qq_group',
    'telegram',
    'discord',
    'github',
    'twitter',
    'youtube',
    'bilibili',
    'whatsapp',
    'custom_links',
  ]
  return (
    contactKeys.some((k) => !!contact[k]?.trim()) ||
    socialKeys.some((k) => !!social[k]?.trim())
  )
}

/** Compact contact lines for the footer brand column */
export function SiteContactDetails() {
  const { contact } = useSiteContactData()
  const rows: { key: string; value: string }[] = [
    { key: 'phone', value: contact.phone ?? '' },
    { key: 'email', value: contact.email ?? '' },
    { key: 'address', value: contact.address ?? '' },
    { key: 'service_hours', value: contact.service_hours ?? '' },
  ].filter((row) => row.value.trim() !== '')

  if (rows.length === 0) return null

  return (
    <div className='text-muted-foreground/80 mt-3 space-y-1 text-xs'>
      {rows.map((row) => (
        <p key={row.key}>{row.value}</p>
      ))}
    </div>
  )
}

const SOCIAL_ICON_DEFS: {
  key: keyof SiteSocialInfo
  label: string
  Icon: ComponentType<{ className?: string }>
}[] = [
  { key: 'github', label: 'GitHub', Icon: SiGithub },
  { key: 'twitter', label: 'X (Twitter)', Icon: SiX },
  { key: 'telegram', label: 'Telegram', Icon: SiTelegram },
  { key: 'discord', label: 'Discord', Icon: SiDiscord },
  { key: 'whatsapp', label: 'WhatsApp', Icon: SiWhatsapp },
  { key: 'youtube', label: 'YouTube', Icon: SiYoutube },
  { key: 'bilibili', label: 'Bilibili', Icon: SiBilibili },
  { key: 'qq_group', label: 'QQ Group', Icon: SiQq },
]

/** Social icon row; values that are not URLs render as plain labeled text */
export function SiteSocialLinks(props: { className?: string }) {
  const { social } = useSiteContactData()
  const customLinks = parseCustomLinks(social.custom_links)

  const items = SOCIAL_ICON_DEFS.filter((def) =>
    !!social[def.key]?.trim()
  ).map((def) => {
    const value = (social[def.key] ?? '').trim()
    return { ...def, value, href: isHttpUrl(value) ? value : undefined }
  })

  if (items.length === 0 && customLinks.length === 0) return null

  return (
    <div className={cn('mt-3 flex flex-wrap items-center gap-3', props.className)}>
      {items.map((item) =>
        item.href ? (
          <a
            key={item.key}
            href={item.href}
            target='_blank'
            rel='noopener noreferrer'
            title={item.label}
            aria-label={item.label}
            className='text-muted-foreground hover:text-foreground hover:bg-muted/60 flex size-8 items-center justify-center rounded-lg transition-colors'
          >
            <item.Icon className='size-4' />
          </a>
        ) : (
          <span
            key={item.key}
            title={item.label}
            className='text-muted-foreground bg-muted/40 flex h-8 items-center gap-1.5 rounded-lg px-2 text-xs'
          >
            <item.Icon className='size-3.5' />
            {item.value}
          </span>
        )
      )}
      {customLinks.map((link) => (
        <a
          key={link.url}
          href={link.url}
          target='_blank'
          rel='noopener noreferrer'
          title={link.label}
          className='text-muted-foreground hover:text-foreground hover:bg-muted/60 flex h-8 items-center gap-1.5 rounded-lg px-2 text-xs transition-colors'
        >
          <Link2 className='size-3.5' />
          {link.label}
        </a>
      ))}
    </div>
  )
}

/** WeChat QR code image, rendered when configured */
export function SiteWechatQr(props: { size?: number; className?: string }) {
  const { t } = useTranslation()
  const { social } = useSiteContactData()
  const url = social.wechat_qrcode?.trim()
  if (!url || !isHttpUrl(url)) return null

  return (
    <div className={cn('flex flex-col items-center gap-1.5', props.className)}>
      <img
        src={url}
        alt={t('WeChat QR Code')}
        style={{ width: props.size ?? 96, height: props.size ?? 96 }}
        className='rounded-lg border object-contain p-1'
      />
      <span className='text-muted-foreground flex items-center gap-1 text-xs'>
        <SiWechat className='size-3.5' />
        {t('WeChat')}
      </span>
    </div>
  )
}

/**
 * Contact card for the about page. Renders nothing (null) when the admin has
 * not configured any contact/social info, so callers can keep their fallback.
 */
export function SiteContactCard() {
  const { t } = useTranslation()
  const { contact, social } = useSiteContactData()

  if (!hasAnyContactInfo(contact, social)) return null

  const rows: {
    key: string
    Icon: LucideIcon
    label: string
    value: string
    href?: string
  }[] = []
  if (contact.company_name?.trim()) {
    rows.push({
      key: 'company',
      Icon: Building2,
      label: t('Company Name'),
      value: contact.company_name,
    })
  }
  if (contact.phone?.trim()) {
    rows.push({
      key: 'phone',
      Icon: Phone,
      label: t('Contact Phone'),
      value: contact.phone,
    })
  }
  if (contact.email?.trim()) {
    rows.push({
      key: 'email',
      Icon: Mail,
      label: t('Contact Email'),
      value: contact.email,
      href: `mailto:${contact.email}`,
    })
  }
  if (contact.address?.trim()) {
    rows.push({
      key: 'address',
      Icon: MapPin,
      label: t('Company Address'),
      value: contact.address,
    })
  }
  if (contact.service_hours?.trim()) {
    rows.push({
      key: 'hours',
      Icon: Clock,
      label: t('Service Hours'),
      value: contact.service_hours,
    })
  }

  const hasQr = !!social.wechat_qrcode?.trim()

  return (
    <div className='border-border/50 bg-card mt-4 rounded-2xl border px-6 py-5'>
      <h3 className='text-base font-semibold tracking-tight'>
        {t('Contact Us')}
      </h3>
      <div className={cn('mt-4 flex flex-wrap gap-8', !hasQr && 'justify-center')}>
        <div className='space-y-3'>
          {rows.map((row) => (
            <div key={row.key} className='flex items-start gap-3 text-sm'>
              <row.Icon className='text-muted-foreground mt-0.5 size-4 shrink-0' />
              <span className='text-muted-foreground w-20 shrink-0'>
                {row.label}
              </span>
              {row.href ? (
                <a
                  href={row.href}
                  className='hover:text-primary transition-colors'
                >
                  {row.value}
                </a>
              ) : (
                <span>{row.value}</span>
              )}
            </div>
          ))}
          <SiteSocialLinks className='mt-4' />
        </div>
        {hasQr && <SiteWechatQr size={112} />}
      </div>
    </div>
  )
}
