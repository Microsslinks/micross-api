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
import { Edit, Plus, Trash2 } from 'lucide-react'
import { type FormEvent, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
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
  createDiscountModelList,
  deleteDiscountModelList,
  updateDiscountModelList,
} from '../api'
import { DISCOUNT_MODEL_LIST_LIMITS, ERROR_MESSAGES } from '../constants'
import {
  useDiscountModelLists,
  useRefreshDiscountModelLists,
} from '../hooks/use-discount-model-lists'
import type { DiscountModelList } from '../types'
import { ModelListField } from './model-list-field'

type DiscountsModelListsDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

/** 表格里预览几个模型名；再多了就只说个数，免得把表格撑歪。 */
const PREVIEW_MODEL_COUNT = 3

/**
 * 模型清单抽屉：管「起好名字、能反复使用的模型名单」（比如「企业VIP标准包」）。
 *
 * 这东西的用处只有一个：填模型清单格子时别再手打、重贴一串模型名。
 * 它不参与计价——清单里有什么、有没有被谁引用，都不影响某个客户按几折算钱，
 * 所以删一份清单不需要引用保护（删完只影响以后还能不能一键带入）。
 *
 * 左边这份表单是「新建 / 编辑」共用的：点某一行的编辑，表单切到编辑那一份；
 * 点「新建清单」回到空白。存完自动回到新建态，列表里能看到刚存下的那份。
 */
export function DiscountsModelListsDrawer({
  open,
  onOpenChange,
}: DiscountsModelListsDrawerProps) {
  const { t } = useTranslation()
  const { lists, isLoading } = useDiscountModelLists()
  const refreshLists = useRefreshDiscountModelLists()
  const [editingId, setEditingId] = useState<number | null>(null)
  const [name, setName] = useState('')
  const [remark, setRemark] = useState('')
  const [models, setModels] = useState<string[]>([])
  const [isSaving, setIsSaving] = useState(false)
  const [formError, setFormError] = useState<string | null>(null)
  const [pendingDelete, setPendingDelete] = useState<DiscountModelList | null>(
    null
  )
  const [isDeleting, setIsDeleting] = useState(false)

  const startCreate = () => {
    setEditingId(null)
    setName('')
    setRemark('')
    setModels([])
    setFormError(null)
  }

  useEffect(() => {
    if (!open) {
      // 关掉就把没存下的草稿清干净，下次打开不会挂着上一次的输入。
      startCreate()
      setPendingDelete(null)
    }
  }, [open])

  const startEdit = (list: DiscountModelList) => {
    setEditingId(list.id)
    setName(list.name)
    setRemark(list.remark ?? '')
    setModels(list.models ?? [])
    setFormError(null)
  }

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    const trimmedName = name.trim()
    if (trimmedName === '') {
      setFormError(t(ERROR_MESSAGES.MODEL_LIST_NAME_REQUIRED))
      return
    }
    if (trimmedName.length > DISCOUNT_MODEL_LIST_LIMITS.NAME_MAX_LENGTH) {
      setFormError(t('The list name is too long'))
      return
    }
    if (remark.trim().length > DISCOUNT_MODEL_LIST_LIMITS.REMARK_MAX_LENGTH) {
      setFormError(t('The remark is too long'))
      return
    }
    if (models.length === 0) {
      setFormError(t(ERROR_MESSAGES.MODELS_REQUIRED))
      return
    }

    setFormError(null)
    setIsSaving(true)
    try {
      const payload = {
        name: trimmedName,
        remark: remark.trim(),
        models,
      }
      const response = editingId
        ? await updateDiscountModelList(editingId, payload)
        : await createDiscountModelList(payload)
      if (response.success) {
        toast.success(
          editingId ? t('Model list updated') : t('Model list created')
        )
        startCreate()
        await refreshLists()
      } else {
        // 重名之类的原因后端会说清（会返回 400 与中文原因），原样把话说给运营。
        setFormError(
          response.message || t(ERROR_MESSAGES.MODEL_LIST_SAVE_FAILED)
        )
      }
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!pendingDelete) return

    setIsDeleting(true)
    try {
      const response = await deleteDiscountModelList(pendingDelete.id)
      if (response.success) {
        toast.success(t('Model list deleted'))
        // 正在编辑的就是被删掉的那份：表单回到新建态，别让人对着幽灵继续改。
        if (editingId === pendingDelete.id) {
          startCreate()
        }
        setPendingDelete(null)
        await refreshLists()
        return
      }
      toast.error(
        response.message || t(ERROR_MESSAGES.MODEL_LIST_DELETE_FAILED)
      )
    } catch (error) {
      handleServerError(error)
    } finally {
      setIsDeleting(false)
    }
  }

  const editingList = lists.find((list) => list.id === editingId) ?? null

  return (
    <>
      <Sheet
        open={open}
        onOpenChange={(nextOpen) => {
          onOpenChange(nextOpen)
        }}
      >
        <SheetContent
          className={sideDrawerContentClassName('sm:max-w-[900px]')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Model Lists')}</SheetTitle>
            <SheetDescription>
              {t(
                'Named, reusable model lists. They only save you retyping model names and never take part in pricing.'
              )}
            </SheetDescription>
          </SheetHeader>

          <form
            id='discount-model-list-form'
            onSubmit={handleSave}
            className={sideDrawerFormClassName()}
          >
            <SideDrawerSection>
              <SideDrawerSectionHeader
                title={editingId ? t('Edit the list') : t('New list')}
                description={
                  editingList
                    ? t('Editing: {{name}}', { name: editingList.name })
                    : t(
                        'Give the list a name, then fill in the models. You can import it later with one click.'
                      )
                }
              />

              <div className='space-y-2'>
                <Label htmlFor='discount-model-list-name'>
                  {t('List name')}
                </Label>
                <Input
                  id='discount-model-list-name'
                  value={name}
                  placeholder={t('e.g. Enterprise VIP standard package')}
                  onChange={(event) => setName(event.target.value)}
                />
              </div>

              <div className='space-y-2'>
                <Label htmlFor='discount-model-list-remark'>
                  {t('Remark')}
                </Label>
                <Input
                  id='discount-model-list-remark'
                  value={remark}
                  placeholder={t('Who this list is for, e.g. signed 2026-03')}
                  onChange={(event) => setRemark(event.target.value)}
                />
              </div>

              <div className='space-y-2'>
                <Label>{t('Model List')}</Label>
                <ModelListField
                  value={models}
                  onChange={setModels}
                  max={DISCOUNT_MODEL_LIST_LIMITS.MODELS_MAX_COUNT}
                  allowImport={false}
                />
              </div>

              {formError && (
                <Alert variant='destructive'>
                  <AlertTitle>
                    {t(ERROR_MESSAGES.MODEL_LIST_SAVE_FAILED)}
                  </AlertTitle>
                  <AlertDescription>{formError}</AlertDescription>
                </Alert>
              )}

              <div className='flex items-center gap-2'>
                {editingId !== null && (
                  <Button type='button' variant='outline' onClick={startCreate}>
                    <Plus className='h-4 w-4' />
                    {t('New list')}
                  </Button>
                )}
              </div>
            </SideDrawerSection>

            <SideDrawerSection>
              <SideDrawerSectionHeader
                title={t('Saved lists')}
                description={t(
                  'These lists feed the "import from a saved list" button in the customer audit and price check drawers.'
                )}
              />

              {isLoading ? (
                <p className='text-muted-foreground text-sm'>
                  {t('Loading...')}
                </p>
              ) : lists.length === 0 ? (
                <p className='text-muted-foreground text-sm'>
                  {t('No saved lists yet.')}
                </p>
              ) : (
                <div className='overflow-x-auto'>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('List name')}</TableHead>
                        <TableHead>{t('Models')}</TableHead>
                        <TableHead>{t('Remark')}</TableHead>
                        <TableHead className='text-right'>
                          {t('Actions')}
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {lists.map((list) => {
                        const names = list.models ?? []
                        const preview = names.slice(0, PREVIEW_MODEL_COUNT)
                        const rest = names.length - preview.length
                        return (
                          <TableRow key={list.id}>
                            <TableCell className='font-medium'>
                              {list.name}
                            </TableCell>
                            <TableCell>
                              <span>
                                {t('{{count}} models', { count: names.length })}
                              </span>
                              {names.length > 0 && (
                                <p className='text-muted-foreground text-xs'>
                                  {preview.join(', ')}
                                  {rest > 0 ? ` +${rest}` : ''}
                                </p>
                              )}
                            </TableCell>
                            <TableCell className='text-muted-foreground max-w-[220px] truncate'>
                              {list.remark || '—'}
                            </TableCell>
                            <TableCell className='text-right whitespace-nowrap'>
                              <Button
                                type='button'
                                variant='ghost'
                                size='icon-sm'
                                aria-label={t('Edit')}
                                onClick={() => startEdit(list)}
                              >
                                <Edit />
                              </Button>
                              <Button
                                type='button'
                                variant='ghost'
                                size='icon-sm'
                                aria-label={t('Delete')}
                                className='text-destructive hover:text-destructive'
                                onClick={() => setPendingDelete(list)}
                              >
                                <Trash2 />
                              </Button>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              )}
            </SideDrawerSection>
          </form>

          <SheetFooter className={sideDrawerFooterClassName()}>
            <SheetClose render={<Button variant='outline' />}>
              {t('Close')}
            </SheetClose>
            <Button
              form='discount-model-list-form'
              type='submit'
              disabled={isSaving}
            >
              {isSaving
                ? t('Saving...')
                : editingId
                  ? t('Save')
                  : t('Create the list')}
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <ConfirmDialog
        open={pendingDelete !== null}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) setPendingDelete(null)
        }}
        title={t('Are you sure?')}
        desc={t(
          'This deletes the model list {{name}}. Models already filled in elsewhere are untouched; only later imports lose this list.',
          { name: pendingDelete?.name ?? '' }
        )}
        destructive
        isLoading={isDeleting}
        confirmText={isDeleting ? t('Deleting...') : t('Delete')}
        handleConfirm={handleDelete}
      />
    </>
  )
}
