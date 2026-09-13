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
import { useMemo } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

/**
 * 折扣策略：两个键。
 *
 * - `discount_setting.min_margin_ratio`（毛利底线，默认 0）：试算与保存前校验用它判断
 *   一条线路会不会亏——进货折扣高于「客户折扣 − 毛利底线」时标红。填 0 表示不赔本就行。
 * - `discount_setting.enable_billing_discount`（默认关）：折扣是否参与实际扣费。
 *   关着时折扣只用于配置与试算，每笔仍按官方标价收费；打开后也只对「绑定了启用中方案」
 *   的客户生效，没绑方案的人照旧。
 *
 * 表单内部刻意不使用带点的服务端键名——react-hook-form 7 会把带点的 name
 * 解释成嵌套路径，导致表单状态与校验、落库的键名不一致。这里用本地字段名
 * 建模，保存前才摊平成 `discount_setting.*`
 * （同样的取舍见 maintenance/performance-section.tsx 的注释）。
 */
const createDiscountSettingSchema = (t: (key: string) => string) =>
  z.object({
    min_margin_ratio: z.string().refine((value) => {
      const trimmed = value.trim()
      if (!trimmed) return true
      if (!/^\d+(\.\d{1,6})?$/.test(trimmed)) return false
      // 输入框的 max 只在原生校验里起作用，而保存按钮走的是 onClick + handleSubmit，
      // 所以上界必须在这里再判一遍。
      return Number(trimmed) <= 1
    }, t('Enter a number between 0 and 1 with up to 6 decimal places, or leave empty')),
    enable_billing_discount: z.boolean(),
  })

type DiscountSettingFormValues = z.infer<
  ReturnType<typeof createDiscountSettingSchema>
>

type FlatDiscountDefaults = {
  'discount_setting.min_margin_ratio': string
  'discount_setting.enable_billing_discount': boolean
}

const buildFormDefaults = (defaults: FlatDiscountDefaults) => ({
  min_margin_ratio: defaults['discount_setting.min_margin_ratio'] ?? '',
  enable_billing_discount:
    defaults['discount_setting.enable_billing_discount'] ?? false,
})

type DiscountSettingSectionProps = {
  defaultValues: FlatDiscountDefaults
}

export function DiscountSettingSection(props: DiscountSettingSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const formDefaults = useMemo(
    () => buildFormDefaults(props.defaultValues),
    [props.defaultValues]
  )

  const form = useForm<DiscountSettingFormValues>({
    resolver: zodResolver(createDiscountSettingSchema(t)),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const onSubmit = async (values: DiscountSettingFormValues) => {
    // 留空与 0 同义：后端把空串也当 0 处理，这里统一成 "0" 少一种落库形态。
    const normalized: FlatDiscountDefaults = {
      'discount_setting.min_margin_ratio':
        values.min_margin_ratio.trim() || '0',
      'discount_setting.enable_billing_discount':
        values.enable_billing_discount,
    }
    const changedKeys = (
      Object.keys(normalized) as Array<keyof FlatDiscountDefaults>
    ).filter((key) => normalized[key] !== props.defaultValues[key])

    if (changedKeys.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of changedKeys) {
      await updateOption.mutateAsync({ key, value: normalized[key] })
    }
  }

  return (
    <SettingsSection title={t('Discount Policy')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='enable_billing_discount'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Apply discounts to billing')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When off, discount plans only drive simulation and pre-save checks and every request is billed at the official price. When on, customers bound to an enabled plan are billed at the discounted price.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value ?? false}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <div>
            <h4 className='font-medium'>{t('Profit Floor')}</h4>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t(
                'Used by discount simulation and pre-save validation. A route is flagged as a loss when its cost ratio is above the customer discount minus this margin.'
              )}
            </p>
          </div>

          <FormField
            control={form.control}
            name='min_margin_ratio'
            render={({ field }) => (
              <FormItem className='max-w-xs'>
                <FormLabel>{t('Minimum Margin Ratio')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={0}
                    max={1}
                    step={0.000001}
                    placeholder='0'
                    value={field.value ?? ''}
                    onChange={(event) => field.onChange(event.target.value)}
                    name={field.name}
                    onBlur={field.onBlur}
                    ref={field.ref}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Example: 0.05 requires the cost ratio to stay at least 5 points below the customer discount. Leave empty or 0 to accept break-even.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
