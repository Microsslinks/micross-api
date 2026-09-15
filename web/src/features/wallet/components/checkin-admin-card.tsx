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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarCheck } from 'lucide-react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { TitledCard } from '@/components/ui/titled-card'
import { getCheckinSetting, updateCheckinSetting } from '../api'

const schema = z.object({
  enabled: z.boolean(),
  minQuota: z.coerce.number().int().min(0),
  maxQuota: z.coerce.number().int().min(0),
})

type Values = z.infer<typeof schema>

/**
 * 「签到奖励」管理员配置卡
 *
 * 原先是业务管理 → 计费与定价里的一个分节，但只有开关和两个额度，
 * 不值得单独占一个导航入口——签到发的是钱包里的钱（每日额度），
 * 所以就近挂在钱包页最下面，与「推荐收益」同一骨架（TitledCard）。
 *
 * 读写走 /api/checkin/setting（管理员权限）：业务管理定位是管理员
 * 即可操作，但签到配置存在选项库里，通用 /api/option/ 是超管权限，
 * 所以后端单开了这对只放行 checkin_setting 白名单键的端点。
 */
export function CheckinAdminCard() {
  const queryClient = useQueryClient()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['checkin-setting'],
    queryFn: getCheckinSetting,
  })

  const updateSetting = useMutation({
    mutationFn: updateCheckinSetting,
    onSuccess: (response) => {
      queryClient.setQueryData(['checkin-setting'], response)
    },
  })

  // 读不到（网络错误等）就不渲染，防止展示假默认值
  if (isError) return null
  const setting = isLoading ? undefined : data?.data
  if (!isLoading && !setting) return null

  const defaults: Values = setting
    ? {
        enabled: setting.enabled,
        minQuota: setting.min_quota,
        maxQuota: setting.max_quota,
      }
    : { enabled: false, minQuota: 0, maxQuota: 0 }

  return (
    <CheckinAdminCardForm
      key={isLoading ? 'loading' : JSON.stringify(defaults)}
      defaults={defaults}
      loading={isLoading}
      saving={updateSetting.isPending}
      onSave={async (values) => {
        try {
          await updateSetting.mutateAsync({
            enabled: values.enabled,
            min_quota: values.minQuota,
            max_quota: values.maxQuota,
          })
          return true
        } catch {
          // 失败提示由 http-client 拦截器统一弹出
          return false
        }
      }}
    />
  )
}

function CheckinAdminCardForm({
  defaults,
  loading,
  saving,
  onSave,
}: {
  defaults: Values
  loading: boolean
  saving: boolean
  onSave: (values: Values) => Promise<boolean>
}) {
  const { t } = useTranslation()

  const form = useForm<Values>({
    resolver: zodResolver(schema) as unknown as Resolver<Values>,
    defaultValues: defaults,
  })

  const { isDirty, isSubmitting } = form.formState
  const enabled = form.watch('enabled')

  async function onSubmit(values: Values) {
    const ok = await onSave(values)
    if (ok) form.reset(values)
  }

  if (loading) {
    return (
      <TitledCard
        title={t('Check-in Rewards')}
        icon={<CalendarCheck className='h-4 w-4' />}
        iconTone='success'
        disableHoverEffect
      >
        <div className='flex flex-col gap-3'>
          <Skeleton className='h-9 w-full' />
          <Skeleton className='h-9 w-full sm:w-2/3' />
        </div>
      </TitledCard>
    )
  }

  return (
    <TitledCard
      title={t('Check-in Rewards')}
      description={t(
        'Allow users to check in daily for random quota rewards'
      )}
      icon={<CalendarCheck className='h-4 w-4' />}
      iconTone='success'
      disableHoverEffect
      action={
        <Button
          type='submit'
          form='wallet-checkin-admin-form'
          size='sm'
          disabled={!isDirty || saving || isSubmitting}
        >
          {t('Save check-in settings')}
        </Button>
      }
    >
      <Form {...form}>
        <form
          id='wallet-checkin-admin-form'
          onSubmit={form.handleSubmit(onSubmit)}
          autoComplete='off'
          className='flex flex-col gap-5'
        >
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between rounded-lg border p-3 sm:p-4'>
                <div className='space-y-0.5'>
                  <FormLabel>{t('Enable check-in feature')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Allow users to check in daily for random quota rewards'
                    )}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={saving || isSubmitting}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          {enabled && (
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='minQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum check-in quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder={t('1000')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Minimum quota amount awarded for check-in')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='maxQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Maximum check-in quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder={t('10000')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Maximum quota amount awarded for check-in')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          )}
        </form>
      </Form>
    </TitledCard>
  )
}
