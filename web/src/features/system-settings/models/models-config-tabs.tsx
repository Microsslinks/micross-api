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
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

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
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'

import { ClaudeSettingsCard, type ClaudeTabHandle } from './claude-settings-card'
import {
  GlobalSettingsCard,
  type GlobalTabHandle,
} from './global-settings-card'
import {
  GeminiSettingsCard,
  type GeminiTabHandle,
} from './gemini-settings-card'
import { GrokSettingsCard, type GrokTabHandle } from './grok-settings-card'

export type ModelTabId = 'global' | 'gemini' | 'claude' | 'grok'

type TabConfig = {
  id: ModelTabId
  labelKey: string
}

const TAB_DEFINITIONS: TabConfig[] = [
  { id: 'global', labelKey: 'Global' },
  { id: 'gemini', labelKey: 'Gemini' },
  { id: 'claude', labelKey: 'Claude' },
  { id: 'grok', labelKey: 'Grok' },
]

export type ModelConfigTabHandle =
  | GlobalTabHandle
  | GeminiTabHandle
  | ClaudeTabHandle
  | GrokTabHandle

export type ModelConfigDefaults = {
  global: Parameters<typeof GlobalSettingsCard>[0]['defaultValues']
  gemini: Parameters<typeof GeminiSettingsCard>[0]['defaultValues']
  claude: Parameters<typeof ClaudeSettingsCard>[0]['defaultValues']
  grok: Parameters<typeof GrokSettingsCard>[0]['defaultValues']
}

type ModelsConfigTabsProps = {
  defaults: ModelConfigDefaults
}

export function ModelsConfigTabs({ defaults }: ModelsConfigTabsProps) {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState<ModelTabId>('global')
  const [isSaving, setIsSaving] = useState(false)
  const [pendingTab, setPendingTab] = useState<ModelTabId | null>(null)

  const handlesRef = useRef<Record<ModelTabId, ModelConfigTabHandle | null>>({
    global: null,
    gemini: null,
    claude: null,
    grok: null,
  })

  const setHandle = (id: ModelTabId) => (handle: ModelConfigTabHandle | null) => {
    handlesRef.current[id] = handle
  }

  const switchTo = (target: ModelTabId) => {
    setActiveTab(target)
    setPendingTab(null)
  }

  const handleTabRequest = (nextId: string) => {
    const target = nextId as ModelTabId
    if (target === activeTab) return
    const current = handlesRef.current[activeTab]
    if (current && current.isDirty()) {
      setPendingTab(target)
      return
    }
    switchTo(target)
  }

  const handleSave = async () => {
    const handle = handlesRef.current[activeTab]
    if (!handle) return
    setIsSaving(true)
    try {
      await handle.submit()
      switchTo(pendingTab ?? activeTab)
    } catch {
      // Save failed: keep the dialog open and stay on the active tab.
    } finally {
      setIsSaving(false)
    }
  }

  const handleDiscard = () => {
    handlesRef.current[activeTab]?.reset()
    switchTo(pendingTab ?? activeTab)
  }

  const handleDialogOpenChange = (open: boolean) => {
    if (!open) setPendingTab(null)
  }

  return (
    <SettingsSection title={t('Model Configuration')} showHeader>
      <Tabs value={activeTab} onValueChange={handleTabRequest}>
        <div className='flex flex-wrap items-center justify-between gap-3 border-b pb-2'>
          <TabsList variant='line'>
            {TAB_DEFINITIONS.map((tab) => (
              <TabsTrigger key={tab.id} value={tab.id}>
                {t(tab.labelKey)}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>
      </Tabs>

      <div className='pt-4'>
        <div role='tabpanel' hidden={activeTab !== 'global'}>
          <GlobalSettingsCard
            defaultValues={defaults.global}
            tabRef={setHandle('global')}
          />
        </div>
        <div role='tabpanel' hidden={activeTab !== 'gemini'}>
          <GeminiSettingsCard
            defaultValues={defaults.gemini}
            tabRef={setHandle('gemini')}
          />
        </div>
        <div role='tabpanel' hidden={activeTab !== 'claude'}>
          <ClaudeSettingsCard
            defaultValues={defaults.claude}
            tabRef={setHandle('claude')}
          />
        </div>
        <div role='tabpanel' hidden={activeTab !== 'grok'}>
          <GrokSettingsCard
            defaultValues={defaults.grok}
            tabRef={setHandle('grok')}
          />
        </div>
      </div>

      <SettingsPageFormActions onSave={handleSave} isSaving={isSaving} />

      <AlertDialog
        open={pendingTab !== null}
        onOpenChange={handleDialogOpenChange}
      >
        <AlertDialogContent>
          <AlertDialogHeader className='text-start'>
            <AlertDialogTitle>{t('Unsaved changes')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                'You have unsaved changes in the current tab. What would you like to do?',
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('Cancel')}</AlertDialogCancel>
            <Button variant='outline' onClick={handleDiscard}>
              {t('Discard')}
            </Button>
            <AlertDialogAction onClick={handleSave} disabled={isSaving}>
              {t('Save and switch')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </SettingsSection>
  )
}