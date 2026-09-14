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
import { Link } from '@tanstack/react-router'
import { ArrowRight, BookOpen } from 'lucide-react'
import { motion, useReducedMotion, useScroll, useTransform } from 'motion/react'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Container } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { useMediaQuery } from '@/hooks'
import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

import { HeroAgents } from '../hero-agents'
import { HeroTerminalDemo } from '../hero-terminal-demo'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

/*
 * Cinematic hero backdrops. One is picked at random per page load, so every
 * visit can land on a different scene. `scrim` is a per-scene darkening boost
 * (0 = default video scrim) for footage that runs brighter than the rest.
 */
const HERO_BACKGROUNDS = [
  { src: '/home/home-bg/home-bg-001.mp4', scrim: 0 },
  { src: '/home/home-bg/home-bg-002.mp4', scrim: 0 },
  { src: '/home/home-bg/home-bg-003.mp4', scrim: 0 },
  { src: '/home/home-bg/home-pg-004.mp4', scrim: 0 },
  { src: '/home/home-bg/home-pg-005.mp4', scrim: 0 },
] as const

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl = (status?.docs_link as string | undefined) || ''

  const stageRef = useRef<HTMLElement>(null)
  const reduceMotion = useReducedMotion()

  // Pick once per mount so re-renders (scroll, status refresh) never swap the
  // scene mid-view.
  const [heroBackground] = useState(
    () =>
      HERO_BACKGROUNDS[
        Math.floor(Math.random() * HERO_BACKGROUNDS.length)
      ] as (typeof HERO_BACKGROUNDS)[number]
  )  // Videos decode asynchronously; fade the backdrop in once frames are ready
  // instead of flashing black over the section.
  const [backdropReady, setBackdropReady] = useState(false)
  /*
   * The cinematic pin only holds while the hero fits one screen. Below `lg`
   * the two columns stack, so the copy, the agent grid and the terminal card
   * need roughly 1000px against a ~670px phone viewport: a pinned, clipped
   * screen would push the headline and the CTAs out of view entirely. The
   * breakpoint MUST match the `hero-scroll-stage` rules in styles/index.css.
   */
  const isPinned = useMediaQuery('(min-width: 1024px)') && !reduceMotion

  // Progress 0 → 1 spans exactly the pinned travel, because the stage height is
  // `100svh + travel` (see `.hero-scroll-stage` in styles/index.css).
  const { scrollYProgress } = useScroll({
    target: stageRef,
    offset: ['start start', 'end end'],
  })

  const backdropScale = useTransform(scrollYProgress, [0, 1], [1, 1.08])
  const backdropOpacity = useTransform(scrollYProgress, [0, 0.55, 1], [1, 1, 0])
  const contentScale = useTransform(scrollYProgress, [0, 1], [1, 0.94])
  const contentY = useTransform(scrollYProgress, [0, 1], [0, -64])
  const contentOpacity = useTransform(scrollYProgress, [0, 0.4, 1], [1, 1, 0])
  const contentBlur = useTransform(
    scrollYProgress,
    [0, 0.45, 1],
    ['blur(0px)', 'blur(0px)', 'blur(8px)']
  )
  const titleY = useTransform(scrollYProgress, [0, 1], [0, -28])

  // Unpinned screens have no scroll travel, so the transforms would fire
  // against a stage that never advances — drop them and render statically.
  const backdropStyle = isPinned
    ? { scale: backdropScale, opacity: backdropOpacity }
    : undefined
  const contentStyle = isPinned
    ? {
        scale: contentScale,
        y: contentY,
        opacity: contentOpacity,
        filter: contentBlur,
      }
    : undefined
  const titleStyle = isPinned ? { y: titleY } : undefined

  const renderDocsButton = () => {
    if (!docsUrl) return null
    const isExternal = docsUrl.startsWith('http')
    if (isExternal) {
      return (
        <Button
          variant='outline'
          size='xl'
          className='group border-border/50 hover:border-border hover:bg-muted/50'
          render={
            <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
          }
        >
          <BookOpen className='text-muted-foreground/80 group-hover:text-foreground size-4 transition-colors duration-200' />
          <span>{t('Docs')}</span>
        </Button>
      )
    }
    return (
      <Button
        variant='outline'
        size='xl'
        className='group border-border/50 hover:border-border hover:bg-muted/50'
        render={<Link to={docsUrl} />}
      >
        <BookOpen className='text-muted-foreground/80 group-hover:text-foreground size-4 transition-colors duration-200' />
        <span>{t('Docs')}</span>
      </Button>
    )
  }

  return (
    /*
     * The first screen is a deep-space backdrop, so it must render with the dark
     * palette no matter which theme the visitor picked.
     *
     * `dark` scopes the dark token set to this section and therefore also drives
     * the `dark:` variants inside it; `bg-black` is the fallback that keeps the
     * section opaque while the photo decodes (and if it ever fails to load),
     * instead of showing `PublicLayout`'s near-white page background. The
     * `data-hero-stage` hook lets the floating header keep its dark styling for
     * as long as this stage still sits behind it.
     *
     * `text-foreground` is load-bearing, not decoration. `dark` only swaps the
     * CSS variables inside this subtree; bare text nodes (the first headline
     * line, agent card names, the label inside the outline CTAs) do not re-read
     * them, they inherit `color` from `body`. On the light theme `body` has
     * already resolved to near-black, so those nodes vanish into the black
     * backdrop. Re-anchoring `color` here makes every inherited node resolve
     * against this section's dark `--foreground` instead.
     */
    <section
      ref={stageRef}
      data-hero-stage
      className={cn(
        'dark text-foreground relative z-10 bg-black',
        isPinned && 'hero-scroll-stage'
      )}
    >
      {/* Pinned screen: stays put for the whole travel distance. Unpinned
          screens (below `lg`, or reduced motion) are plain flow blocks. */}
      <div
        className={cn(
          isPinned ? 'sticky top-0 h-svh overflow-hidden' : 'relative'
        )}
      >
        <motion.div
          aria-hidden
          className='pointer-events-none absolute inset-0'
          style={backdropStyle}
        >
          {/*
           * One random scene from the cinematic pool, cross-faded in once it
           * can play. Deliberately NOT gated on `prefers-reduced-motion`:
           * this is a slow, ambient marketing backdrop (muted, no camera
           * cuts), and gating it made every visitor whose OS disables UI
           * animations land on the static fallback. Only the scroll pin
           * below still respects the reduced-motion preference.
           */}
          <video
            key={heroBackground.src}
            src={heroBackground.src}
            autoPlay
            muted
            loop
            playsInline
            preload='auto'
            disablePictureInPicture
            aria-hidden
            onCanPlay={() => setBackdropReady(true)}
            className={cn(
              'size-full object-cover object-center select-none transition-opacity duration-1000',
              backdropReady ? 'opacity-100' : 'opacity-0'
            )}
          />
        </motion.div>

        {/* Grid pattern, carried over from the previous hero for texture. */}
        <div
          aria-hidden
          className='pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_60%_50%_at_40%_40%,black_20%,transparent_100%)] bg-[size:4rem_4rem] opacity-[0.06]'
        />

        {/* Base scrim: near-opaque on the copy side, but stays tinted all the
            way across so even the brightest footage reads as a dark scene. */}
        <div
          aria-hidden
          className='pointer-events-none absolute inset-0 bg-gradient-to-r from-black/95 via-black/75 to-black/35'
        />
        {/* Dedicated scrim for video backdrops: they carry bright, moving
            scenes that fight the copy far harder than a static photo ever
            did, so they get their own extra gradient on top of the shared
            one. `heroBackground.scrim` darkens a specific scene further
            (0–0.25) without touching the others; tune per entry in
            HERO_BACKGROUNDS above. */}
        <div
          aria-hidden
          className='pointer-events-none absolute inset-0'
          style={{
            background:
              'linear-gradient(to right, rgba(0,0,0,0.8) 0%, rgba(0,0,0,0.6) 45%, rgba(0,0,0,0.4) 100%)',
            opacity: 0.85 + heroBackground.scrim,
          }}
        />
        {/* Bottom scrim dissolves the screen instead of cutting it off. */}
        <div
          aria-hidden
          className='pointer-events-none absolute inset-x-0 bottom-0 h-40 bg-gradient-to-b from-transparent to-black/70'
        />

        {/*
         * `py-28` on unpinned screens keeps the copy clear of the floating
         * header, which PublicLayout does not offset for a full-bleed page.
         */}
        <motion.div
          className={cn(
            'relative flex',
            isPinned ? 'h-full' : 'min-h-svh py-28'
          )}
          style={contentStyle}
        >
          {/*
           * `my-auto` where `items-center` used to sit: auto margins collapse
           * to zero once the content outgrows the screen, so a short viewport
           * keeps the headline pinned to the top instead of clipping it off
           * both ends.
           */}
          <Container className='my-auto grid grid-cols-1 items-center gap-12 lg:grid-cols-12 lg:gap-8'>
            {/* Left Column: Title, description, action buttons and application support */}
            <div className='flex flex-col items-start text-left lg:col-span-6'>
              {/* Top Pill Badge */}
              <div
                className='landing-animate-fade-up mb-5 inline-flex items-center gap-1.5 rounded-full border border-blue-500/20 bg-blue-500/5 px-3 py-1.5 text-[11px] font-medium text-blue-600 opacity-0 shadow-xs dark:border-blue-400/20 dark:bg-blue-400/5 dark:text-blue-400'
                style={{ animationDelay: '0ms' }}
              >
                <span className='relative flex size-1.5'>
                  <span className='absolute inline-flex h-full w-full animate-ping rounded-full bg-blue-400 opacity-75' />
                  <span className='relative inline-flex size-1.5 rounded-full bg-blue-500 dark:bg-blue-400' />
                </span>
                <span>{t('Token Factory Era Has Arrived')}</span>
              </div>

              <motion.div style={titleStyle}>
                <h1
                  className='landing-animate-fade-up text-[clamp(2.25rem,4.5vw,3.25rem)] leading-[1.15] font-bold tracking-tight'
                  style={{ animationDelay: '60ms' }}
                >
                  {t('One API Gateway')}
                  <br />
                  <span className='brand-gradient-text'>
                    {t('Every Model at Your Fingertips')}
                  </span>
                </h1>
              </motion.div>

              <div
                className='landing-animate-fade-up mt-8 flex flex-wrap items-center gap-3 opacity-0'
                style={{ animationDelay: '180ms' }}
              >
                {props.isAuthenticated ? (
                  <>
                    <Button
                      size='xl'
                      className='group'
                      render={<Link to='/dashboard' />}
                    >
                      {t('Go to Dashboard')}
                      <ArrowRight className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
                    </Button>
                    {renderDocsButton()}
                  </>
                ) : (
                  <>
                    <Button
                      size='xl'
                      className='group'
                      render={<Link to='/sign-up' />}
                    >
                      {t('Get Started')}
                      <ArrowRight className='ml-1.5 size-4 transition-transform duration-200 group-hover:translate-x-0.5' />
                    </Button>
                    <Button
                      variant='outline'
                      size='xl'
                      className='border-border/50 hover:border-border hover:bg-muted/50'
                      render={<Link to='/pricing' />}
                    >
                      {t('View Pricing')}
                    </Button>
                    {renderDocsButton()}
                  </>
                )}
              </div>

              {/* AI Agents (只放 6 个精选，其余在「查看更多」弹窗里) */}
              <HeroAgents />
            </div>

            {/*
             * Right Column: Hero Terminal API Demo
             *
             * Deliberately held back ~2s and revealed with the slower cinematic
             * curve: the console is opaque enough to compete with the backdrop,
             * so it has to arrive after the copy has landed and drift in
             * gradually rather than covering the nebula straight away.
             */}
            <div
              className='landing-animate-cinematic-emerge flex w-full justify-center opacity-0 lg:col-span-6'
              style={{ animationDelay: '2000ms' }}
            >
              <HeroTerminalDemo className='mt-8 lg:mt-0' />
            </div>
          </Container>
        </motion.div>
      </div>
    </section>
  )
}
