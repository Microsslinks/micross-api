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
import { Calculator, ClipboardCheck, Route as RouteIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'

import { DiscountsCustomerAuditDrawer } from './components/discounts-customer-audit-drawer'
import { DiscountsDialogs } from './components/discounts-dialogs'
import { DiscountsPrimaryButtons } from './components/discounts-primary-buttons'
import { DiscountsProvider } from './components/discounts-provider'
import { DiscountsRoutingDrawer } from './components/discounts-routing-drawer'
import { DiscountsSimulateDrawer } from './components/discounts-simulate-drawer'
import { DiscountsTable } from './components/discounts-table'

/**
 * 折扣页：管方案（折扣、规则、绑了谁），再给两个按客户的入口——
 * 试算看「按这个折扣会收多少钱、每条线路赚多少」，路由策略定「钱花在哪条线路上」。
 * 折扣定的是收入，路由策略定的是成本，两者放一起正好对照着用。
 */
export function Discounts() {
  return (
    <DiscountsProvider>
      <DiscountsPage />
    </DiscountsProvider>
  )
}

function DiscountsPage() {
  const { t } = useTranslation()
  const [simulateOpen, setSimulateOpen] = useState(false)
  const [routingOpen, setRoutingOpen] = useState(false)
  const [auditOpen, setAuditOpen] = useState(false)

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>{t('Discount Plans')}</SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button variant='outline' onClick={() => setRoutingOpen(true)}>
            <RouteIcon className='h-4 w-4' />
            {t('Routing Policy')}
          </Button>
          <Button variant='outline' onClick={() => setSimulateOpen(true)}>
            <Calculator className='h-4 w-4' />
            {t('Simulate')}
          </Button>
          <Button variant='outline' onClick={() => setAuditOpen(true)}>
            <ClipboardCheck className='h-4 w-4' />
            {t('Customer Pricing Audit')}
          </Button>
          <DiscountsPrimaryButtons />
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <DiscountsTable />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <DiscountsDialogs />
      <DiscountsSimulateDrawer
        open={simulateOpen}
        onOpenChange={setSimulateOpen}
      />
      <DiscountsRoutingDrawer
        open={routingOpen}
        onOpenChange={setRoutingOpen}
      />
      <DiscountsCustomerAuditDrawer
        open={auditOpen}
        onOpenChange={setAuditOpen}
      />
    </>
  )
}
