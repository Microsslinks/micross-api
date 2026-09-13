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
import type { TopNavLink } from '../types'
import { Container, type ContainerWidth } from './container'
import { Footer } from './footer'
import { PublicHeader, type PublicHeaderProps } from './public-header'

type PublicLayoutProps = {
  children: React.ReactNode
  /**
   * Content width token applied by the layout (see `.docs/frontend/layout-system.md`).
   *
   * Pass `false` only for surfaces that own their layout end to end — a
   * full-bleed iframe, admin-authored HTML, or a hero-first landing page that
   * handles its own top offset and width. Such pages also default to no
   * header offset and no footer, both of which can be re-enabled explicitly.
   */
  container?: ContainerWidth | false
  /** Reserves vertical room for the floating header. Defaults to `true`. */
  headerOffset?: boolean
  /**
   * Mounts the shared page footer. Defaults to `true` — every public page is
   * expected to close with the footer. Full-bleed surfaces (iframes,
   * isolated admin HTML) opt out explicitly.
   */
  showFooter?: boolean
  navContent?: React.ReactNode
  headerProps?: Omit<PublicHeaderProps, 'navContent'>
  navLinks?: TopNavLink[]
  showThemeSwitch?: boolean
  showAuthButtons?: boolean
  showNotifications?: boolean
  logo?: React.ReactNode
  siteName?: string
}

export function PublicLayout(props: PublicLayoutProps) {
  // A page that owns its own layout renders no <main> wrapper, so the header
  // offset (which lives on that wrapper) does not apply to it.
  const ownsLayout = props.container === false
  const headerOffset = props.headerOffset ?? !ownsLayout
  const contentWidth: ContainerWidth =
    props.container === undefined || props.container === false
      ? 'default'
      : props.container

  return (
    <div className='bg-background text-foreground relative min-h-svh overflow-x-clip'>
      <PublicHeader
        navContent={props.navContent}
        navLinks={props.navLinks}
        showThemeSwitch={props.showThemeSwitch}
        showAuthButtons={props.showAuthButtons}
        showNotifications={props.showNotifications}
        logo={props.logo}
        siteName={props.siteName}
        {...props.headerProps}
      />

      {ownsLayout ? (
        props.children
      ) : (
        <main className={headerOffset ? 'pt-16 sm:pt-20' : undefined}>
          <Container width={contentWidth}>{props.children}</Container>
        </main>
      )}

      {(props.showFooter ?? true) && <Footer />}
    </div>
  )
}
