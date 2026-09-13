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
import { Container } from '@/components/layout'
import { cn } from '@/lib/utils'

import { SECTION_PY } from '../../constants'

export function HowItWorks() {
  const { t } = useTranslation()

  const steps = [
    {
      num: '1',
      title: t('Configure'),
      desc: t(
        'Add your API keys, set up channels and configure access permissions'
      ),
      image: '/home/3-step/1-config-picture.jpg',
    },
    {
      num: '2',
      title: t('Connect'),
      desc: t('Connect via OpenAI, Claude, Gemini and more'),
      image: '/home/3-step/2-onnection.jpg',
    },
    {
      num: '3',
      title: t('Monitor'),
      desc: t('Track usage, costs and performance with real-time analytics'),
      image: '/home/3-step/3-monitor-picture.jpg',
    },
  ] as const

  return (
    <section
      className={cn('border-border/40 relative z-10 border-t', SECTION_PY)}
    >
      <Container>
        <AnimateInView className='mb-12 text-center md:mb-14'>
          <h2 className='text-2xl font-bold tracking-tight md:text-3xl'>
            {t('Three steps')}{' '}
            <span className='brand-gradient-text'>{t('to get started')}</span>
          </h2>
        </AnimateInView>

        <div className='grid gap-8 md:grid-cols-3 md:gap-10'>
          {steps.map((step, i) => (
            <AnimateInView
              key={step.num}
              delay={i * 150}
              animation='fade-up'
              className='relative'
            >
              <div className='group/step border-border bg-card shadow-card relative overflow-hidden rounded-2xl border transition-all duration-300 hover:-translate-y-1 hover:shadow-lg'>
                <div className='relative h-52 overflow-hidden md:h-56'>
                  <img
                    src={step.image}
                    alt={step.title}
                    loading='lazy'
                    className='h-full w-full object-cover transition-transform duration-500 group-hover/step:scale-105'
                  />
                  <div className='absolute inset-0 bg-gradient-to-t from-black/85 via-black/45 to-black/5' />
                  <div className='absolute inset-x-0 bottom-0 p-5'>
                    <h3 className='text-2xl leading-tight font-bold tracking-tight text-white drop-shadow-[0_2px_8px_rgba(0,0,0,0.7)] md:text-3xl'>
                      {step.title}
                    </h3>
                  </div>
                </div>
                <p className='text-muted-foreground px-5 py-5 text-sm leading-relaxed'>
                  {step.desc}
                </p>
              </div>

              <div className='bg-primary absolute -top-3 -right-2 z-10 flex size-12 items-center justify-center rounded-full shadow-md md:-right-3'>
                <span className='text-primary-foreground text-2xl leading-none font-black'>
                  {step.num}
                </span>
              </div>
            </AnimateInView>
          ))}
        </div>
      </Container>
    </section>
  )
}
