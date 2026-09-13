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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import i18next from 'i18next'
import { beforeAll, beforeEach, describe, expect, test, vi } from 'vitest'

import { DiscountSettingSection } from '../discount-setting-section'

const { mutateAsync } = vi.hoisted(() => ({ mutateAsync: vi.fn() }))

// The option update goes over the network; keeping it out of the test is what
// lets us assert the exact payload the section sends.
vi.mock('../../hooks/use-update-option', () => ({
  useUpdateOption: () => ({ mutateAsync, isPending: false }),
}))

const MARGIN_LABEL = 'Minimum Margin Ratio'
const MARGIN_INVALID_MESSAGE =
  'Enter a number between 0 and 1 with up to 6 decimal places, or leave empty'
const BILLING_LABEL = 'Apply discounts to billing'

const renderSection = (marginRatio: string, billingDiscount = false) => {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  })
  const view = render(
    <QueryClientProvider client={queryClient}>
      <DiscountSettingSection
        defaultValues={{
          'discount_setting.min_margin_ratio': marginRatio,
          'discount_setting.enable_billing_discount': billingDiscount,
        }}
      />
    </QueryClientProvider>
  )

  const input = screen.getByRole('spinbutton', { name: MARGIN_LABEL })
  const form = view.container.querySelector('form')

  if (!form) throw new Error('Discount section did not render a form')

  return { ...view, queryClient, input, form }
}

describe('discount margin setting', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      [MARGIN_LABEL]: MARGIN_LABEL,
      [MARGIN_INVALID_MESSAGE]: MARGIN_INVALID_MESSAGE,
      [BILLING_LABEL]: BILLING_LABEL,
    })
  })

  beforeEach(() => {
    mutateAsync.mockClear()
  })

  test('shows the margin ratio stored on the server', () => {
    const { queryClient, input } = renderSection('0.05')

    expect(input).toHaveValue(0.05)

    queryClient.clear()
  })

  test('stores an emptied margin ratio as an explicit zero', async () => {
    const { queryClient, input, form } = renderSection('0.05')

    fireEvent.change(input, { target: { value: '' } })
    fireEvent.submit(form)

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({
        key: 'discount_setting.min_margin_ratio',
        value: '0',
      })
    )

    queryClient.clear()
  })

  test('saves the billing switch turned on, on its own key', async () => {
    const { queryClient, form } = renderSection('0', false)

    fireEvent.click(screen.getByRole('switch', { name: BILLING_LABEL }))
    fireEvent.submit(form)

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({
        key: 'discount_setting.enable_billing_discount',
        value: true,
      })
    )

    queryClient.clear()
  })

  test('leaves the billing switch untouched when only the margin changed', async () => {
    const { queryClient, input, form } = renderSection('0.05', true)

    fireEvent.change(input, { target: { value: '0.1' } })
    fireEvent.submit(form)

    await waitFor(() =>
      expect(mutateAsync).toHaveBeenCalledWith({
        key: 'discount_setting.min_margin_ratio',
        value: '0.1',
      })
    )
    expect(mutateAsync).not.toHaveBeenCalledWith({
      key: 'discount_setting.enable_billing_discount',
      value: true,
    })

    queryClient.clear()
  })

  test('blocks a margin ratio above one instead of saving it', async () => {
    const { queryClient, input, form } = renderSection('0')

    fireEvent.change(input, { target: { value: '1.5' } })
    fireEvent.submit(form)

    await waitFor(() => expect(input).toHaveAttribute('aria-invalid', 'true'))
    expect(screen.getByText(MARGIN_INVALID_MESSAGE)).toBeInTheDocument()
    expect(mutateAsync).not.toHaveBeenCalled()

    queryClient.clear()
  })
})
