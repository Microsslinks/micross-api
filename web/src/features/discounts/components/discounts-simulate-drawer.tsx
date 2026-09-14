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
import { Search } from 'lucide-react'
import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  SideDrawerSection,
  SideDrawerSectionHeader,
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { StatusBadge } from '@/components/status-badge'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { ComboboxInput } from '@/components/ui/combobox-input'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { handleServerError } from '@/lib/handle-server-error'

import { searchCustomers, simulateDiscount } from '../api'
import {
  DISCOUNT_SIMULATE_LIMITS,
  ERROR_MESSAGES,
  getDiscountBindingSourceLabel,
  getDiscountPlanStatusLabel,
  getDiscountRuleScopeLabel,
  getDiscountSourceLabel,
} from '../constants'
import { useModelNameOptions } from '../hooks/use-model-name-options'
import { formatMarginText, formatRatioText } from '../lib'
import type { DiscountCustomer, DiscountSimulateResult } from '../types'

type DiscountsSimulateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * 试算抽屉：选一个客户、填一个模型名，看这个客户按几折、会走哪几条线路、每条赚多少。
 * 只读——试算结果不落库，后端也不改路由。
 */
export function DiscountsSimulateDrawer({
  open,
  onOpenChange,
}: DiscountsSimulateDrawerProps) {
  const { t } = useTranslation()
  const { options } = useModelNameOptions()
  const [keyword, setKeyword] = useState('')
  const [customers, setCustomers] = useState<DiscountCustomer[]>([])
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(false)
  const [selectedCustomerId, setSelectedCustomerId] = useState('')
  const [modelName, setModelName] = useState('')
  const [isSimulating, setIsSimulating] = useState(false)
  const [result, setResult] = useState<DiscountSimulateResult | null>(null)
  const [formError, setFormError] = useState<string | null>(null)

  const loadCustomers = useCallback(
    async (searchKeyword: string) => {
      setIsLoadingCustomers(true)
      try {
        const response = await searchCustomers({
          keyword: searchKeyword,
          page_size: DISCOUNT_SIMULATE_LIMITS.CUSTOMER_PAGE_SIZE,
        })
        if (response.success) {
          setCustomers(response.data?.items ?? [])
        } else {
          setCustomers([])
          setFormError(
            response.message || t(ERROR_MESSAGES.CUSTOMERS_LOAD_FAILED)
          )
        }
      } catch (error) {
        setCustomers([])
        handleServerError(error)
      } finally {
        setIsLoadingCustomers(false)
      }
    },
    [t]
  )

  useEffect(() => {
    if (!open) {
      // 关掉就把上一次的输入和结果清干净，下次打开不会挂着旧数据。
      setKeyword('')
      setCustomers([])
      setSelectedCustomerId('')
      setModelName('')
      setResult(null)
      setFormError(null)
      return
    }
    void loadCustomers('')
  }, [open, loadCustomers])

  const handleSimulate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const trimmedModel = modelName.trim()
    const customerId = Number.parseInt(selectedCustomerId, 10)

    if (!Number.isFinite(customerId) || customerId <= 0) {
      setFormError(t(ERROR_MESSAGES.CUSTOMER_REQUIRED))
      return
    }
    if (!trimmedModel) {
      setFormError(t(ERROR_MESSAGES.MODEL_REQUIRED))
      return
    }
    if (trimmedModel.length > DISCOUNT_SIMULATE_LIMITS.MODEL_NAME_MAX_LENGTH) {
      setFormError(t(ERROR_MESSAGES.MODEL_TOO_LONG))
      return
    }

    setFormError(null)
    setIsSimulating(true)
    try {
      const response = await simulateDiscount({
        userId: customerId,
        model: trimmedModel,
      })
      if (response.success && response.data) {
        setResult(response.data)
      } else {
        setResult(null)
        setFormError(response.message || t(ERROR_MESSAGES.SIMULATE_FAILED))
      }
    } catch (error) {
      setResult(null)
      handleServerError(error)
    } finally {
      setIsSimulating(false)
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen)
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[640px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Discount Simulation')}</SheetTitle>
          <SheetDescription>
            {t(
              "Pick a customer and a model to see the applied discount and each upstream route's margin."
            )}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-simulate-form'
          onSubmit={handleSimulate}
          className={sideDrawerFormClassName()}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Customer')} />
            <div className='flex items-end gap-2'>
              <div className='flex-1 space-y-2'>
                <Label htmlFor='discount-customer-keyword'>
                  {t('Search customers by username')}
                </Label>
                <Input
                  id='discount-customer-keyword'
                  value={keyword}
                  placeholder={t('Search customers by username')}
                  onChange={(event) => setKeyword(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') {
                      event.preventDefault()
                      void loadCustomers(keyword)
                    }
                  }}
                />
              </div>
              <Button
                type='button'
                variant='outline'
                disabled={isLoadingCustomers}
                onClick={() => void loadCustomers(keyword)}
              >
                <Search className='h-4 w-4' />
                {isLoadingCustomers ? t('Searching...') : t('Search')}
              </Button>
            </div>
            <Select
              items={customers.map((customer) => ({
                value: String(customer.id),
                label: customer.username,
              }))}
              value={selectedCustomerId}
              onValueChange={(value) => setSelectedCustomerId(value ?? '')}
            >
              <SelectTrigger className='w-full'>
                <SelectValue placeholder={t('Select a customer')} />
              </SelectTrigger>
              <SelectContent alignItemWithTrigger={false}>
                <SelectGroup>
                  {customers.map((customer) => (
                    <SelectItem key={customer.id} value={String(customer.id)}>
                      {customer.display_name
                        ? `${customer.username} (${customer.display_name})`
                        : customer.username}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
            {!isLoadingCustomers && customers.length === 0 && (
              <p className='text-muted-foreground text-xs'>
                {t('No customers found')}
              </p>
            )}
          </SideDrawerSection>

          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Model name')} />
            <ComboboxInput
              value={modelName}
              onValueChange={setModelName}
              options={options}
              placeholder={t('Enter a model name to simulate')}
              allowCustomValue
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'A trailing wildcard is supported, e.g. claude-*; the model must be billed to this customer.'
              )}
            </p>
          </SideDrawerSection>

          {formError && (
            <Alert variant='destructive'>
              <AlertTitle>{t('Simulation failed')}</AlertTitle>
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          )}

          {result && (
            <>
              <SideDrawerSection>
                <SideDrawerSectionHeader title={t('Result')} />
                <dl className='grid grid-cols-2 gap-3 text-sm'>
                  <div className='space-y-1'>
                    <dt className='text-muted-foreground'>{t('Customer')}</dt>
                    <dd className='font-medium'>{result.user.username}</dd>
                  </div>
                  <div className='space-y-1'>
                    <dt className='text-muted-foreground'>{t('Group')}</dt>
                    <dd className='font-medium'>{result.user.group || '-'}</dd>
                  </div>
                  <div className='space-y-1'>
                    <dt className='text-muted-foreground'>{t('Model')}</dt>
                    <dd className='font-medium'>{result.model}</dd>
                  </div>
                  <div className='space-y-1'>
                    <dt className='text-muted-foreground'>{t('Vendor')}</dt>
                    <dd className='font-medium'>{result.vendor || '-'}</dd>
                  </div>
                  <div className='space-y-1'>
                    <dt className='text-muted-foreground'>{t('Plan')}</dt>
                    <dd className='font-medium'>
                      {result.plan
                        ? `${result.plan.name} · ${getDiscountPlanStatusLabel(t, result.plan.status)}`
                        : t('No plan bound')}
                    </dd>
                  </div>
                  <div className='space-y-1'>
                    <dt className='text-muted-foreground'>
                      {t('Profit Floor')}
                    </dt>
                    <dd className='font-medium'>
                      {formatRatioText(result.min_margin_ratio)}
                    </dd>
                  </div>
                </dl>
              </SideDrawerSection>

              <SideDrawerSection>
                <SideDrawerSectionHeader
                  title={t('Applied Discount')}
                  description={t('Discount Source')}
                />
                <div className='flex items-baseline gap-2'>
                  <span className='text-2xl font-semibold'>
                    {formatRatioText(result.resolution.discount)}
                  </span>
                  <span className='text-muted-foreground text-sm'>
                    {getDiscountSourceLabel(t, result.resolution.source)}
                  </span>
                </div>
                {result.resolution.matched_rule && (
                  // 规则命中时按方案基础折扣出价：上面的「Applied Discount」已经展示数字，
                  // 这里只说清「命中的是哪条规则的哪个范围」。rule.Discount 已废弃，不再单独展示。
                  <p className='text-muted-foreground text-xs'>
                    {t('Matched Rule')}
                    {': '}
                    {getDiscountRuleScopeLabel(
                      t,
                      result.resolution.matched_rule.scope_type
                    )}
                    {' · '}
                    {result.resolution.matched_rule.scope_value}
                    {' · '}
                    {t('Priority')} {result.resolution.matched_rule.priority}
                  </p>
                )}
              </SideDrawerSection>

              {result.candidates.length > 0 && (
                <SideDrawerSection>
                  <SideDrawerSectionHeader
                    title={t('Bound plans')}
                    description={t(
                      'Plans bound to this customer; the pricing rules pick one for each model.'
                    )}
                  />
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Plan')}</TableHead>
                        <TableHead>{t('Discount')}</TableHead>
                        <TableHead>{t('Source')}</TableHead>
                        <TableHead>{t('Result')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {result.candidates.map((candidate, index) => (
                        <TableRow key={`${candidate.plan_id}-${index}`}>
                          <TableCell>{candidate.plan_name || '-'}</TableCell>
                          <TableCell>
                            {formatRatioText(candidate.discount)}
                          </TableCell>
                          <TableCell>
                            {getDiscountBindingSourceLabel(t, candidate.source)}
                          </TableCell>
                          <TableCell>
                            {candidate.applied ? (
                              <StatusBadge copyable={false} variant='success'>
                                {t('Applied')}
                              </StatusBadge>
                            ) : (
                              <span className='text-muted-foreground text-xs'>
                                {candidate.reject_reason || t('Not applied')}
                              </span>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </SideDrawerSection>
              )}

              {!result.cost_known && (
                <Alert>
                  <AlertTitle>{t('Cost is unknown')}</AlertTitle>
                  <AlertDescription>
                    {t(
                      'At least one route has no cost ratio on record, so margins cannot be judged.'
                    )}
                  </AlertDescription>
                </Alert>
              )}

              <SideDrawerSection>
                <SideDrawerSectionHeader
                  title={t('Routes')}
                  description={t(
                    'Gross margin = customer discount ÷ cost ratio − 1.'
                  )}
                />
                {result.channels.length === 0 ? (
                  <p className='text-muted-foreground text-sm'>
                    {t('No routes available for this customer and model.')}
                  </p>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Route')}</TableHead>
                        <TableHead>{t('Cost Ratio')}</TableHead>
                        <TableHead>{t('Gross Margin')}</TableHead>
                        <TableHead>{t('Status')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {result.channels.map((channel) => (
                        <TableRow key={channel.channel_id}>
                          <TableCell>{channel.channel_name}</TableCell>
                          <TableCell>
                            {formatRatioText(channel.cost_ratio)}
                          </TableCell>
                          <TableCell>
                            {formatMarginText(channel.gross_margin)}
                          </TableCell>
                          <TableCell>
                            {channel.passes_floor === null ? (
                              <span className='text-muted-foreground'>
                                {t('Not recorded')}
                              </span>
                            ) : (
                              <StatusBadge
                                copyable={false}
                                variant={
                                  channel.passes_floor ? 'success' : 'danger'
                                }
                              >
                                {channel.passes_floor
                                  ? t('Break-even or better')
                                  : t('Below floor')}
                              </StatusBadge>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </SideDrawerSection>

              {result.warnings.length > 0 && (
                <SideDrawerSection>
                  <SideDrawerSectionHeader title={t('Warnings')} />
                  <ul className='text-muted-foreground list-disc space-y-1 pl-4 text-sm'>
                    {result.warnings.map((warning) => (
                      <li key={warning}>{warning}</li>
                    ))}
                  </ul>
                </SideDrawerSection>
              )}
            </>
          )}
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          <Button
            form='discount-simulate-form'
            type='submit'
            disabled={isSimulating}
          >
            {isSimulating ? t('Simulating...') : t('Simulate')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
