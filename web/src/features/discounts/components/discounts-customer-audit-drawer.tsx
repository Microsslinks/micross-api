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
import { Textarea } from '@/components/ui/textarea'
import { handleServerError } from '@/lib/handle-server-error'

import { auditCustomerPricing, searchCustomers } from '../api'
import {
  CUSTOMER_AUDIT_VERDICT_CONFIG,
  DISCOUNT_CUSTOMER_AUDIT_LIMITS,
  DISCOUNT_SIMULATE_LIMITS,
  ERROR_MESSAGES,
  getDiscountPlanStatusLabel,
  getDiscountSourceLabel,
} from '../constants'
import { formatMarginText, formatRatioText } from '../lib'
import type { CustomerAuditResult, DiscountCustomer } from '../types'

type DiscountsCustomerAuditDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/**
 * 把粘贴的清单文本拆成模型名数组：换行、中英文逗号、顿号都当分隔符，
 * 空白与重复项去掉，顺序保留（运营对着客户的单子一行行核对）。
 * 空格不当分隔符——模型名里允许有空格，拆碎就对不上线路了。
 */
function parseModelList(raw: string): string[] {
  const seen = new Set<string>()
  const names: string[] = []
  for (const piece of raw.split(/[\n\r,，、]/)) {
    const name = piece.trim()
    if (!name || seen.has(name)) {
      continue
    }
    seen.add(name)
    names.push(name)
  }
  return names
}

/**
 * 客户核算抽屉：选一个客户、贴一份模型清单，逐个模型看按几折、
 * 最便宜线路成本与毛利，最后一句话——这份报价能不能签。
 * 只读——核算结果不落库，后端也不改任何价。
 */
export function DiscountsCustomerAuditDrawer({
  open,
  onOpenChange,
}: DiscountsCustomerAuditDrawerProps) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [customers, setCustomers] = useState<DiscountCustomer[]>([])
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(false)
  const [selectedCustomerId, setSelectedCustomerId] = useState('')
  const [modelListText, setModelListText] = useState('')
  const [isAuditing, setIsAuditing] = useState(false)
  const [result, setResult] = useState<CustomerAuditResult | null>(null)
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
      setModelListText('')
      setResult(null)
      setFormError(null)
      return
    }
    void loadCustomers('')
  }, [open, loadCustomers])

  const handleAudit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const customerId = Number.parseInt(selectedCustomerId, 10)
    const models = parseModelList(modelListText)

    if (!Number.isFinite(customerId) || customerId <= 0) {
      setFormError(t(ERROR_MESSAGES.CUSTOMER_REQUIRED))
      return
    }
    if (models.length === 0) {
      setFormError(t(ERROR_MESSAGES.MODELS_REQUIRED))
      return
    }
    if (models.length > DISCOUNT_CUSTOMER_AUDIT_LIMITS.MODELS_MAX_COUNT) {
      setFormError(t(ERROR_MESSAGES.MODELS_TOO_MANY))
      return
    }

    setFormError(null)
    setIsAuditing(true)
    try {
      const response = await auditCustomerPricing({
        userId: customerId,
        models,
      })
      if (response.success && response.data) {
        setResult(response.data)
      } else {
        setResult(null)
        setFormError(response.message || t(ERROR_MESSAGES.AUDIT_FAILED))
      }
    } catch (error) {
      setResult(null)
      handleServerError(error)
    } finally {
      setIsAuditing(false)
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen)
      }}
    >
      <SheetContent
        className={sideDrawerContentClassName('sm:max-w-[1000px]')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Customer Pricing Audit')}</SheetTitle>
          <SheetDescription>
            {t(
              'Pick a customer and paste the model list to see per-model discounts, cheapest route costs, and whether the quote is safe to sign.'
            )}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-customer-audit-form'
          onSubmit={handleAudit}
          className={sideDrawerFormClassName()}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Customer')} />
            <div className='flex items-end gap-2'>
              <div className='flex-1 space-y-2'>
                <Label htmlFor='discount-audit-customer-keyword'>
                  {t('Search customers by username')}
                </Label>
                <Input
                  id='discount-audit-customer-keyword'
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
            <SideDrawerSectionHeader title={t('Model List')} />
            <Textarea
              value={modelListText}
              rows={8}
              maxLength={DISCOUNT_CUSTOMER_AUDIT_LIMITS.MODEL_LIST_MAX_LENGTH}
              placeholder={t(
                'Paste the model list, one model per line. Commas are also accepted.'
              )}
              onChange={(event) => setModelListText(event.target.value)}
            />
            <p className='text-muted-foreground text-xs'>
              {t('A single audit covers at most 100 models.')}
            </p>
          </SideDrawerSection>

          {formError && (
            <Alert variant='destructive'>
              <AlertTitle>{t('Audit failed')}</AlertTitle>
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          )}

          {result && (
            <>
              <Alert
                variant={
                  result.summary.signable ? 'default' : 'destructive'
                }
              >
                <AlertTitle>{t('Conclusion')}</AlertTitle>
                <AlertDescription>
                  {result.summary.conclusion}
                </AlertDescription>
              </Alert>

              {result.warnings.length > 0 && (
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

              <SideDrawerSection>
                <SideDrawerSectionHeader
                  title={t('Per-model results')}
                  description={t(
                    'Gross margin = customer discount ÷ cheapest cost ratio − 1.'
                  )}
                />
                {result.models.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No models to audit.')}
                </p>
              ) : (
                <div className='overflow-x-auto'>
                  <Table>
                    <TableHeader>
                      <TableRow>
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
                        return (
                          <TableRow key={row.model}>
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
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              )}
              </SideDrawerSection>
            </>
          )}
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          <Button
            form='discount-customer-audit-form'
            type='submit'
            disabled={isAuditing}
          >
            {isAuditing ? t('Auditing...') : t('Run Audit')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
