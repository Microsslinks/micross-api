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

import { buildCustomerInviteLink } from '../invite'

// 客户号只认链接这一条路：链接上的参数名必须与注册页读的一致（customer_code），
// 一旦改名，经销商拷出去的文案看上去没变，客户注册后却拿不到归属和价格。
describe('customer invite link', () => {
  test('carries the code as the register page reads it', () => {
    const link = buildCustomerInviteLink('AG7K2M9P4QXZ')

    expect(link.endsWith('/sign-up?customer_code=AG7K2M9P4QXZ')).toBe(true)
    expect(new URL(link).searchParams.get('customer_code')).toBe('AG7K2M9P4QXZ')
  })

  test('points at the sign-up route of the current site', () => {
    const link = buildCustomerInviteLink('AG7K2M9P4QXZ')

    expect(new URL(link).pathname).toBe('/sign-up')
    expect(new URL(link).origin).toBe(window.location.origin)
  })
})
