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
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

import type { PromoModel } from '../constants'
import { PromoLogo } from './model-logos'

interface PromoModelCardProps {
  model: PromoModel
  delay?: number
}

export function PromoModelCard(props: PromoModelCardProps) {
  const { t } = useTranslation()

  return (
    <AnimateInView
      as='li'
      animation='fade-up'
      delay={props.delay}
      className='group border-border/50 bg-card shadow-card hover:border-border hover:shadow-card-lift relative overflow-hidden rounded-2xl border transition-all duration-300 hover:-translate-y-1.5'
    >
      <span className='absolute top-0 right-0 flex flex-col items-end gap-0.5 rounded-bl-[22px] bg-gradient-to-br from-[#ff6a3d] to-[#ff2d55] px-4 py-2.5 text-white shadow-md'>
        <i className='text-[10px] leading-none font-bold tracking-[0.1em] uppercase not-italic opacity-90'>
          {t('Up to')}
        </i>
        <b className='text-xl leading-tight font-black tracking-tight'>
          {t(props.model.promo)}
        </b>
      </span>

      <div className='p-6'>
        <div className='flex items-center gap-3.5'>
          <PromoLogo brand={props.model.brand} />
          <div>
            <h4 className='text-lg font-bold tracking-tight'>
              {props.model.name}
            </h4>
            <p className='text-muted-foreground mt-0.5 text-xs'>
              {t(props.model.maker)}
            </p>
          </div>
        </div>

        <div className='mt-5 flex flex-col gap-2'>
          {props.model.plans.map((plan) => (
            <div
              key={plan.id}
              className='border-border/40 bg-muted/20 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 rounded-2xl border px-3.5 py-2'
            >
              <i className='font-mono text-[11px] font-semibold not-italic'>
                {plan.id}
              </i>
              <em className='text-muted-foreground text-[10px] font-medium not-italic tabular-nums'>
                <b className='font-semibold'>{t('Input')}</b> {plan.input} ·{' '}
                <b className='font-semibold'>{t('Output')}</b> {plan.output} ·{' '}
                <b className='font-semibold'>{t('Cache')}</b> {plan.cache}
              </em>
            </div>
          ))}
        </div>

        <p className='text-muted-foreground/70 mt-5 text-[10px] font-medium'>
          {t('Vendor list price · per 1M tokens')}
        </p>
      </div>
    </AnimateInView>
  )
}
