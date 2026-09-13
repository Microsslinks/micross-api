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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeAll, describe, expect, test } from 'vitest'

import type { CustomerCode } from '../../types'
import type { CustomerCodesSource } from '../customer-codes-panel'

const i18n = (await import('i18next')).default
const { CustomerCodesPanel } = await import('../customer-codes-panel')

// 邀请文案里带着经销商自己的域名，这里只验"鼠标停上去能看到整段文案"，
// 文案内容用测试自己的词条，免得跟着产品文案一起改。
beforeAll(() => {
  i18n.addResourceBundle(
    'en',
    'translation',
    {
      'Dealer customer invitation message':
        'Invite code {{code}} | link {{link}}',
    },
    true,
    true
  )
})

const AVAILABLE_CODE: CustomerCode = {
  id: 7,
  code: 'KR7M2Q',
  agent_id: 3,
  plan_id: 0,
  max_uses: 1,
  used_count: 0,
  bound_user_id: 0,
  bound_username: '',
  bound_display_name: '',
  expired_at: 0,
  status: 1,
  remark: '',
  created_at: 1700000000,
  updated_at: 1700000000,
}

function makeSource(items: CustomerCode[]): CustomerCodesSource {
  return {
    list: async () => ({ items, total: items.length }),
    create: async () => ({ success: true }),
    revoke: async () => ({ success: true }),
    plans: [],
  }
}

describe('customer codes panel', () => {
  test('鼠标停在复制按钮上时弹出完整的邀请文案', async () => {
    const user = userEvent.setup()
    render(<CustomerCodesPanel source={makeSource([AVAILABLE_CODE])} />)

    const copyButton = await screen.findByRole('button', { name: 'Copy' })
    expect(screen.queryByText(/Invite code KR7M2Q/)).toBeNull()

    await user.hover(copyButton)

    await waitFor(
      () => {
        expect(screen.getByText(/Invite code KR7M2Q/)).toBeTruthy()
      },
      { timeout: 3000 }
    )
  })
})
