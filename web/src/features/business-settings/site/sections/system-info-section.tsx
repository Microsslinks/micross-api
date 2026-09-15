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
import { zodResolver } from '@hookform/resolvers/zod'
import type { Control, Resolver } from 'react-hook-form'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { ContentEditorDialog } from '@/features/business-settings/site/components/content-editor-dialog'
import { FormDirtyIndicator } from '@/features/system-settings/components/form-dirty-indicator'
import { FormNavigationGuard } from '@/features/system-settings/components/form-navigation-guard'
import {
  SettingsForm,
  SettingsFormGrid,
  SettingsFormGridItem,
} from '@/features/system-settings/components/settings-form-layout'
import { SettingsPageFormActions } from '@/features/system-settings/components/settings-page-context'
import { SettingsSection } from '@/features/system-settings/components/settings-section'
import { useSettingsForm } from '@/features/system-settings/hooks/use-settings-form'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import { isHttpUrl } from '@/lib/content-format'

const _systemInfoSchema = z.object({
  SystemName: z.string().min(1),
  ServerAddress: z.string().optional(),
  Logo: z.string().url().optional().or(z.literal('')),
  Icp: z.string().optional(),
  general_setting: z.object({
    docs_link: z.string().optional(),
  }),
  site_contact: z.object({
    company_name: z.string().optional(),
    phone: z.string().optional(),
    email: z.string().optional(),
    address: z.string().optional(),
    service_hours: z.string().optional(),
  }),
  site_social: z.object({
    wechat_qrcode: z.string().optional(),
    qq_group: z.string().optional(),
    telegram: z.string().optional(),
    discord: z.string().optional(),
    github: z.string().optional(),
    twitter: z.string().optional(),
    youtube: z.string().optional(),
    bilibili: z.string().optional(),
    whatsapp: z.string().optional(),
    custom_links: z.string().optional(),
  }),
  site_content: z.object({
    home_page_content_enabled: z.boolean(),
    about_enabled: z.boolean(),
    footer_enabled: z.boolean(),
    user_agreement_enabled: z.boolean(),
    privacy_policy_enabled: z.boolean(),
  }),
})

type SystemInfoFormValues = z.infer<typeof _systemInfoSchema>

/** Large-text option values edited via dialog, not the form */
export type SiteContentValues = {
  HomePageContent: string
  About: string
  Footer: string
  'legal.user_agreement': string
  'legal.privacy_policy': string
}

type SystemInfoSectionProps = {
  defaultValues: SystemInfoFormValues
  contentValues: SiteContentValues
}

function normalizeValue(value: unknown): string {
  if (value === undefined || value === null) return ''
  return typeof value === 'string' ? value : String(value)
}

function normalizeBool(value: unknown, fallback = true): boolean {
  if (value === undefined || value === null) return fallback
  return value === true || value === 'true' || value === '1'
}

type ContentItem = {
  /** Option key of the large-text value */
  key: keyof SiteContentValues
  titleKey: string
  /** Form field path of the visibility switch (undefined = no switch) */
  enabledField?:
    | 'site_content.home_page_content_enabled'
    | 'site_content.about_enabled'
    | 'site_content.footer_enabled'
    | 'site_content.user_agreement_enabled'
    | 'site_content.privacy_policy_enabled'
  /** What the site falls back to when the content is empty */
  emptyNoteKey: string
  placeholderKey?: string
}

const CONTENT_ITEMS: ContentItem[] = [
  {
    key: 'HomePageContent',
    titleKey: 'Home Page Content',
    enabledField: 'site_content.home_page_content_enabled',
    emptyNoteKey: 'Using the default home page',
    placeholderKey: 'Welcome to our site...',
  },
  {
    key: 'About',
    titleKey: 'About',
    enabledField: 'site_content.about_enabled',
    emptyNoteKey: 'Using the default about page',
    placeholderKey:
      'Enter HTML code or a URL (e.g., https://example.com) to embed as iframe',
  },
  {
    key: 'Footer',
    titleKey: 'Footer',
    enabledField: 'site_content.footer_enabled',
    emptyNoteKey: 'Using the default footer',
    placeholderKey: '© 2025 Your Company. All rights reserved.',
  },
  {
    key: 'legal.user_agreement',
    titleKey: 'User Agreement',
    enabledField: 'site_content.user_agreement_enabled',
    emptyNoteKey: 'Not configured yet',
    placeholderKey: 'Provide Markdown, HTML, or an external URL',
  },
  {
    key: 'legal.privacy_policy',
    titleKey: 'Privacy Policy',
    enabledField: 'site_content.privacy_policy_enabled',
    emptyNoteKey: 'Not configured yet',
    placeholderKey: 'Provide Markdown, HTML, or an external URL',
  },
]

function ContentStatusText(props: { value: string; emptyNoteKey: string }) {
  const { t } = useTranslation()
  const raw = props.value.trim()
  if (!raw) {
    return (
      <span>
        {t('Not set')} · {t(props.emptyNoteKey)}
      </span>
    )
  }
  if (isHttpUrl(raw)) {
    return <span>{t('External URL')}</span>
  }
  return (
    <span>{t('Set · {{count}} characters', { count: raw.length })}</span>
  )
}

function ContentRow(props: {
  control: Control<SystemInfoFormValues>
  item: ContentItem
  value: string
  onEdit: () => void
}) {
  const { t } = useTranslation()
  const { item } = props
  return (
    <div className='border-border/50 flex items-center justify-between gap-4 border-b py-3.5 last:border-b-0 last:pb-1 first:pt-1'>
      <div className='min-w-0'>
        <div className='text-sm font-medium'>{t(item.titleKey)}</div>
        <div className='text-muted-foreground mt-0.5 text-xs'>
          <ContentStatusText
            value={props.value}
            emptyNoteKey={item.emptyNoteKey}
          />
        </div>
      </div>
      <div className='flex shrink-0 items-center gap-3'>
        {item.enabledField && (
          <FormField
            control={props.control}
            name={item.enabledField}
            render={({ field }) => (
              <FormItem className='space-y-0'>
                <FormControl>
                  <Switch
                    size='sm'
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    aria-label={t(item.titleKey)}
                  />
                </FormControl>
              </FormItem>
            )}
          />
        )}
        <Button variant='outline' size='sm' onClick={props.onEdit}>
          {t('Edit')}
        </Button>
      </div>
    </div>
  )
}

export function SystemInfoSection({
  defaultValues,
  contentValues,
}: SystemInfoSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const [editingKey, setEditingKey] = useState<string | null>(null)
  const editingItem = CONTENT_ITEMS.find((item) => item.key === editingKey)

  const normalizedDefaults: SystemInfoFormValues = {
    SystemName: normalizeValue(defaultValues.SystemName),
    ServerAddress: normalizeValue(defaultValues.ServerAddress),
    Logo: normalizeValue(defaultValues.Logo),
    Icp: normalizeValue(defaultValues.Icp),
    general_setting: {
      docs_link: normalizeValue(defaultValues.general_setting?.docs_link),
    },
    site_contact: {
      company_name: normalizeValue(defaultValues.site_contact?.company_name),
      phone: normalizeValue(defaultValues.site_contact?.phone),
      email: normalizeValue(defaultValues.site_contact?.email),
      address: normalizeValue(defaultValues.site_contact?.address),
      service_hours: normalizeValue(defaultValues.site_contact?.service_hours),
    },
    site_social: {
      wechat_qrcode: normalizeValue(defaultValues.site_social?.wechat_qrcode),
      qq_group: normalizeValue(defaultValues.site_social?.qq_group),
      telegram: normalizeValue(defaultValues.site_social?.telegram),
      discord: normalizeValue(defaultValues.site_social?.discord),
      github: normalizeValue(defaultValues.site_social?.github),
      twitter: normalizeValue(defaultValues.site_social?.twitter),
      youtube: normalizeValue(defaultValues.site_social?.youtube),
      bilibili: normalizeValue(defaultValues.site_social?.bilibili),
      whatsapp: normalizeValue(defaultValues.site_social?.whatsapp),
      custom_links: normalizeValue(defaultValues.site_social?.custom_links),
    },
    site_content: {
      home_page_content_enabled: normalizeBool(
        defaultValues.site_content?.home_page_content_enabled
      ),
      about_enabled: normalizeBool(defaultValues.site_content?.about_enabled),
      footer_enabled: normalizeBool(defaultValues.site_content?.footer_enabled),
      user_agreement_enabled: normalizeBool(
        defaultValues.site_content?.user_agreement_enabled
      ),
      privacy_policy_enabled: normalizeBool(
        defaultValues.site_content?.privacy_policy_enabled
      ),
    },
  }

  const systemInfoSchemaWithI18n = z.object({
    SystemName: z.string().min(1, {
      error: () => t('System name is required'),
    }),
    ServerAddress: z.string().optional(),
    Logo: z.string().url().optional().or(z.literal('')),
    Icp: z.string().optional(),
    general_setting: z.object({
      docs_link: z.string().optional(),
    }),
    site_contact: z.object({
      company_name: z.string().optional(),
      phone: z.string().optional(),
      email: z.string().optional(),
      address: z.string().optional(),
      service_hours: z.string().optional(),
    }),
    site_social: z.object({
      wechat_qrcode: z.string().optional(),
      qq_group: z.string().optional(),
      telegram: z.string().optional(),
      discord: z.string().optional(),
      github: z.string().optional(),
      twitter: z.string().optional(),
      youtube: z.string().optional(),
      bilibili: z.string().optional(),
      whatsapp: z.string().optional(),
      custom_links: z.string().optional(),
    }),
    site_content: z.object({
      home_page_content_enabled: z.boolean(),
      about_enabled: z.boolean(),
      footer_enabled: z.boolean(),
      user_agreement_enabled: z.boolean(),
      privacy_policy_enabled: z.boolean(),
    }),
  })

  const { form, handleSubmit, handleReset, isDirty, isSubmitting } =
    useSettingsForm<SystemInfoFormValues>({
      resolver: zodResolver(systemInfoSchemaWithI18n) as Resolver<
        SystemInfoFormValues,
        unknown,
        SystemInfoFormValues
      >,
      defaultValues: normalizedDefaults,
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          let v = normalizeValue(value)
          if (key === 'ServerAddress') {
            v = v.replace(/\/+$/, '')
          }
          await updateOption.mutateAsync({
            key,
            value: v,
          })
        }
      },
    })

  const logoUrl = form.watch('Logo')?.trim() ?? ''

  const textField = (
    name:
      | 'general_setting.docs_link'
      | 'site_contact.company_name'
      | 'site_contact.phone'
      | 'site_contact.email'
      | 'site_contact.address'
      | 'site_contact.service_hours'
      | 'site_social.wechat_qrcode'
      | 'site_social.qq_group'
      | 'site_social.telegram'
      | 'site_social.discord'
      | 'site_social.github'
      | 'site_social.twitter'
      | 'site_social.youtube'
      | 'site_social.bilibili'
      | 'site_social.whatsapp'
      | 'site_social.custom_links',
    label: string,
    placeholder?: string,
    description?: string,
    spanFull = false
  ) => (
    <SettingsFormGridItem key={name} span={spanFull ? 'full' : undefined}>
      <FormField
        control={form.control}
        name={name}
        render={({ field }) => (
          <FormItem>
            <FormLabel>{label}</FormLabel>
            <FormControl>
              <Input placeholder={placeholder} {...field} />
            </FormControl>
            {description ? (
              <FormDescription>{description}</FormDescription>
            ) : null}
            <FormMessage />
          </FormItem>
        )}
      />
    </SettingsFormGridItem>
  )

  return (
    <>
      <FormNavigationGuard when={isDirty} />

      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            onReset={handleReset}
            isSaving={isSubmitting || updateOption.isPending}
            isResetDisabled={!isDirty}
          />
          <FormDirtyIndicator isDirty={isDirty} />

          <SettingsSection title={t('Brand')}>
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='SystemName'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('System Name')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('New API')} {...field} />
                    </FormControl>
                    <FormDescription>
                      {t('The name displayed across the application')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='Logo'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Logo URL')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('https://example.com/logo.png')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('URL to your logo image (optional)')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='ServerAddress'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Server Address')}</FormLabel>
                    <FormControl>
                      <Input placeholder='https://yourdomain.com' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'The public URL of your server, used for OAuth callbacks, webhooks, and other external integrations'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              {textField(
                'general_setting.docs_link',
                t('Documentation Link'),
                t('https://docs.example.com'),
                t(
                  'Link used by the header Docs entry and the home page documentation button'
                )
              )}

              {logoUrl ? (
                <SettingsFormGridItem span='full'>
                  <div className='flex items-center gap-3'>
                    <img
                      src={logoUrl}
                      alt={t('Logo URL')}
                      className='border-border/50 size-12 rounded-lg border object-contain p-1'
                    />
                    <span className='text-muted-foreground text-xs'>
                      {t('Logo preview')}
                    </span>
                  </div>
                </SettingsFormGridItem>
              ) : null}
            </SettingsFormGrid>
          </SettingsSection>

          <SettingsSection title={t('Contact Information')}>
            <SettingsFormGrid>
              {textField(
                'site_contact.company_name',
                t('Company Name'),
                t('Your company name')
              )}
              {textField(
                'site_contact.phone',
                t('Contact Phone'),
                '+86 400-000-0000'
              )}
              {textField(
                'site_contact.email',
                t('Contact Email'),
                'support@example.com'
              )}
              {textField(
                'site_contact.service_hours',
                t('Service Hours'),
                t('e.g., Weekdays 9:00 - 18:00')
              )}
              {textField(
                'site_contact.address',
                t('Company Address'),
                t('Your office address'),
                undefined,
                true
              )}
            </SettingsFormGrid>
          </SettingsSection>

          <SettingsSection title={t('Social Media')}>
            <SettingsFormGrid>
              {textField(
                'site_social.wechat_qrcode',
                t('WeChat QR Code'),
                'https://example.com/wechat-qr.png',
                t('Image URL of your WeChat group or support QR code')
              )}
              {textField('site_social.qq_group', t('QQ Group'), '123456789')}
              {textField(
                'site_social.telegram',
                'Telegram',
                'https://t.me/yourgroup'
              )}
              {textField(
                'site_social.discord',
                'Discord',
                'https://discord.gg/xxxxx'
              )}
              {textField(
                'site_social.github',
                'GitHub',
                'https://github.com/yourorg'
              )}
              {textField(
                'site_social.twitter',
                t('X (Twitter)'),
                'https://x.com/youraccount'
              )}
              {textField(
                'site_social.whatsapp',
                'WhatsApp',
                'https://wa.me/8613800000000'
              )}
              {textField(
                'site_social.youtube',
                'YouTube',
                'https://youtube.com/@yourchannel'
              )}
              {textField(
                'site_social.bilibili',
                t('Bilibili'),
                'https://space.bilibili.com/123456'
              )}
              {textField(
                'site_social.custom_links',
                t('Custom Links'),
                '[{"label":"...","url":"https://..."}]',
                t(
                  'JSON array of extra links, e.g. [{"label":"Forum","url":"https://..."}]'
                ),
                true
              )}
            </SettingsFormGrid>
          </SettingsSection>

          <SettingsSection title={t('Content Pages')}>
            <div className='text-muted-foreground mb-2 text-xs'>
              {t(
                'Large content is edited in a dialog. Turn off a switch to hide that page on the site.'
              )}
            </div>
            {CONTENT_ITEMS.map((item) => (
              <ContentRow
                key={item.key}
                control={form.control}
                item={item}
                value={contentValues[item.key] ?? ''}
                onEdit={() => setEditingKey(item.key)}
              />
            ))}
          </SettingsSection>

          <SettingsSection title={t('Filing Information')}>
            <SettingsFormGrid>
              <FormField
                control={form.control}
                name='Icp'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('ICP Filing Number')}</FormLabel>
                    <FormControl>
                      <Input placeholder='京ICP备00000000号' {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Shown in the footer as a link to beian.miit.gov.cn. Leave empty to hide it.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </SettingsFormGrid>
          </SettingsSection>
        </SettingsForm>
      </Form>

      {editingItem && (
        <ContentEditorDialog
          open
          onOpenChange={(open) => {
            if (!open) setEditingKey(null)
          }}
          title={t(editingItem.titleKey)}
          optionKey={editingItem.key}
          initialValue={contentValues[editingItem.key] ?? ''}
          placeholder={
            editingItem.placeholderKey ? t(editingItem.placeholderKey) : undefined
          }
        />
      )}
    </>
  )
}
