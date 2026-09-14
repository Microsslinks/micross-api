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
import { useQuery } from '@tanstack/react-query'
import {
  ArrowRight,
  Building2,
  Check,
  Coins,
  Code,
  Crown,
  Database,
  Gauge,
  Globe,
  Layers,
  Link2,
  Server,
  ShieldCheck,
  Sparkles,
  Users,
  Zap,
} from 'lucide-react'
import { Trans, useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { RichContent } from '@/components/rich-content'
import { Skeleton } from '@/components/ui/skeleton'
import { isHttpUrl, isLikelyHtml } from '@/lib/content-format'
import { cn } from '@/lib/utils'

import { getAboutContent } from './api'

/** 蓝→紫品牌渐变文字。基线 A 的统一写法，唯一实现在 `styles/index.css` 的 `.brand-gradient-text`。 */
const GRADIENT_TEXT = 'brand-gradient-text'

/** 卡片色调：仅作图标底色与角部光晕，语义色取直觉关联。 */
const TONES = {
  blue: {
    shell: 'bg-blue-500/10 text-blue-600 dark:text-blue-400',
    glow: 'bg-blue-500/10',
  },
  emerald: {
    shell: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
    glow: 'bg-emerald-500/10',
  },
  violet: {
    shell: 'bg-violet-500/10 text-violet-600 dark:text-violet-400',
    glow: 'bg-violet-500/10',
  },
} as const

type Tone = keyof typeof TONES

function SectionHeading(props: {
  label: string
  title: string
  note?: string
}) {
  return (
    <div className='mb-5 flex flex-col gap-2 md:flex-row md:items-end md:justify-between md:gap-8'>
      <div>
        <p className='text-primary text-xs font-bold tracking-[0.14em] uppercase'>
          {props.label}
        </p>
        <h2 className='mt-1.5 text-2xl font-bold tracking-tight md:text-3xl'>
          {props.title}
        </h2>
      </div>
      {props.note && (
        <p className='text-muted-foreground max-w-md text-sm leading-relaxed md:text-right'>
          {props.note}
        </p>
      )}
    </div>
  )
}

/** 内容卡：`rounded-2xl` + 弱描边 + 角部光晕（基线 A 的公开页卡片配方）。 */
function ContentCard(props: {
  tone: Tone
  className?: string
  children: React.ReactNode
}) {
  return (
    <div
      className={cn(
        'border-border/50 bg-card relative overflow-hidden rounded-2xl border p-6 transition-shadow duration-200 hover:shadow-md',
        props.className
      )}
    >
      <div
        aria-hidden
        className={cn(
          'pointer-events-none absolute -top-20 -right-20 size-44 rounded-full blur-2xl',
          TONES[props.tone].glow
        )}
      />
      <div className='relative'>{props.children}</div>
    </div>
  )
}

function CardBadge(props: {
  tone: Tone
  icon: React.ReactNode
  title: string
}) {
  return (
    <div className='flex items-center gap-3.5'>
      <span
        className={cn(
          'flex size-13 shrink-0 items-center justify-center rounded-2xl',
          TONES[props.tone].shell
        )}
      >
        {props.icon}
      </span>
      <div>
        <h3 className='text-lg font-bold tracking-tight'>{props.title}</h3>
      </div>
    </div>
  )
}

function Tag(props: { children: React.ReactNode }) {
  return (
    <span className='bg-muted/50 text-muted-foreground rounded-md px-2.5 py-1.5 text-xs font-medium'>
      {props.children}
    </span>
  )
}

function TagRow(props: { children: React.ReactNode }) {
  return <div className='mt-4 flex flex-wrap gap-2'>{props.children}</div>
}

/**
 * 管理员未配置「关于」内容时的默认页面：微观互联公司页。
 *
 * 文案与结构来自业务方提供的公司资料（`.docs/frontend/ui-consistency.md` §六）。
 * 文本全部走 i18n（`en` 为 key，`zh` / `zh-TW` 已译，其余语言英文回落）。
 *
 * 上游署名与许可声明（Powered by / Based on / License）属受保护信息，
 * 已统一收敛到全局 Footer；版式可调，但品牌名与链接不得删除或改写。
 * 见根目录 AGENTS.md「Protected project information」。
 */
function CompanyProfile() {
  const { t } = useTranslation()

  const transformItems = [
    {
      tone: 'blue' as Tone,
      icon: <Sparkles className='size-5' strokeWidth={1.7} />,
      title: t('AI technology services'),
      body: t(
        'With large models and intelligent technology at the core, we provide AI technology services to customers at home and abroad.'
      ),
    },
    {
      tone: 'violet' as Tone,
      icon: <Coins className='size-5' strokeWidth={1.7} />,
      title: t('Token services'),
      body: t(
        'Token-based products and services that build a value circulation system for the future.'
      ),
    },
    {
      tone: 'emerald' as Tone,
      icon: <Layers className='size-5' strokeWidth={1.7} />,
      title: t('Token factory'),
      body: t(
        'Build and operate a Token factory serving customers at home and abroad, scaling AI and Token capability.'
      ),
    },
  ]

  const techItems = [
    {
      icon: <Link2 className='size-5' strokeWidth={1.7} />,
      name: t('Blockchain core technology'),
      meta: t('Independent and controllable'),
    },
    {
      icon: <Database className='size-5' strokeWidth={1.7} />,
      name: t('Big data'),
      meta: t('Partner of the GAC'),
    },
    {
      icon: <Globe className='size-5' strokeWidth={1.7} />,
      name: t('IoT'),
      meta: t('Full-chain visibility'),
    },
    {
      icon: <Zap className='size-5' strokeWidth={1.7} />,
      name: t('Intelligent technology'),
      meta: t('AI driven'),
    },
  ]

  const honors = [
    {
      tone: 'blue' as Tone,
      icon: <ShieldCheck className='size-6' strokeWidth={1.7} />,
      title: t('Big data value-added service provider'),
      maker: t('General Administration of Customs · contracted'),
      tags: [t('Officially contracted'), t('Data services')],
      body: t(
        'A big data value-added service provider contracted by the General Administration of Customs, serving customs supervision and trade facilitation with data capability.'
      ),
    },
    {
      tone: 'emerald' as Tone,
      icon: <Gauge className='size-6' strokeWidth={1.7} />,
      title: t('Big data joint laboratory'),
      maker: t('Co-built with the GAC'),
      tags: [t('Jointly built'), t('Research cooperation')],
      body: t(
        'Co-built the “Big Data Joint Laboratory” with the General Administration of Customs for joint research and application innovation in big data technology.'
      ),
    },
    {
      tone: 'violet' as Tone,
      icon: <Code className='size-6' strokeWidth={1.7} />,
      title: t('Blockchain smart clearance · sole pilot'),
      maker: t('GAC project · Tianjin customs district'),
      tags: [t('Sole pilot'), t('Tianjin customs district first')],
      body: t(
        'Undertakes the only pilot of the General Administration of Customs blockchain smart clearance project, implemented first in the Tianjin customs district.'
      ),
    },
  ]

  const products = [
    {
      tone: 'blue' as Tone,
      icon: <Globe className='size-6' strokeWidth={1.7} />,
      title: t('Humao Tong software platform'),
      maker: t('Proprietary · core product'),
      body: t(
        'A collaboration platform and end-to-end service for the whole cross-border trade ecosystem, connecting every stage of trade and building the best route to buy and sell globally.'
      ),
      tags: [
        t('Full-ecosystem collaboration'),
        t('End-to-end services'),
        t('Buy and sell globally'),
      ],
      flow: [] as string[],
    },
    {
      tone: 'emerald' as Tone,
      icon: <ArrowRight className='size-6' strokeWidth={1.7} />,
      title: t('Blockchain green clearance pilot'),
      maker: t(
        'Advanced with the GAC Department of Science and Technology and Information Center'
      ),
      body: t(
        'Piloting blockchain green clearance for categories such as auto parts and electronic components. Registered users on the platform directly enjoy a trust-first-supervise-later model with fast release, bringing a new clearance experience and greatly improving trade efficiency for enterprises.'
      ),
      tags: [] as string[],
      flow: [
        t('Trust first, supervise later'),
        t('Fast release'),
        t('Fully online processing'),
      ],
    },
  ]

  const deployTags = [
    t('Data stays inside your intranet'),
    t('Unified multi-upstream access'),
    t('Usage metering and billing'),
    t('Department and dealer hierarchy'),
    t('Audit logs'),
  ]

  const deployTiers = [
    {
      tone: 'blue' as Tone,
      icon: <Users className='size-5' strokeWidth={1.7} />,
      name: t('Team edition'),
      users: t('Up to 50 users'),
      deployPrice: '¥10,000',
      maintPrice: '¥5,000',
      features: [t('Version upgrades'), t('Business-hours support')],
    },
    {
      tone: 'emerald' as Tone,
      icon: <Building2 className='size-5' strokeWidth={1.7} />,
      name: t('Enterprise edition'),
      users: t('Up to 200 users'),
      deployPrice: '¥30,000',
      maintPrice: '¥15,000',
      features: [
        t('Regular health checks'),
        t('Upstream onboarding guidance'),
        t('Version upgrades'),
      ],
    },
    {
      tone: 'violet' as Tone,
      icon: <Crown className='size-5' strokeWidth={1.7} />,
      name: t('Flagship edition'),
      users: t('Up to 1000 users'),
      deployPrice: '¥80,000',
      maintPrice: '¥40,000',
      features: [
        t('SLA response times'),
        t('Dedicated support channel'),
        t('All Enterprise benefits'),
      ],
    },
  ]

  const milestones = [
    {
      num: '01',
      title: t('Cross-border trade platform'),
      body: t(
        'Built the Humao Tong platform on blockchain technology, creating a full-ecosystem collaboration and end-to-end service system.'
      ),
      now: false,
    },
    {
      num: '02',
      title: t('Deep cooperation with the GAC'),
      body: t(
        'Contracted as a big data value-added service provider, co-built the big data joint laboratory, and undertook the only blockchain smart clearance pilot.'
      ),
      now: false,
    },
    {
      num: '03',
      title: t('2025 · Strategic transformation'),
      body: t(
        'Transformed into an AI technology and Token service provider, opening a second growth curve.'
      ),
      now: true,
    },
    {
      num: '04',
      title: t('Token factory'),
      body: t(
        'Build and operate a Token factory serving customers at home and abroad, scaling AI and Token capability.'
      ),
      now: false,
    },
  ]

  return (
    <div className='flex flex-col gap-14 py-10 md:gap-16 md:py-14'>
      {/* 概述 */}
      <section className='text-center'>
        <h1 className='mx-auto max-w-4xl text-[clamp(1.75rem,3.6vw,2.5rem)] leading-[1.2] font-bold tracking-tight'>
          <Trans
            i18nKey='2025 full transition, <1>AI technology services and Token factory</1>'
            components={{ 1: <span className={GRADIENT_TEXT} /> }}
          />
        </h1>
        <p className='text-muted-foreground mx-auto mt-4 max-w-2xl text-sm leading-relaxed md:text-base'>
          {t(
            'Microsslink (Beijing) Data Services Co., Ltd. — from a cross-border trade collaboration platform to an AI technology and Token service provider for the world.'
          )}
        </p>
      </section>

      {/* 战略转型 */}
      <section>
        <SectionHeading
          label={t('2025 Transformation')}
          title={t('Strategic transformation')}
          note={t(
            'Rooted in China, serving the world — a second growth curve built on AI technology and Token services.'
          )}
        />
        <div className='border-border/50 bg-card shadow-card relative overflow-hidden rounded-3xl border p-6 md:p-8'>
          <div
            aria-hidden
            className='bg-primary/10 pointer-events-none absolute -top-28 -right-24 size-80 rounded-full blur-3xl'
          />
          <div className='relative'>
            <p className='bg-primary/10 text-primary inline-flex items-center gap-2 rounded-full px-3.5 py-1.5 text-xs font-bold'>
              <Zap className='size-3.5' strokeWidth={2} />
              {t('2025 · A defining year')}
            </p>
            <h3 className='mt-4 text-2xl leading-snug font-bold tracking-tight md:text-3xl'>
              <Trans
                i18nKey='Becoming an <1>AI technology service provider</1> and a <3>Token service provider</3>'
                components={{
                  1: <span className={GRADIENT_TEXT} />,
                  3: <span className={GRADIENT_TEXT} />,
                }}
              />
            </h3>
            <p className='text-muted-foreground mt-4 max-w-3xl text-sm leading-relaxed'>
              {t(
                'In 2025 Microsslink completed its strategic transformation, upgrading from a cross-border trade digitization provider to an AI technology and Token service provider. The goal ahead is to build and operate a Token factory that serves customers at home and abroad, continuously delivering AI capability and Token-based services.'
              )}
            </p>
            <div className='mt-6 grid gap-4 md:grid-cols-3'>
              {transformItems.map((item) => (
                <div
                  key={item.title}
                  className='border-border/50 bg-muted/25 rounded-2xl border p-5'
                >
                  <span
                    className={cn(
                      'mb-3 flex size-11 items-center justify-center rounded-xl',
                      TONES[item.tone].shell
                    )}
                  >
                    {item.icon}
                  </span>
                  <b className='block text-[15px] font-bold tracking-tight'>
                    {item.title}
                  </b>
                  <span className='text-muted-foreground mt-1.5 block text-xs leading-relaxed'>
                    {item.body}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* 公司简介 */}
      <section>
        <SectionHeading
          label={t('About us')}
          title={t('Company profile')}
          note={t(
            'Blockchain at the core, building the best route to buy and sell globally.'
          )}
        />
        <div className='border-border/50 bg-card shadow-card rounded-2xl border p-6 md:p-7'>
          <div className='grid gap-7 md:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)]'>
            <div>
              <h3 className='text-lg font-bold tracking-tight'>
                {t(
                  'A collaboration platform and service provider for the whole cross-border trade ecosystem'
                )}
              </h3>
              <p className='text-muted-foreground mt-3.5 text-sm leading-relaxed'>
                <Trans
                  i18nKey='Microsslink (Beijing) Data Services Co., Ltd. (“Microsslink”) is a <1>provider of collaboration platform, end-to-end services and extended services for the whole cross-border trade ecosystem</1>, applying <3>blockchain technology</3> together with the internet, big data, IoT and intelligent technology to build the best route to buy and sell globally.'
                  components={{
                    1: <strong className='text-foreground font-semibold' />,
                    3: <strong className='text-foreground font-semibold' />,
                  }}
                />
              </p>
              <p className='text-muted-foreground mt-3 text-sm leading-relaxed'>
                <Trans
                  i18nKey='Its core product is the <1>Humao Tong software platform</1>, on which the company holds independent intellectual property rights. As a big data value-added service provider contracted by the General Administration of Customs, it co-built the “Big Data Joint Laboratory” with the GAC and undertakes the only pilot of the GAC blockchain smart clearance project, starting in the Tianjin customs district.'
                  components={{
                    1: <strong className='text-foreground font-semibold' />,
                  }}
                />
              </p>
            </div>
            <div className='flex flex-col justify-center gap-2.5'>
              {techItems.map((item) => (
                <div
                  key={item.name}
                  className='border-border/50 bg-muted/25 flex items-center gap-3 rounded-xl border px-4 py-3'
                >
                  <span className='text-primary shrink-0'>{item.icon}</span>
                  <b className='text-sm font-semibold'>{item.name}</b>
                  <span className='text-muted-foreground ml-auto text-xs'>
                    {item.meta}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* 资质与国家级合作 */}
      <section>
        <SectionHeading
          label={t('Qualifications')}
          title={t('National-level partnerships and qualifications')}
          note={t(
            'Deep cooperation with the General Administration of Customs, undertaking national-level pilot programs.'
          )}
        />
        <div className='grid gap-4 md:grid-cols-3'>
          {honors.map((item) => (
            <ContentCard key={item.title} tone={item.tone}>
              <CardBadge tone={item.tone} icon={item.icon} title={item.title} />
              <p className='text-muted-foreground mt-1 text-xs'>{item.maker}</p>
              <TagRow>
                {item.tags.map((tag) => (
                  <Tag key={tag}>{tag}</Tag>
                ))}
              </TagRow>
              <p className='text-muted-foreground mt-4 text-[13px] leading-relaxed'>
                {item.body}
              </p>
            </ContentCard>
          ))}
        </div>
      </section>

      {/* 核心产品 */}
      <section>
        <SectionHeading
          label={t('Core products')}
          title={t('Core products and clearance innovation')}
          note={t(
            'A proprietary platform plus blockchain green clearance, reshaping trade efficiency for enterprises.'
          )}
        />
        <div className='grid gap-4 md:grid-cols-2'>
          {products.map((item) => (
            <ContentCard key={item.title} tone={item.tone}>
              <CardBadge tone={item.tone} icon={item.icon} title={item.title} />
              <p className='text-muted-foreground mt-1 text-xs'>{item.maker}</p>
              <p className='text-muted-foreground mt-4 text-sm leading-relaxed'>
                {item.body}
              </p>
              {item.tags.length > 0 && (
                <TagRow>
                  {item.tags.map((tag) => (
                    <Tag key={tag}>{tag}</Tag>
                  ))}
                </TagRow>
              )}
              {item.flow.length > 0 && (
                <div className='mt-4 flex flex-wrap gap-2.5'>
                  {item.flow.map((step) => (
                    <span
                      key={step}
                      className='border-border text-muted-foreground inline-flex items-center gap-1.5 rounded-lg border border-dashed px-3 py-2 text-xs font-semibold'
                    >
                      <ArrowRight className='text-primary size-3.5' />
                      {step}
                    </span>
                  ))}
                </div>
              )}
            </ContentCard>
          ))}
        </div>
      </section>

      {/* 私有化部署：系统免费，按用户量收部署与运维服务费 */}
      <section>
        <SectionHeading
          label={t('Private deployment')}
          title={t('Deploy this system inside your intranet')}
          note={t(
            'The software itself is free — we charge only for deployment and maintenance, priced by user count.'
          )}
        />
        <div className='border-border/50 bg-card shadow-card relative overflow-hidden rounded-3xl border p-6 md:p-8'>
          <div
            aria-hidden
            className='bg-primary/10 pointer-events-none absolute -top-28 -right-24 size-80 rounded-full blur-3xl'
          />
          <div className='relative'>
            <p className='bg-primary/10 text-primary inline-flex items-center gap-2 rounded-full px-3.5 py-1.5 text-xs font-bold'>
              <Server className='size-3.5' strokeWidth={2} />
              {t('Software: free of charge (open-source based)')}
            </p>
            <h3 className='mt-4 text-2xl leading-snug font-bold tracking-tight md:text-3xl'>
              <Trans
                i18nKey='Your own <1>AI API management gateway</1>, inside your intranet'
                components={{ 1: <span className={GRADIENT_TEXT} /> }}
              />
            </h3>
            <p className='text-muted-foreground mt-4 max-w-3xl text-sm leading-relaxed'>
              {t(
                'The same system powering this site can be deployed entirely inside your intranet as your own AI API management gateway: unified upstream access, centralized key management, usage metering and billing, department and dealer hierarchy, and audit logs — all closed-loop inside your intranet. Your data never leaves the company.'
              )}
            </p>
            <TagRow>
              {deployTags.map((tag) => (
                <Tag key={tag}>{tag}</Tag>
              ))}
            </TagRow>
          </div>
        </div>
        <div className='mt-4 grid gap-4 md:grid-cols-3'>
          {deployTiers.map((tier) => (
            <ContentCard key={tier.name} tone={tier.tone}>
              <CardBadge tone={tier.tone} icon={tier.icon} title={tier.name} />
              <p className='text-muted-foreground mt-1 text-xs'>{tier.users}</p>
              <div className='bg-muted/25 mt-4 rounded-xl p-3.5'>
                <p className='text-base font-bold'>
                  {tier.deployPrice}
                  <span className='text-muted-foreground ml-1.5 text-xs font-medium'>
                    {t('one-time deployment')}
                  </span>
                </p>
                <p className='mt-1 text-base font-bold'>
                  {tier.maintPrice}
                  <span className='text-muted-foreground ml-1.5 text-xs font-medium'>
                    {t('maintenance per year')}
                  </span>
                </p>
              </div>
              <ul className='mt-4 space-y-2'>
                {tier.features.map((feature) => (
                  <li key={feature} className='flex items-center gap-2 text-sm'>
                    <Check
                      className='text-primary size-4 shrink-0'
                      strokeWidth={2}
                    />
                    {feature}
                  </li>
                ))}
              </ul>
            </ContentCard>
          ))}
        </div>
        {/*
          联系方式（地址/电话/邮箱/社交）计划做成系统设置项，
          做好后在此处渲染具体信息；当前先放通用文案预留位置。
        */}
        <div className='border-border/50 bg-card mt-4 rounded-2xl border px-6 py-5 text-center'>
          <p className='text-sm font-semibold'>
            {t(
              'For 1000+ users, multi-node or high-availability deployments: contact us for a quote.'
            )}
          </p>
          <p className='text-muted-foreground mt-1.5 text-sm'>
            {t('For a deployment proposal and quote, please contact us.')}
          </p>
        </div>
      </section>

      {/* 发展历程 */}
      <section>
        <SectionHeading
          label={t('Milestones')}
          title={t('Development milestones')}
          note={t(
            'From a cross-border trade platform to a Token factory, one step at a time.'
          )}
        />
        <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
          {milestones.map((item) => (
            <div
              key={item.num}
              className='border-border/50 bg-card shadow-card hover:shadow-card-lift rounded-2xl border p-5 transition-all duration-300 hover:-translate-y-1'
            >
              <span
                className={cn(
                  'flex size-9 items-center justify-center rounded-xl text-sm font-bold',
                  item.now
                    ? 'bg-gradient-to-br from-blue-400 to-violet-500 text-white'
                    : 'bg-muted/50 text-primary'
                )}
              >
                {item.num}
              </span>
              <h4 className='mt-3.5 text-[15px] font-bold tracking-tight'>
                {item.title}
              </h4>
              <p className='text-muted-foreground mt-2 text-xs leading-relaxed'>
                {item.body}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* 愿景 */}
      <section className='border-border/50 bg-card shadow-card rounded-2xl border px-6 py-8 text-center'>
        <b className='block text-xl font-bold tracking-tight md:text-2xl'>
          <Trans
            i18nKey='Rooted in China, <1>serving the world</1>'
            components={{ 1: <span className={GRADIENT_TEXT} /> }}
          />
        </b>
        <span className='text-muted-foreground mt-2.5 block text-sm leading-relaxed'>
          {t(
            'Building and operating a Token factory for customers at home and abroad · Microsslink'
          )}
        </span>
      </section>

      {/* 备案号已上移到全局页脚，由「业务设置 → 站点 → 系统信息」配置，见 Footer */}
    </div>
  )
}

export function About() {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['about-content'],
    queryFn: getAboutContent,
  })

  const rawContent = data?.data?.trim() ?? ''
  const hasContent = rawContent.length > 0
  const isUrl = hasContent && isHttpUrl(rawContent)
  const contentIsHtml = hasContent && isLikelyHtml(rawContent)

  if (isLoading) {
    return (
      <PublicLayout container='default'>
        <div className='flex flex-col gap-4 py-12'>
          <Skeleton className='h-8 w-[45%]' />
          <Skeleton className='h-4 w-full' />
          <Skeleton className='h-4 w-[90%]' />
          <Skeleton className='h-4 w-[80%]' />
        </div>
      </PublicLayout>
    )
  }

  if (!hasContent) {
    return (
      <PublicLayout container='default'>
        <CompanyProfile />
      </PublicLayout>
    )
  }

  if (isUrl) {
    return (
      <PublicLayout container={false}>
        <iframe
          src={rawContent}
          className='h-[calc(100vh-3.5rem)] w-full border-0'
          title={t('About')}
          sandbox='allow-forms allow-popups allow-popups-to-escape-sandbox allow-scripts'
        />
      </PublicLayout>
    )
  }

  if (contentIsHtml) {
    return (
      <PublicLayout container='full'>
        <RichContent
          mode='html'
          htmlVariant='isolated'
          content={rawContent}
          className='prose-neutral dark:prose-invert max-w-none'
        />
      </PublicLayout>
    )
  }

  return (
    <PublicLayout container='default'>
      <div className='py-8'>
        <RichContent
          mode='markdown'
          content={rawContent}
          className='prose-neutral dark:prose-invert'
        />
      </div>
    </PublicLayout>
  )
}
