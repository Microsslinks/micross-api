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
import { LayoutGrid } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

import { AGENTS } from '../constants'
import { AgentLinkCard } from './agent-link-card'

export function AgentsMoreDialog() {
  const { t } = useTranslation()

  return (
    <Dialog>
      <DialogTrigger className='group border-border/40 text-muted-foreground shadow-card hover:border-border hover:bg-muted/30 hover:text-foreground hover:shadow-card-lift flex min-w-0 items-center gap-2 rounded-xl border border-dashed px-3 py-2.5 text-[13px] font-semibold transition-all duration-300 hover:-translate-y-1'>
        <span className='bg-background/60 grid size-8 shrink-0 place-items-center rounded-lg'>
          <LayoutGrid className='size-4' />
        </span>
        <span className='truncate group-hover:underline group-hover:underline-offset-3'>
          {t('View More')}
        </span>
      </DialogTrigger>
      <DialogContent className='sm:max-w-3xl'>
        <DialogHeader>
          <DialogTitle className='text-sm font-semibold'>
            {t('Agent Ecosystem')}
          </DialogTitle>
          <DialogDescription className='text-xs'>
            {t(
              'OpenAI-compatible interface. Point any agent at Microsslink with your API key and it is ready to run.'
            )}
          </DialogDescription>
        </DialogHeader>
        <div className='-mr-1 max-h-[60vh] overflow-y-auto pr-1'>
          <div className='grid gap-2.5 sm:grid-cols-2 lg:grid-cols-3'>
            {AGENTS.map((agent) => (
              <AgentLinkCard key={agent.id} agent={agent} showDescription />
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
