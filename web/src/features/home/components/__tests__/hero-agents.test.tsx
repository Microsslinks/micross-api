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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { beforeAll, describe, expect, test } from 'vitest'

import { AGENTS } from '../../constants'
import { HeroAgents } from '../hero-agents'

describe('hero agents block', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      'AI Agents': 'AI Agents',
      'View More': 'View More',
      'Agent Ecosystem': 'Agent Ecosystem',
    })
  })

  test('renders only the six featured agents, each as an external link', () => {
    render(<HeroAgents />)

    const featured = AGENTS.filter((agent) => agent.featured)
    expect(featured).toHaveLength(6)

    const links = screen.getAllByRole('link')
    expect(links.map((link) => link.getAttribute('href'))).toEqual(
      featured.map((agent) => agent.url)
    )
    expect(
      links.every((link) => link.getAttribute('target') === '_blank')
    ).toBe(true)
    expect(
      links.every((link) => link.getAttribute('rel')?.includes('noopener'))
    ).toBe(true)
  })

  test('opens a dialog listing every agent from the trailing card', async () => {
    const user = userEvent.setup()
    render(<HeroAgents />)

    await user.click(screen.getByRole('button', { name: 'View More' }))

    const dialog = await screen.findByRole('dialog')
    const links = within(dialog).getAllByRole('link')
    expect(links).toHaveLength(AGENTS.length)
    expect(links.map((link) => link.getAttribute('href'))).toEqual(
      AGENTS.map((agent) => agent.url)
    )
  })
})
