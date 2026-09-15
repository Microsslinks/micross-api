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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { RichContent } from '@/components/rich-content'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { useUpdateOption } from '@/features/system-settings/hooks/use-update-option'
import { isHttpUrl, isLikelyHtml } from '@/lib/content-format'

type ContentEditorDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  /** Option key to persist, e.g. 'HomePageContent' or 'legal.user_agreement' */
  optionKey: string
  /** Current saved value */
  initialValue: string
  placeholder?: string
}

/**
 * Large-text editor dialog for site content pages (home / about / footer /
 * user agreement / privacy policy). Edit tab shows a monospace textarea;
 * Preview tab renders the content the same way the public page does.
 */
export function ContentEditorDialog({
  open,
  onOpenChange,
  title,
  optionKey,
  initialValue,
  placeholder,
}: ContentEditorDialogProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [value, setValue] = useState(initialValue)

  // Re-sync local state each time the dialog opens
  useEffect(() => {
    if (open) {
      setValue(initialValue)
    }
  }, [open, initialValue])

  const trimmed = value.trim()
  const isUrl = isHttpUrl(trimmed)
  const contentIsHtml = !isUrl && isLikelyHtml(trimmed)

  const handleSave = async () => {
    await updateOption.mutateAsync({ key: optionKey, value })
    onOpenChange(false)
  }

  const renderPreviewBody = () => {
    if (!trimmed) {
      return (
        <p className='text-muted-foreground text-sm'>
          {t('Nothing to preview yet.')}
        </p>
      )
    }
    if (isUrl) {
      return (
        <div className='space-y-2'>
          <p className='text-muted-foreground text-sm'>
            {t(
              'This content is an external URL and will be embedded or linked as configured.'
            )}
          </p>
          <a
            href={trimmed}
            target='_blank'
            rel='noopener noreferrer'
            className='text-primary text-sm underline underline-offset-3'
          >
            {trimmed}
          </a>
        </div>
      )
    }
    return (
      <RichContent
        mode={contentIsHtml ? 'html' : 'markdown'}
        content={trimmed}
      />
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='flex max-h-[85vh] w-full flex-col gap-0 sm:max-w-4xl'>
        <DialogHeader className='px-1 pt-1'>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {t(
              'Supports Markdown, HTML, or a full URL starting with http(s).'
            )}
          </DialogDescription>
        </DialogHeader>

        <Tabs defaultValue='edit' className='flex min-h-0 flex-1 flex-col gap-3 px-1 py-3'>
          <TabsList className='self-start'>
            <TabsTrigger value='edit'>{t('Edit')}</TabsTrigger>
            <TabsTrigger value='preview'>{t('Preview')}</TabsTrigger>
          </TabsList>

          <TabsContent value='edit' className='min-h-0 flex-1'>
            <Textarea
              value={value}
              onChange={(e) => setValue(e.target.value)}
              placeholder={placeholder}
              className='h-full min-h-[50vh] resize-none font-mono text-[13px] leading-relaxed'
            />
          </TabsContent>

          <TabsContent
            value='preview'
            className='bg-muted/30 max-h-[60vh] min-h-[50vh] overflow-auto rounded-lg border p-4'
          >
            {renderPreviewBody()}
          </TabsContent>
        </Tabs>

        <DialogFooter>
          <Button
            variant='outline'
            onClick={() => onOpenChange(false)}
            disabled={updateOption.isPending}
          >
            {t('Cancel')}
          </Button>
          <Button
            onClick={() => void handleSave()}
            disabled={updateOption.isPending}
          >
            {t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
