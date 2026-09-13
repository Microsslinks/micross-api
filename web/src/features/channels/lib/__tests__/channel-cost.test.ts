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
import { describe, expect, test } from 'vitest'

import {
  COST_STALE_DAYS,
  formatCostRatio,
  formatCostRatioPercent,
  isCostRatioConfigured,
  isCostRatioStale,
} from '../channel-cost'

const ONE_DAY_SECONDS = 24 * 60 * 60

describe('channel cost ratio', () => {
  test('treats unreadable or non-positive values as not configured', () => {
    for (const broken of [
      null,
      undefined,
      '',
      '   ',
      '0',
      '-0.1',
      'abc',
      'NaN',
      'Infinity',
    ]) {
      expect(isCostRatioConfigured(broken)).toBe(false)
    }

    expect(isCostRatioConfigured('0.27')).toBe(true)
    expect(isCostRatioConfigured(' 0.27 ')).toBe(true)
  })

  test('folds trailing zeros so the list matches what was typed in', () => {
    expect(formatCostRatio('0.270000')).toBe('0.27')
    expect(formatCostRatio(' 0.3000 ')).toBe('0.3')
    expect(formatCostRatio('')).toBe('')
    expect(formatCostRatio('abc')).toBe('')
  })

  test('renders the discount as a percentage for the tooltip', () => {
    expect(formatCostRatioPercent('0.27')).toBe('27%')
    expect(formatCostRatioPercent('0.275')).toBe('27.5%')
    expect(formatCostRatioPercent('1')).toBe('100%')
    expect(formatCostRatioPercent('')).toBe('')
  })

  test('counts a missing timestamp as stale, since its age cannot be known', () => {
    const nowSeconds = Math.floor(Date.now() / 1000)

    expect(isCostRatioStale(0)).toBe(true)
    expect(isCostRatioStale(null)).toBe(true)
    expect(isCostRatioStale(undefined)).toBe(true)
    expect(isCostRatioStale(nowSeconds)).toBe(false)
    expect(isCostRatioStale(nowSeconds - 29 * ONE_DAY_SECONDS)).toBe(false)
  })

  test('flags cost ratios older than the stale window', () => {
    const nowSeconds = Math.floor(Date.now() / 1000)
    const pastDefaultWindow =
      nowSeconds - (COST_STALE_DAYS * ONE_DAY_SECONDS + 60)

    expect(isCostRatioStale(pastDefaultWindow)).toBe(true)
    expect(isCostRatioStale(nowSeconds - 2 * ONE_DAY_SECONDS, 1)).toBe(true)
    expect(isCostRatioStale(nowSeconds - 100, 1)).toBe(false)
  })
})
