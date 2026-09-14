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
import { ChevronDown, ChevronRight, Search } from 'lucide-react'
import {
  Fragment,
  type FormEvent,
  useCallback,
  useEffect,
  useState,
} from 'react'
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
import { formatTimestampToDate } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { queryCustomerPrice, searchCustomers } from '../api'
import {
  CUSTOMER_AUDIT_VERDICT_CONFIG,
  DISCOUNT_CUSTOMER_AUDIT_LIMITS,
  DISCOUNT_SIMULATE_LIMITS,
  ERROR_MESSAGES,
  getDiscountBindingSourceLabel,
  getDiscountPlanStatusLabel,
  getDiscountRuleScopeLabel,
  getDiscountSourceLabel,
} from '../constants'
import { formatMarginText, formatRatioText } from '../lib'
import type {
  CustomerPriceBookEntry,
  CustomerPriceQueryResult,
  DiscountCustomer,
} from '../types'
import { ModelListField } from './model-list-field'

type DiscountsPriceQueryDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/** 价目本里一套价的卡片：谁挂的、管哪些模型、在不在生效期。 */
function PriceBookCard({ entry }: { entry: CustomerPriceBookEntry }) {
  const { t } = useTranslation()
  const effectiveFrom =
    entry.effective_from > 0
      ? formatTimestampToDate(entry.effective_from)
      : t('Unlimited')
  const effectiveTo =
    entry.effective_to > 0
      ? formatTimestampToDate(entry.effective_to)
      : t('Unlimited')

  return (
    <div className='rounded-md border p-3'>
      <div className='flex flex-wrap items-center gap-2'>
        <span className='text-sm font-medium'>
          {entry.plan_name || t('Plan deleted')}
        </span>
        <StatusBadge copyable={false} variant='neutral'>
          {getDiscountPlanStatusLabel(t, entry.plan_status)}
        </StatusBadge>
        <StatusBadge copyable={false} variant='neutral'>
          {getDiscountBindingSourceLabel(t, entry.source)}
        </StatusBadge>
        {!entry.in_window && (
          <StatusBadge copyable={false} variant='warning'>
            {entry.window_reason}
          </StatusBadge>
        )}
      </div>
      <p className='text-muted-foreground mt-1 text-xs'>
        {t('Base discount')}: {formatRatioText(entry.base_discount)}
        {' · '}
        {t('Effective from')}: {effectiveFrom}
        {' · '}
        {t('Effective to')}: {effectiveTo}
      </p>
      <ul className='text-muted-foreground mt-2 space-y-1 text-xs'>
        {entry.rules.length === 0 ? (
          <li>{t('No rules: the base discount covers every model.')}</li>
        ) : (
          entry.rules.map((rule) => (
            <li
              key={`${entry.binding_id}-${rule.scope_type}-${rule.scope_value}-${rule.priority}-${rule.discount}`}
            >
              {getDiscountRuleScopeLabel(t, rule.scope_type)}
              {' · '}
              {rule.scope_value}
              {' · '}
              {formatRatioText(rule.discount)}
              {rule.status === 0 && ` · ${t('Disabled')}`}
            </li>
          ))
        )}
      </ul>
    </div>
  )
}

/**
 * 客户查价抽屉：选一个客户，先看他身上挂了哪几套价（价目本），
 * 再填要问的模型，逐个模型看现价按几折、命中的是哪套、其余几套输在哪一层、
 * 还有几条线路、毛利多少。
 * 只读——不落库，也不改任何价。
 */
export function DiscountsPriceQueryDrawer({
  open,
  onOpenChange,
}: DiscountsPriceQueryDrawerProps) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [customers, setCustomers] = useState<DiscountCustomer[]>([])
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(false)
  const [selectedCustomerId, setSelectedCustomerId] = useState('')
  const [selectedModels, setSelectedModels] = useState<string[]>([])
  const [isQuerying, setIsQuerying] = useState(false)
  const [result, setResult] = useState<CustomerPriceQueryResult | null>(null)
  const [formError, setFormError] = useState<string | null>(null)
  const [expandedModels, setExpandedModels] = useState<string[]>([])

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
      setSelectedModels([])
      setResult(null)
      setFormError(null)
      setExpandedModels([])
      return
    }
    void loadCustomers('')
  }, [open, loadCustomers])

  const runQuery = useCallback(
    async (customerId: number, models: string[]) => {
      setIsQuerying(true)
      setFormError(null)
      try {
        const response = await queryCustomerPrice({
          userId: customerId,
          models,
        })
        if (response.success && response.data) {
          setResult(response.data)
          setExpandedModels([])
        } else {
          setResult(null)
          setFormError(response.message || t(ERROR_MESSAGES.PRICE_QUERY_FAILED))
        }
      } catch (error) {
        setResult(null)
        handleServerError(error)
      } finally {
        setIsQuerying(false)
      }
    },
    [t]
  )

  const handleCustomerChange = (value: string) => {
    setSelectedCustomerId(value)
    setResult(null)
    setFormError(null)
    setExpandedModels([])
    const customerId = Number.parseInt(value, 10)
    if (!Number.isFinite(customerId) || customerId <= 0) {
      return
    }
    // 选完客户立刻把他的价目本拉出来：先看清身上有几套价，再决定问哪个模型。
    void runQuery(customerId, [])
  }

  const handleQuery = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const customerId = Number.parseInt(selectedCustomerId, 10)
    const models = selectedModels

    if (!Number.isFinite(customerId) || customerId <= 0) {
      setFormError(t(ERROR_MESSAGES.CUSTOMER_REQUIRED))
      return
    }
    if (models.length > DISCOUNT_CUSTOMER_AUDIT_LIMITS.MODELS_MAX_COUNT) {
      setFormError(t(ERROR_MESSAGES.MODELS_TOO_MANY))
      return
    }

    await runQuery(customerId, models)
  }

  const toggleModel = (model: string) => {
    setExpandedModels((current) =>
      current.includes(model)
        ? current.filter((item) => item !== model)
        : [...current, model]
    )
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen)
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[1000px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Customer Price Check')}</SheetTitle>
          <SheetDescription>
            {t(
              'Pick a customer to see every price on him, which one applies to each model, and why the others lost.'
            )}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-price-query-form'
          onSubmit={handleQuery}
          className={sideDrawerFormClassName()}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Customer')} />
            <div className='flex items-end gap-2'>
              <div className='flex-1 space-y-2'>
                <Label htmlFor='discount-price-query-customer-keyword'>
                  {t('Search customers by username')}
                </Label>
                <Input
                  id='discount-price-query-customer-keyword'
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
              onValueChange={(value) => handleCustomerChange(value ?? '')}
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

          {result && (
            <SideDrawerSection>
              <SideDrawerSectionHeader
                title={t('Price Book')}
                description={t(
                  'Every price on this customer, who put it there, and which models it covers.'
                )}
              />
              {result.price_book.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No plans on this customer')}
                </p>
              ) : (
                <div className='space-y-3'>
                  {result.price_book.map((entry) => (
                    <PriceBookCard key={entry.binding_id} entry={entry} />
                  ))}
                </div>
              )}
            </SideDrawerSection>
          )}

          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Model List')} />
            <ModelListField
              value={selectedModels}
              onChange={setSelectedModels}
              max={DISCOUNT_CUSTOMER_AUDIT_LIMITS.MODELS_MAX_COUNT}
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'Leave it empty to look at the price book only; a single lookup covers at most 100 models.'
              )}
            </p>
          </SideDrawerSection>

          {formError && (
            <Alert variant='destructive'>
              <AlertTitle>{t('Price lookup failed')}</AlertTitle>
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          )}

          {result && result.warnings.length > 0 && (
            <Alert>
              <AlertTitle>{t('Warnings')}</AlertTitle>
              <AlertDescription>
                <ul className='list-disc space-y-1 pl-4'>
                  {result.warnings.map((warning) => (
                    <li key={warning}>{warning}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          )}

          {result && result.models.length > 0 && (
            <>
              <Alert
                variant={result.summary.signable ? 'default' : 'destructive'}
              >
                <AlertTitle>{t('Conclusion')}</AlertTitle>
                <AlertDescription>{result.summary.conclusion}</AlertDescription>
              </Alert>

              <SideDrawerSection>
                <SideDrawerSectionHeader
                  title={t('Per-model prices')}
                  description={t(
                    'Gross margin = customer discount ÷ cheapest cost ratio − 1.'
                  )}
                />
                <div className='overflow-x-auto'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead className='w-[40px]' />
                        <TableHead>{t('Model')}</TableHead>
                        <TableHead>{t('Applied Discount')}</TableHead>
                        <TableHead>{t('Plan')}</TableHead>
                        <TableHead>{t('Routes')}</TableHead>
                        <TableHead>{t('Cheapest cost ratio')}</TableHead>
                        <TableHead>{t('Gross Margin')}</TableHead>
                        <TableHead>{t('Verdict')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {result.models.map((row) => {
                        const verdict =
                          CUSTOMER_AUDIT_VERDICT_CONFIG[row.verdict]
                        const isExpanded = expandedModels.includes(row.model)
                        return (
                          <Fragment key={row.model}>
                            <TableRow>
                              <TableCell>
                                {row.candidates.length > 0 && (
                                  <Button
                                    type='button'
                                    variant='ghost'
                                    size='icon'
                                    className='h-6 w-6'
                                    aria-expanded={isExpanded}
                                    aria-label={t('Show candidates')}
                                    onClick={() => toggleModel(row.model)}
                                  >
                                    {isExpanded ? (
                                      <ChevronDown className='h-4 w-4' />
                                    ) : (
                                      <ChevronRight className='h-4 w-4' />
                                    )}
                                  </Button>
                                )}
                              </TableCell>
                              <TableCell className='font-medium'>
                                {row.model}
                              </TableCell>
                              <TableCell>
                                <span className='font-medium'>
                                  {formatRatioText(row.discount)}
                                </span>
                                <p className='text-muted-foreground text-xs'>
                                  {getDiscountSourceLabel(t, row.source)}
                                </p>
                              </TableCell>
                              <TableCell>
                                {row.plan
                                  ? `${row.plan.name} · ${getDiscountPlanStatusLabel(t, row.plan.status)}`
                                  : t('No plan bound')}
                              </TableCell>
                              <TableCell>
                                {row.channel_count === 0 ? (
                                  <span className='text-muted-foreground'>
                                    {t('No routes')}
                                  </span>
                                ) : (
                                  `${row.usable_count}/${row.channel_count}`
                                )}
                              </TableCell>
                              <TableCell>
                                {row.cheapest_cost ? (
                                  formatRatioText(row.cheapest_cost)
                                ) : (
                                  <span className='text-muted-foreground'>
                                    {t('Not recorded')}
                                  </span>
                                )}
                              </TableCell>
                              <TableCell>
                                {row.gross_margin ? (
                                  formatMarginText(row.gross_margin)
                                ) : (
                                  <span className='text-muted-foreground'>
                                    {t('Not recorded')}
                                  </span>
                                )}
                              </TableCell>
                              <TableCell>
                                {verdict ? (
                                  <StatusBadge
                                    copyable={false}
                                    variant={verdict.variant}
                                  >
                                    {t(verdict.labelKey)}
                                  </StatusBadge>
                                ) : (
                                  row.verdict
                                )}
                                <p className='text-muted-foreground mt-1 max-w-[240px] text-xs'>
                                  {row.verdict_detail}
                                </p>
                              </TableCell>
                            </TableRow>
                            {isExpanded && (
                              <TableRow>
                                <TableCell
                                  colSpan={8}
                                  className='bg-muted/40 whitespace-normal'
                                >
                                  <p className='text-muted-foreground mb-2 text-xs'>
                                    {t(
                                      'Each price on this customer, and how this model resolved.'
                                    )}
                                  </p>
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
                                      {row.candidates.map((candidate) => (
                                        <TableRow
                                          key={`${row.model}-${candidate.plan_id}-${candidate.specificity}-${candidate.source}`}
                                        >
                                          <TableCell>
                                            {candidate.plan_name || '-'}
                                          </TableCell>
                                          <TableCell>
                                            {formatRatioText(
                                              candidate.discount
                                            )}
                                          </TableCell>
                                          <TableCell>
                                            {getDiscountBindingSourceLabel(
                                              t,
                                              candidate.source
                                            )}
                                          </TableCell>
                                          <TableCell>
                                            {candidate.applied ? (
                                              <StatusBadge
                                                copyable={false}
                                                variant='success'
                                              >
                                                {t('Applied')}
                                              </StatusBadge>
                                            ) : (
                                              <span className='text-muted-foreground text-xs'>
                                                {candidate.reject_reason ||
                                                  t('Not applied')}
                                              </span>
                                            )}
                                          </TableCell>
                                        </TableRow>
                                      ))}
                                    </TableBody>
                                  </Table>
                                </TableCell>
                              </TableRow>
                            )}
                          </Fragment>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              </SideDrawerSection>
            </>
          )}
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          <Button
            form='discount-price-query-form'
            type='submit'
            disabled={isQuerying}
          >
            {isQuerying ? t('Looking up...') : t('Look up prices')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
