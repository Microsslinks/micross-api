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
import { type FormEvent, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  SideDrawerSection,
  SideDrawerSectionHeader,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { handleServerError } from '@/lib/handle-server-error'

import {
  createDiscountPlan,
  updateDiscountPlan,
  validateDiscountPlan,
} from '../api'
import {
  ERROR_MESSAGES,
  getDiscountBillingModeOptions,
  getDiscountOwnerOptions,
  getDiscountRuleScopeLabel,
  getDiscountStatusOptions,
  getDiscountViolationReasonLabel,
} from '../constants'
import {
  DISCOUNT_PLAN_FORM_DEFAULT_VALUES,
  buildDiscountPlanPayload,
  formatRatioText,
  getDiscountPlanFormSchema,
  transformDiscountPlanToFormDefaults,
  type DiscountPlanFormValues,
} from '../lib'
import {
  DISCOUNT_OWNER,
  DISCOUNT_PLAN_STATUS,
  type DiscountPlanPayload,
  type DiscountValidateResult,
} from '../types'
import { useDiscounts } from './discounts-provider'

type DiscountsPlanMutateDrawerProps = {
  open: boolean
  /** 新建还是改已有的：只有改已有的时候才能跑保存前检查（检查要方案 ID）。 */
  isUpdate: boolean
  onOpenChange: (open: boolean) => void
}

export function DiscountsPlanMutateDrawer({
  open,
  isUpdate,
  onOpenChange,
}: DiscountsPlanMutateDrawerProps) {
  const { t } = useTranslation()
  const { currentRow, triggerRefresh } = useDiscounts()
  const [values, setValues] = useState<DiscountPlanFormValues>(
    DISCOUNT_PLAN_FORM_DEFAULT_VALUES
  )
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [isSaving, setIsSaving] = useState(false)
  /** 保存前检查的结果：有它就说明「有话说」，按钮改成「仍然保存」。 */
  const [checkResult, setCheckResult] = useState<DiscountValidateResult | null>(
    null
  )
  const [checkFailed, setCheckFailed] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    if (isUpdate && currentRow) {
      setValues(transformDiscountPlanToFormDefaults(currentRow))
    } else {
      setValues(DISCOUNT_PLAN_FORM_DEFAULT_VALUES)
    }
    setErrors({})
    setCheckResult(null)
    setCheckFailed(null)
  }, [open, isUpdate, currentRow])

  const setField = <K extends keyof DiscountPlanFormValues>(
    field: K,
    value: DiscountPlanFormValues[K]
  ) => {
    setValues((previous) => ({ ...previous, [field]: value }))
    setErrors((previous) => {
      if (!previous[field]) return previous
      const next = { ...previous }
      delete next[field]
      return next
    })
  }

  const save = async (payload: DiscountPlanPayload, planId: number | null) => {
    setIsSaving(true)
    try {
      const result =
        planId === null
          ? await createDiscountPlan(payload)
          : await updateDiscountPlan(planId, payload)

      if (result.success) {
        toast.success(planId === null ? t('Plan created') : t('Plan saved'))
        onOpenChange(false)
        triggerRefresh()
        return
      }
      toast.error(result.message || t(ERROR_MESSAGES.PLAN_SAVE_FAILED))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSaving(false)
    }
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const parsed = getDiscountPlanFormSchema(t).safeParse(values)
    if (!parsed.success) {
      const nextErrors: Record<string, string> = {}
      for (const issue of parsed.error.issues) {
        const field = String(issue.path[0] ?? '')
        if (field && !nextErrors[field]) nextErrors[field] = issue.message
      }
      setErrors(nextErrors)
      return
    }
    setErrors({})

    const payload = buildDiscountPlanPayload(values)
    if (!isUpdate || !currentRow) {
      await save(payload, null)
      return
    }

    // 第一次点保存：先问一遍后端这套折扣会不会亏。
    // 有话说就摆出来等运营再点一次，**不阻断**——亏本促销与大客户见面礼都是真实业务。
    if (!checkResult) {
      setIsSaving(true)
      try {
        const check = await validateDiscountPlan(currentRow.id)
        if (check.success && check.data) {
          setCheckFailed(null)
          if (
            check.data.violations.length > 0 ||
            check.data.warnings.length > 0
          ) {
            setCheckResult(check.data)
            return
          }
        } else {
          // 检查跑不起来（网络或后端报错）时不能假装没问题，但也不该拦住保存。
          setCheckFailed(
            check.message || t(ERROR_MESSAGES.PLAN_VALIDATE_FAILED)
          )
        }
      } catch (error) {
        setCheckFailed(t(ERROR_MESSAGES.PLAN_VALIDATE_FAILED))
        handleServerError(error)
      } finally {
        setIsSaving(false)
      }
    }

    await save(payload, currentRow.id)
  }

  const ownerOptions = getDiscountOwnerOptions(t)
  const statusOptions = getDiscountStatusOptions(t)
  const billingModeOptions = getDiscountBillingModeOptions(t)

  let submitLabel = t('Save')
  if (checkResult) submitLabel = t('Save anyway')
  if (isSaving) submitLabel = t('Saving...')

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onOpenChange(false)
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[560px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{isUpdate ? t('Edit plan') : t('New plan')}</SheetTitle>
          <SheetDescription>
            {t(
              'A plan gives one customer a different price than the official one.'
            )}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-plan-form'
          onSubmit={handleSubmit}
          className={sideDrawerFormClassName()}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Basic information')} />

            <div className='space-y-2'>
              <Label htmlFor='discount-plan-name'>{t('Name')}</Label>
              <Input
                id='discount-plan-name'
                value={values.name}
                placeholder={t('For example: Acme annual contract')}
                onChange={(event) => setField('name', event.target.value)}
              />
              {errors.name && (
                <p className='text-destructive text-xs'>{errors.name}</p>
              )}
            </div>

            <div className='space-y-2'>
              <Label>{t('Owner')}</Label>
              <ToggleGroup
                value={[values.owner_type]}
                onValueChange={(value) => {
                  // 单选：点已选中的那项等于取消，忽略掉。
                  const next = value.find((item) => item !== values.owner_type)
                  if (next) setField('owner_type', next)
                }}
                aria-label={t('Owner')}
                variant='outline'
                size='lg'
                spacing={2}
                className='grid w-full grid-cols-2 gap-2'
              >
                {ownerOptions.map((option) => (
                  <ToggleGroupItem
                    key={option.value}
                    value={option.value}
                    className='h-auto min-h-10 w-full'
                  >
                    {option.label}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
            </div>

            {values.owner_type === DISCOUNT_OWNER.AGENT && (
              <div className='space-y-2'>
                <Label htmlFor='discount-plan-owner-id'>
                  {t('Agent user ID')}
                </Label>
                <Input
                  id='discount-plan-owner-id'
                  type='number'
                  value={values.owner_id === 0 ? '' : String(values.owner_id)}
                  onChange={(event) =>
                    setField(
                      'owner_id',
                      Number.parseInt(event.target.value, 10) || 0
                    )
                  }
                />
                {errors.owner_id && (
                  <p className='text-destructive text-xs'>{errors.owner_id}</p>
                )}
              </div>
            )}

            <div className='space-y-2'>
              <Label>{t('Status')}</Label>
              <ToggleGroup
                value={[String(values.status)]}
                onValueChange={(value) => {
                  // 单选：点已选中的那项等于取消，忽略掉。
                  const next = value.find(
                    (item) => item !== String(values.status)
                  )
                  if (next === undefined) return
                  setField(
                    'status',
                    Number(next) === DISCOUNT_PLAN_STATUS.DISABLED
                      ? DISCOUNT_PLAN_STATUS.DISABLED
                      : DISCOUNT_PLAN_STATUS.ENABLED
                  )
                }}
                aria-label={t('Status')}
                variant='outline'
                size='lg'
                spacing={2}
                className='grid w-full grid-cols-2 gap-2'
              >
                {statusOptions.map((option) => (
                  <ToggleGroupItem
                    key={option.value}
                    value={option.value}
                    className='h-auto min-h-10 w-full'
                  >
                    {option.label}
                  </ToggleGroupItem>
                ))}
              </ToggleGroup>
            </div>
          </SideDrawerSection>

          <SideDrawerSection>
            <SideDrawerSectionHeader
              title={t('Discounts')}
              description={t(
                'Discounts are ratios between 0 and 1: 0.3 means the customer pays 30% of the official price. Rules set which models receive the base discount.'
              )}
            />

            <div className='space-y-2'>
              <Label htmlFor='discount-plan-base'>{t('Base discount')}</Label>
              <Input
                id='discount-plan-base'
                value={values.base_discount}
                onChange={(event) =>
                  setField('base_discount', event.target.value)
                }
              />
              {errors.base_discount && (
                <p className='text-destructive text-xs'>
                  {errors.base_discount}
                </p>
              )}
              <p className='text-muted-foreground text-xs'>
                {t(
                  'The discount applied to every model covered by a rule. Add rules in the next step to scope which models receive this price.'
                )}
              </p>
            </div>

            <div className='space-y-2'>
              <Label htmlFor='discount-plan-min'>{t('Minimum discount')}</Label>
              <Input
                id='discount-plan-min'
                value={values.min_discount}
                onChange={(event) =>
                  setField('min_discount', event.target.value)
                }
              />
              {errors.min_discount && (
                <p className='text-destructive text-xs'>
                  {errors.min_discount}
                </p>
              )}
              <p className='text-muted-foreground text-xs'>
                {t('The lowest discount this plan allows.')}
              </p>
            </div>

            <div className='space-y-2'>
              <Label>{t('Plan billing mode')}</Label>
              <Select
                items={billingModeOptions}
                value={values.billing_mode}
                onValueChange={(value) =>
                  setField('billing_mode', String(value ?? ''))
                }
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Plan billing mode')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {billingModeOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Recorded for now; billing still charges by usage until it is wired up.'
                )}
              </p>
            </div>

            {values.owner_type === DISCOUNT_OWNER.AGENT && (
              <div className='space-y-2'>
                <Label htmlFor='discount-plan-commission'>
                  {t('Commission ratio')}
                </Label>
                <Input
                  id='discount-plan-commission'
                  value={values.commission_ratio}
                  onChange={(event) =>
                    setField('commission_ratio', event.target.value)
                  }
                />
                {errors.commission_ratio && (
                  <p className='text-destructive text-xs'>
                    {errors.commission_ratio}
                  </p>
                )}
              </div>
            )}

            {/*
             * task-09：发放折算比例。所有方案都可填（与 owner 无关）—— 后端 NormalizeDefaults
             * 把每个方案的 topup_conversion_rate 兜到 1.0，缺省就是 1:1 行为不变。
             * 实际会被经销商用的是 agent 方案的字段；平台方案的字段会保持 1.0，因为不是
             * 经销商定的价——但保留字段可以避免 owner 切换（platform <-> agent）时丢值。
             */}
            <div className='space-y-2'>
              <Label htmlFor='discount-plan-topup-rate'>
                {t('Topup conversion rate')}
              </Label>
              <Input
                id='discount-plan-topup-rate'
                type='number'
                inputMode='decimal'
                step='0.001'
                min='0'
                max='1'
                value={values.topup_conversion_rate}
                placeholder='1'
                onChange={(event) =>
                  setField('topup_conversion_rate', event.target.value)
                }
              />
              {errors.topup_conversion_rate && (
                <p className='text-destructive text-xs'>
                  {errors.topup_conversion_rate}
                </p>
              )}
              <p className='text-muted-foreground text-xs'>
                {t(
                  'Ratio applied when this plan is used to top up a customer: 0.875 means 100 face value costs the dealer 87.5. Defaults to 1.0 (no conversion).'
                )}
              </p>
            </div>
          </SideDrawerSection>

          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Remark')} />
            <Input
              value={values.remark}
              placeholder={t('Who asked for this plan, and why')}
              onChange={(event) => setField('remark', event.target.value)}
            />
            {errors.remark && (
              <p className='text-destructive text-xs'>{errors.remark}</p>
            )}
          </SideDrawerSection>

          {!isUpdate && (
            <Alert>
              <AlertTitle>{t('Pre-save check')}</AlertTitle>
              <AlertDescription>
                {t(
                  'The check needs a saved plan, so it runs the next time you save this plan after adding its rules.'
                )}
              </AlertDescription>
            </Alert>
          )}

          {checkFailed && (
            <Alert>
              <AlertTitle>{t('Pre-save check did not run')}</AlertTitle>
              <AlertDescription>{checkFailed}</AlertDescription>
            </Alert>
          )}

          {checkResult && (
            <Alert variant='destructive'>
              <AlertTitle>{t('This plan may lose money')}</AlertTitle>
              <AlertDescription>
                <p>
                  {t(
                    'Saving is not blocked. Fix the problems below, or save anyway if the loss is intentional.'
                  )}
                </p>
                {checkResult.violations.length > 0 && (
                  <ul className='list-disc space-y-1 pl-4'>
                    {checkResult.violations.map((violation) => (
                      <li
                        key={`${violation.scope_type}-${violation.scope_value}-${violation.reason}`}
                      >
                        <span className='font-medium'>
                          {getDiscountRuleScopeLabel(t, violation.scope_type)}{' '}
                          {violation.scope_value}
                        </span>
                        {' · '}
                        {formatRatioText(violation.discount)}
                        {' · '}
                        <span>
                          {getDiscountViolationReasonLabel(t, violation.reason)}
                        </span>
                        <div>{violation.detail}</div>
                        {violation.available_channels.length > 0 && (
                          <div>
                            {t('Routes that could still work:')}{' '}
                            {violation.available_channels.join(', ')}
                          </div>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
                {checkResult.warnings.length > 0 && (
                  <ul className='list-disc space-y-1 pl-4'>
                    {checkResult.warnings.map((warning) => (
                      <li key={warning}>{warning}</li>
                    ))}
                  </ul>
                )}
                <p>
                  {t('Profit floor:')}{' '}
                  {formatRatioText(checkResult.min_margin_ratio)}
                </p>
              </AlertDescription>
            </Alert>
          )}
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Cancel')}
          </SheetClose>
          <Button
            form='discount-plan-form'
            type='submit'
            disabled={isSaving}
            variant={checkResult ? 'destructive' : 'default'}
          >
            {submitLabel}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
