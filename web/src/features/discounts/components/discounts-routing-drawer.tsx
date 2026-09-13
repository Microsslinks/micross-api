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
  getDiscountRouting,
  resetDiscountRouting,
  saveDiscountRouting,
  searchCustomers,
} from '../api'
import {
  DISCOUNT_ROUTING_LIMITS,
  DISCOUNT_SIMULATE_LIMITS,
  ERROR_MESSAGES,
  getDiscountRoutingStrategyOptions,
} from '../constants'
import {
  DISCOUNT_ROUTING_STRATEGY,
  type DiscountCustomer,
  type DiscountRouting,
} from '../types'

type DiscountsRoutingDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/** 「是否允许走亏损线路」的两个取值，跟 ToggleGroup 的字符串取值对上。 */
const COST_BREACH_VALUES = ['0', '1'] as const

/**
 * 路由策略抽屉：选一个客户，配他走线路的口径、以及一条保本线路都没有时是否放行。
 *
 * 两件事都只对「真的在打折」的客户有意义（全价客户本来就不会亏），所以放在折扣模块下。
 * 没单独配过的客户显示为默认口径，而不是一片空白——「没配过」是绝大多数客户的常态。
 */
export function DiscountsRoutingDrawer({
  open,
  onOpenChange,
}: DiscountsRoutingDrawerProps) {
  const { t } = useTranslation()
  const [keyword, setKeyword] = useState('')
  const [customers, setCustomers] = useState<DiscountCustomer[]>([])
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(false)
  const [selectedCustomerId, setSelectedCustomerId] = useState('')
  const [routing, setRouting] = useState<DiscountRouting | null>(null)
  const [isLoadingRouting, setIsLoadingRouting] = useState(false)
  const [strategy, setStrategy] = useState<string>(DISCOUNT_ROUTING_STRATEGY.MARGIN)
  const [allowCostBreach, setAllowCostBreach] = useState<string>(
    COST_BREACH_VALUES[0]
  )
  const [remark, setRemark] = useState('')
  const [isSaving, setIsSaving] = useState(false)
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

  const loadRouting = useCallback(
    async (userId: number) => {
      setIsLoadingRouting(true)
      setFormError(null)
      try {
        const response = await getDiscountRouting(userId)
        if (response.success && response.data) {
          setRouting(response.data)
          setStrategy(response.data.routing_strategy)
          setAllowCostBreach(response.data.allow_cost_breach ? '1' : '0')
          setRemark(response.data.remark ?? '')
        } else {
          setRouting(null)
          setFormError(response.message || t(ERROR_MESSAGES.ROUTING_LOAD_FAILED))
        }
      } catch (error) {
        setRouting(null)
        handleServerError(error)
      } finally {
        setIsLoadingRouting(false)
      }
    },
    [t]
  )

  useEffect(() => {
    if (!open) {
      // 关掉就把上一次的输入清干净，下次打开不会挂着旧客户的数据。
      setKeyword('')
      setCustomers([])
      setSelectedCustomerId('')
      setRouting(null)
      setStrategy(DISCOUNT_ROUTING_STRATEGY.MARGIN)
      setAllowCostBreach(COST_BREACH_VALUES[0])
      setRemark('')
      setFormError(null)
      return
    }
    void loadCustomers('')
  }, [open, loadCustomers])

  useEffect(() => {
    const userId = Number.parseInt(selectedCustomerId, 10)
    if (!Number.isFinite(userId) || userId <= 0) {
      setRouting(null)
      return
    }
    void loadRouting(userId)
  }, [selectedCustomerId, loadRouting])

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const userId = Number.parseInt(selectedCustomerId, 10)
    if (!Number.isFinite(userId) || userId <= 0) {
      setFormError(t(ERROR_MESSAGES.CUSTOMER_REQUIRED))
      return
    }
    const trimmedRemark = remark.trim()
    if (trimmedRemark.length > DISCOUNT_ROUTING_LIMITS.REMARK_MAX_LENGTH) {
      setFormError(t('The remark is too long'))
      return
    }

    setFormError(null)
    setIsSaving(true)
    try {
      const response = await saveDiscountRouting({
        user_id: userId,
        routing_strategy: strategy,
        allow_cost_breach: allowCostBreach === '1',
        remark: trimmedRemark,
      })
      if (response.success) {
        // 回读一次：界面上显示的是后端真正生效的口径，而不是刚才提交的意图。
        await loadRouting(userId)
      } else {
        setFormError(response.message || t(ERROR_MESSAGES.ROUTING_SAVE_FAILED))
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSaving(false)
    }
  }

  const handleReset = async () => {
    const userId = Number.parseInt(selectedCustomerId, 10)
    if (!Number.isFinite(userId) || userId <= 0) {
      setFormError(t(ERROR_MESSAGES.CUSTOMER_REQUIRED))
      return
    }

    setFormError(null)
    setIsSaving(true)
    try {
      const response = await resetDiscountRouting(userId)
      if (response.success) {
        await loadRouting(userId)
      } else {
        setFormError(response.message || t(ERROR_MESSAGES.ROUTING_RESET_FAILED))
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSaving(false)
    }
  }

  const hasCustomer = Number.parseInt(selectedCustomerId, 10) > 0
  const isBusy = isSaving || isLoadingRouting

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen)
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[640px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Routing Policy')}</SheetTitle>
          <SheetDescription>
            {t(
              'Decide how a customer picks upstream routes, and whether the platform may serve them at a loss.'
            )}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-routing-form'
          onSubmit={handleSave}
          className={sideDrawerFormClassName()}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Customer')} />
            <div className='flex items-end gap-2'>
              <div className='flex-1 space-y-2'>
                <Label htmlFor='discount-routing-keyword'>
                  {t('Search customers by username')}
                </Label>
                <Input
                  id='discount-routing-keyword'
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
            {routing && (
              <p className='text-muted-foreground text-xs'>
                {routing.configured
                  ? t(
                      'This customer has its own routing policy; other customers use the default.'
                    )
                  : t(
                      'This customer uses the default routing policy and has no row of its own.'
                    )}
                {routing.configured && routing.updated_at > 0 && (
                  <>
                    {' '}
                    {t('Last updated at')}{' '}
                    {new Date(routing.updated_at * 1000).toLocaleString()}
                  </>
                )}
              </p>
            )}
          </SideDrawerSection>

          {hasCustomer && (
            <>
              <SideDrawerSection>
                <SideDrawerSectionHeader
                  title={t('Routing Strategy')}
                  description={t('How this customer picks among break-even routes.')}
                />
                <ToggleGroup
                  value={[strategy]}
                  onValueChange={(value) => {
                    // 单选：点已选中的那项等于取消，忽略掉。
                    const next = value.find((item) => item !== strategy)
                    if (next) setStrategy(next)
                  }}
                  variant='outline'
                  size='lg'
                  spacing={2}
                  className='grid w-full grid-cols-2 gap-2'
                >
                  {getDiscountRoutingStrategyOptions(t).map((option) => (
                    <ToggleGroupItem
                      key={option.value}
                      value={option.value}
                      className='h-auto min-h-10 w-full'
                    >
                      {option.label}
                    </ToggleGroupItem>
                  ))}
                </ToggleGroup>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Highest margin first: every extra point of margin stays with the platform. Stable route first: the upstream priority decides, so the steadiest route wins even if it earns less.'
                  )}
                </p>
              </SideDrawerSection>

              <SideDrawerSection>
                <SideDrawerSectionHeader
                  title={t('Allow cost breach')}
                  description={t(
                    'What happens when no route can serve this customer without a loss.'
                  )}
                />
                <ToggleGroup
                  value={[allowCostBreach]}
                  onValueChange={(value) => {
                    // 单选：点已选中的那项等于取消，忽略掉。
                    const next = value.find((item) => item !== allowCostBreach)
                    if (next) setAllowCostBreach(next)
                  }}
                  variant='outline'
                  size='lg'
                  spacing={2}
                  className='grid w-full grid-cols-2 gap-2'
                >
                  <ToggleGroupItem
                    value={COST_BREACH_VALUES[0]}
                    className='h-auto min-h-10 w-full'
                  >
                    {t('Not allowed')}
                  </ToggleGroupItem>
                  <ToggleGroupItem
                    value={COST_BREACH_VALUES[1]}
                    className='h-auto min-h-10 w-full'
                  >
                    {t('Allowed')}
                  </ToggleGroupItem>
                </ToggleGroup>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Not allowed: the request fails and the error names the routes that would work and how much each one loses. Allowed: the request is served on the cheapest losing route and every such request is flagged in the logs.'
                  )}
                </p>
              </SideDrawerSection>

              <SideDrawerSection>
                <SideDrawerSectionHeader title={t('Remark')} />
                <Input
                  value={remark}
                  maxLength={DISCOUNT_ROUTING_LIMITS.REMARK_MAX_LENGTH}
                  placeholder={t('Why this customer is special, e.g. the ticket number')}
                  onChange={(event) => setRemark(event.target.value)}
                />
              </SideDrawerSection>
            </>
          )}

          {formError && (
            <Alert variant='destructive'>
              <AlertTitle>{t('Setting the routing policy failed')}</AlertTitle>
              <AlertDescription>{formError}</AlertDescription>
            </Alert>
          )}
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          <Button
            type='button'
            variant='outline'
            disabled={!hasCustomer || isBusy || !routing?.configured}
            onClick={() => void handleReset()}
          >
            {t('Reset to default')}
          </Button>
          <Button form='discount-routing-form' type='submit' disabled={!hasCustomer || isBusy}>
            {isSaving ? t('Saving...') : t('Save')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
