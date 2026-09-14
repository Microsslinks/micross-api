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
import { Link2, Link2Off, Search } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
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
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
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

import {
  createDiscountBinding,
  deleteDiscountBinding,
  getDiscountBindings,
  searchCustomers,
} from '../api'
import {
  DISCOUNT_PLAN_LIMITS,
  DISCOUNT_SIMULATE_LIMITS,
  ERROR_MESSAGES,
  getDiscountBindingSourceLabel,
  getDiscountSubjectLabel,
} from '../constants'
import {
  DISCOUNT_BINDING_SOURCE,
  DISCOUNT_SUBJECT,
  type DiscountBinding,
  type DiscountCustomer,
} from '../types'
import { useDiscounts } from './discounts-provider'

type DiscountsBindingsDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DiscountsBindingsDrawer({
  open,
  onOpenChange,
}: DiscountsBindingsDrawerProps) {
  const { t } = useTranslation()
  const { currentRow, triggerRefresh } = useDiscounts()
  const [bindings, setBindings] = useState<DiscountBinding[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [keyword, setKeyword] = useState('')
  const [customers, setCustomers] = useState<DiscountCustomer[]>([])
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(false)
  /** 绑定接口只收回客户 ID，这里把搜到的用户名记下来，列表里才好认人。 */
  const [knownNames, setKnownNames] = useState<Record<number, string>>({})
  const [selectedCustomerId, setSelectedCustomerId] = useState('')
  const [isBinding, setIsBinding] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [unbindTarget, setUnbindTarget] = useState<DiscountBinding | null>(null)
  const [isUnbinding, setIsUnbinding] = useState(false)

  const planId = currentRow?.id ?? 0

  const loadBindings = useCallback(async () => {
    if (!planId) return
    setIsLoading(true)
    try {
      const result = await getDiscountBindings({
        plan_id: planId,
        page: 1,
        page_size: DISCOUNT_PLAN_LIMITS.BINDING_PAGE_SIZE,
      })
      if (result.success) {
        setBindings(result.data?.items ?? [])
        setLoadError(null)
      } else {
        setBindings([])
        setLoadError(result.message || t(ERROR_MESSAGES.BINDINGS_LOAD_FAILED))
      }
    } catch (error) {
      setBindings([])
      handleServerError(error)
    } finally {
      setIsLoading(false)
    }
  }, [planId, t])

  const loadCustomers = useCallback(
    async (searchKeyword: string) => {
      setIsLoadingCustomers(true)
      try {
        const response = await searchCustomers({
          keyword: searchKeyword,
          page_size: DISCOUNT_SIMULATE_LIMITS.CUSTOMER_PAGE_SIZE,
        })
        if (response.success) {
          const items = response.data?.items ?? []
          setCustomers(items)
          setKnownNames((previous) => {
            const next = { ...previous }
            for (const customer of items) {
              next[customer.id] = customer.username
            }
            return next
          })
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
      setBindings([])
      setKeyword('')
      setCustomers([])
      setSelectedCustomerId('')
      setFormError(null)
      setLoadError(null)
      setUnbindTarget(null)
      return
    }
    void loadBindings()
    void loadCustomers('')
  }, [open, loadBindings, loadCustomers])

  const handleBind = async () => {
    const customerId = Number.parseInt(selectedCustomerId, 10)
    if (!Number.isFinite(customerId) || customerId <= 0) {
      setFormError(t(ERROR_MESSAGES.CUSTOMER_REQUIRED))
      return
    }

    setFormError(null)
    setIsBinding(true)
    try {
      const result = await createDiscountBinding({
        subject_type: DISCOUNT_SUBJECT.USER,
        subject_id: customerId,
        plan_id: planId,
        effective_from: 0,
        effective_to: 0,
        source: DISCOUNT_BINDING_SOURCE.MANUAL,
      })
      if (result.success) {
        toast.success(t('Customer bound to this plan'))
        setSelectedCustomerId('')
        await loadBindings()
        triggerRefresh()
        return
      }
      toast.error(result.message || t(ERROR_MESSAGES.BINDING_SAVE_FAILED))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsBinding(false)
    }
  }

  const handleUnbind = async () => {
    if (!unbindTarget) return

    setIsUnbinding(true)
    try {
      const result = await deleteDiscountBinding(unbindTarget.id)
      if (result.success) {
        toast.success(t('Customer unbound from this plan'))
        setUnbindTarget(null)
        await loadBindings()
        triggerRefresh()
        return
      }
      toast.error(result.message || t(ERROR_MESSAGES.BINDING_DELETE_FAILED))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsUnbinding(false)
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onOpenChange(false)
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[640px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Bind customers')}</SheetTitle>
          <SheetDescription>
            {currentRow?.name
              ? t('Customers bound to {{name}}', { name: currentRow.name })
              : t('Customers bound to this plan')}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-binding-form'
          className={sideDrawerFormClassName()}
          onSubmit={(event) => {
            event.preventDefault()
            void handleBind()
          }}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader
              title={t('Bind a customer')}
              description={t(
                'Binding adds this plan alongside the ones the customer already has; the pricing rules decide which plan applies to each model.'
              )}
            />

            <div className='flex items-end gap-2'>
              <div className='flex-1 space-y-2'>
                <Label htmlFor='discount-binding-keyword'>
                  {t('Search customers by username')}
                </Label>
                <Input
                  id='discount-binding-keyword'
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

            <div className='flex items-end gap-2'>
              <div className='flex-1 space-y-2'>
                <Label>{t('Customer')}</Label>
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
                        <SelectItem
                          key={customer.id}
                          value={String(customer.id)}
                        >
                          {customer.display_name
                            ? `${customer.username} (${customer.display_name})`
                            : customer.username}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
              <Button type='submit' disabled={isBinding}>
                <Link2 className='h-4 w-4' />
                {isBinding ? t('Binding...') : t('Bind')}
              </Button>
            </div>

            {!isLoadingCustomers && customers.length === 0 && (
              <p className='text-muted-foreground text-xs'>
                {t('No customers found')}
              </p>
            )}

            {formError && (
              <Alert variant='destructive'>
                <AlertTitle>{t('Cannot bind')}</AlertTitle>
                <AlertDescription>{formError}</AlertDescription>
              </Alert>
            )}
          </SideDrawerSection>

          <SideDrawerSection>
            <SideDrawerSectionHeader title={t('Bound customers')} />

            {loadError && (
              <Alert variant='destructive'>
                <AlertTitle>{t('Failed to load the bindings')}</AlertTitle>
                <AlertDescription>{loadError}</AlertDescription>
              </Alert>
            )}

            {!loadError && bindings.length === 0 && (
              <p className='text-muted-foreground text-sm'>
                {isLoading
                  ? t('Loading...')
                  : t('No customer is bound to this plan yet.')}
              </p>
            )}

            {bindings.length > 0 && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Customer')}</TableHead>
                    <TableHead>{t('Source')}</TableHead>
                    <TableHead>{t('Effective from')}</TableHead>
                    <TableHead>{t('Effective to')}</TableHead>
                    <TableHead>{t('Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {bindings.map((binding) => {
                    const name = knownNames[binding.subject_id]
                    return (
                      <TableRow key={binding.id}>
                        <TableCell>
                          <div className='flex flex-col'>
                            <span className='font-medium'>
                              {name ||
                                t('{{type}} #{{id}}', {
                                  type: getDiscountSubjectLabel(
                                    t,
                                    binding.subject_type
                                  ),
                                  id: binding.subject_id,
                                })}
                            </span>
                            {name && (
                              <span className='text-muted-foreground font-mono text-xs'>
                                #{binding.subject_id}
                              </span>
                            )}
                          </div>
                        </TableCell>
                        <TableCell>
                          {getDiscountBindingSourceLabel(t, binding.source)}
                        </TableCell>
                        <TableCell className='text-xs'>
                          {binding.effective_from > 0
                            ? formatTimestampToDate(binding.effective_from)
                            : t('Unlimited')}
                        </TableCell>
                        <TableCell className='text-xs'>
                          {binding.effective_to > 0
                            ? formatTimestampToDate(binding.effective_to)
                            : t('Unlimited')}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant='ghost'
                            size='icon-sm'
                            aria-label={t('Unbind')}
                            onClick={() => setUnbindTarget(binding)}
                          >
                            <Link2Off />
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            )}
          </SideDrawerSection>
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
        </SheetFooter>
      </SheetContent>

      <AlertDialog
        open={unbindTarget !== null}
        onOpenChange={(nextOpen) => !nextOpen && setUnbindTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Are you sure?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                '{{name}} falls back to the official price as soon as it is unbound.',
                {
                  name: unbindTarget
                    ? knownNames[unbindTarget.subject_id] ||
                      `#${unbindTarget.subject_id}`
                    : '',
                }
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isUnbinding}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleUnbind}
              disabled={isUnbinding}
              variant='destructive'
            >
              {isUnbinding ? t('Unbinding...') : t('Unbind')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Sheet>
  )
}
