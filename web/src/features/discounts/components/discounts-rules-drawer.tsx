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
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState, type FormEvent } from 'react'
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
import { StatusBadge } from '@/components/status-badge'
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

import {
  createDiscountRule,
  deleteDiscountRule,
  getDiscountRules,
  updateDiscountRule,
} from '../api'
import {
  ERROR_MESSAGES,
  getDiscountRuleScopeLabel,
  getDiscountScopeOptions,
  getDiscountStatusOptions,
} from '../constants'
import { useModelNameOptions } from '../hooks/use-model-name-options'
import {
  DISCOUNT_RULE_FORM_DEFAULT_VALUES,
  buildDiscountRulePayload,
  getDiscountRuleFormSchema,
  transformDiscountRuleToFormDefaults,
  type DiscountRuleFormValues,
} from '../lib'
import {
  DISCOUNT_PLAN_STATUS,
  DISCOUNT_SCOPE,
  type DiscountRule,
} from '../types'
import { useDiscounts } from './discounts-provider'

type DiscountsRulesDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DiscountsRulesDrawer({
  open,
  onOpenChange,
}: DiscountsRulesDrawerProps) {
  const { t } = useTranslation()
  const { currentRow } = useDiscounts()
  const { options: modelOptions } = useModelNameOptions()
  const [rules, setRules] = useState<DiscountRule[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  /** 表单收起来时是 null；新建时是 'create'；改某条时是那条规则。 */
  const [editingRule, setEditingRule] = useState<
    DiscountRule | 'create' | null
  >(null)
  const [values, setValues] = useState<DiscountRuleFormValues>(
    DISCOUNT_RULE_FORM_DEFAULT_VALUES
  )
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [isSaving, setIsSaving] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<DiscountRule | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)

  const planId = currentRow?.id ?? 0

  const loadRules = useCallback(async () => {
    if (!planId) return
    setIsLoading(true)
    try {
      const result = await getDiscountRules(planId)
      if (result.success) {
        setRules(result.data ?? [])
        setLoadError(null)
      } else {
        setRules([])
        setLoadError(result.message || t(ERROR_MESSAGES.RULES_LOAD_FAILED))
      }
    } catch (error) {
      setRules([])
      handleServerError(error)
    } finally {
      setIsLoading(false)
    }
  }, [planId, t])

  useEffect(() => {
    if (!open) {
      setRules([])
      setEditingRule(null)
      setValues(DISCOUNT_RULE_FORM_DEFAULT_VALUES)
      setErrors({})
      setLoadError(null)
      setDeleteTarget(null)
      return
    }
    void loadRules()
  }, [open, loadRules])

  const setField = <K extends keyof DiscountRuleFormValues>(
    field: K,
    value: DiscountRuleFormValues[K]
  ) => {
    setValues((previous) => ({ ...previous, [field]: value }))
    setErrors((previous) => {
      if (!previous[field]) return previous
      const next = { ...previous }
      delete next[field]
      return next
    })
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const parsed = getDiscountRuleFormSchema(t).safeParse(values)
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
    if (editingRule === null) return

    const payload = buildDiscountRulePayload(values)
    setIsSaving(true)
    try {
      const result =
        editingRule === 'create'
          ? await createDiscountRule(planId, payload)
          : await updateDiscountRule(editingRule.id, payload)

      if (result.success) {
        toast.success(
          editingRule === 'create' ? t('Rule added') : t('Rule saved')
        )
        setEditingRule(null)
        setValues(DISCOUNT_RULE_FORM_DEFAULT_VALUES)
        await loadRules()
        return
      }
      toast.error(result.message || t(ERROR_MESSAGES.RULE_SAVE_FAILED))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return

    setIsDeleting(true)
    try {
      const result = await deleteDiscountRule(deleteTarget.id)
      if (result.success) {
        toast.success(t('Rule deleted'))
        setDeleteTarget(null)
        await loadRules()
        return
      }
      toast.error(result.message || t(ERROR_MESSAGES.RULE_DELETE_FAILED))
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsDeleting(false)
    }
  }

  const scopeOptions = getDiscountScopeOptions(t)
  const statusOptions = getDiscountStatusOptions(t)

  let submitLabel = t('Save')
  if (editingRule === 'create') submitLabel = t('Add rule')
  if (isSaving) submitLabel = t('Saving...')

  return (
    <Sheet
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) onOpenChange(false)
      }}
    >
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[640px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Rules')}</SheetTitle>
          <SheetDescription>
            {currentRow?.name
              ? t('Rules of {{name}}', { name: currentRow.name })
              : t('Rules of this plan')}
          </SheetDescription>
        </SheetHeader>

        <form
          id='discount-rule-form'
          onSubmit={handleSubmit}
          className={sideDrawerFormClassName()}
        >
          <SideDrawerSection>
            <SideDrawerSectionHeader
              title={t('Rules')}
              description={t(
                'Rules set which models receive the plan base discount. The first matching rule wins; the highest priority is checked first.'
              )}
            />

            {loadError && (
              <Alert variant='destructive'>
                <AlertTitle>{t('Failed to load the rules')}</AlertTitle>
                <AlertDescription>{loadError}</AlertDescription>
              </Alert>
            )}

            {!loadError && rules.length === 0 && (
              <p className='text-muted-foreground text-sm'>
                {isLoading
                  ? t('Loading...')
                  : t(
                      'No rules yet. The base discount is not applied to any model.'
                    )}
              </p>
            )}

            {rules.length > 0 && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Scope')}</TableHead>
                    <TableHead>{t('Priority')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rules.map((rule) => (
                    <TableRow key={rule.id}>
                      <TableCell>
                        {getDiscountRuleScopeLabel(t, rule.scope_type)}
                        {' · '}
                        <span className='font-mono text-xs'>
                          {rule.scope_value}
                        </span>
                      </TableCell>
                      <TableCell>{rule.priority}</TableCell>
                      <TableCell>
                        <StatusBadge
                          label={
                            rule.status === DISCOUNT_PLAN_STATUS.ENABLED
                              ? t('Enabled')
                              : t('Disabled')
                          }
                          variant={
                            rule.status === DISCOUNT_PLAN_STATUS.ENABLED
                              ? 'success'
                              : 'neutral'
                          }
                          copyable={false}
                        />
                      </TableCell>
                      <TableCell>
                        <div className='flex items-center gap-1'>
                          <Button
                            variant='ghost'
                            size='icon-sm'
                            aria-label={t('Edit')}
                            onClick={() => {
                              setEditingRule(rule)
                              setValues(
                                transformDiscountRuleToFormDefaults(rule)
                              )
                              setErrors({})
                            }}
                          >
                            <Pencil />
                          </Button>
                          <Button
                            variant='ghost'
                            size='icon-sm'
                            aria-label={t('Delete')}
                            onClick={() => setDeleteTarget(rule)}
                          >
                            <Trash2 />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}

            {editingRule === null && (
              <Button
                type='button'
                variant='outline'
                onClick={() => {
                  setEditingRule('create')
                  setValues(DISCOUNT_RULE_FORM_DEFAULT_VALUES)
                  setErrors({})
                }}
              >
                <Plus className='h-4 w-4' />
                {t('Add rule')}
              </Button>
            )}
          </SideDrawerSection>

          {editingRule !== null && (
            <SideDrawerSection>
              <SideDrawerSectionHeader
                title={
                  editingRule === 'create' ? t('Add rule') : t('Edit rule')
                }
              />

              <div className='space-y-2'>
                <Label>{t('Scope')}</Label>
                <Select
                  items={scopeOptions}
                  value={values.scope_type}
                  onValueChange={(value) =>
                    setField('scope_type', String(value ?? ''))
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder={t('Scope')} />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {scopeOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <div className='space-y-2'>
                <Label htmlFor='discount-rule-scope-value'>
                  {t('Scope value')}
                </Label>
                {values.scope_type === DISCOUNT_SCOPE.MODEL ? (
                  // 模型名可以从目录里挑：写错的模型名不会报错，只会让这条规则
                  // 永远不命中，所以这里给可选项比让人手打更安全（也可自由输入）。
                  <ComboboxInput
                    id='discount-rule-scope-value'
                    options={modelOptions}
                    value={values.scope_value}
                    allowCustomValue
                    placeholder={t('Search models or type a model name')}
                    emptyText='No matching items'
                    onValueChange={(value) => setField('scope_value', value)}
                  />
                ) : (
                  <Input
                    id='discount-rule-scope-value'
                    value={values.scope_value}
                    placeholder={t(
                      'A model name or a vendor name, e.g. claude-3-5-sonnet or anthropic'
                    )}
                    onChange={(event) =>
                      setField('scope_value', event.target.value)
                    }
                  />
                )}
                {errors.scope_value && (
                  <p className='text-destructive text-xs'>
                    {errors.scope_value}
                  </p>
                )}
              </div>

              <div className='space-y-2'>
                <Label htmlFor='discount-rule-priority'>{t('Priority')}</Label>
                <Input
                  id='discount-rule-priority'
                  type='number'
                  value={String(values.priority)}
                  onChange={(event) =>
                    setField(
                      'priority',
                      Number.parseInt(event.target.value, 10) || 0
                    )
                  }
                />
                {errors.priority && (
                  <p className='text-destructive text-xs'>{errors.priority}</p>
                )}
                <p className='text-muted-foreground text-xs'>
                  {t('Higher number wins when several rules match.')}
                </p>
              </div>

              <div className='space-y-2'>
                <Label>{t('Status')}</Label>
                <Select
                  items={statusOptions}
                  value={String(values.status)}
                  onValueChange={(value) =>
                    setField(
                      'status',
                      Number(value) === DISCOUNT_PLAN_STATUS.DISABLED
                        ? DISCOUNT_PLAN_STATUS.DISABLED
                        : DISCOUNT_PLAN_STATUS.ENABLED
                    )
                  }
                >
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder={t('Status')} />
                  </SelectTrigger>
                  <SelectContent alignItemWithTrigger={false}>
                    <SelectGroup>
                      {statusOptions.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>

              <div className='flex gap-2'>
                <Button
                  type='button'
                  variant='ghost'
                  onClick={() => {
                    setEditingRule(null)
                    setValues(DISCOUNT_RULE_FORM_DEFAULT_VALUES)
                    setErrors({})
                  }}
                >
                  {t('Cancel')}
                </Button>
              </div>
            </SideDrawerSection>
          )}
        </form>

        <SheetFooter className={sideDrawerFooterClassName()}>
          <SheetClose render={<Button variant='outline' />}>
            {t('Close')}
          </SheetClose>
          {editingRule !== null && (
            <Button form='discount-rule-form' type='submit' disabled={isSaving}>
              {submitLabel}
            </Button>
          )}
        </SheetFooter>
      </SheetContent>

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(nextOpen) => !nextOpen && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('Are you sure?')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('This will permanently delete the rule for')}{' '}
              <span className='font-semibold'>{deleteTarget?.scope_value}</span>
              {t('. This action cannot be undone.')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>
              {t('Cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={isDeleting}
              variant='destructive'
            >
              {isDeleting ? t('Deleting...') : t('Delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Sheet>
  )
}
