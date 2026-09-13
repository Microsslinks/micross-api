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

import { CUSTOMER_CODE_REGEX, registerFormSchema } from './constants'

// 客户号是注册页唯一的选填业务字段。这里的规则必须与后端签发规则一致
// （model/agent_code.go：前缀 AG + 10 位大写字母数字），否则客户抄对了号也会被前端挡下，
// 或者抄错了要等提交一趟才知道。
describe('register form customer code', () => {
  const baseForm = {
    username: 'customer',
    email: '',
    password: 'password123',
    confirmPassword: 'password123',
    customerCode: '',
  }

  function customerCodeError(customerCode: string) {
    const result = registerFormSchema.safeParse({ ...baseForm, customerCode })
    if (result.success) return null
    return (
      result.error.issues.find((issue) => issue.path[0] === 'customerCode')
        ?.message ?? null
    )
  }

  test('accepts a form without a customer code', () => {
    expect(customerCodeError('')).toBeNull()
  })

  test('accepts a lower-case code, which is how it is usually copied by hand', () => {
    expect(customerCodeError('ag7k2m9p4qxz')).toBeNull()
  })

  test('accepts a code with surrounding spaces', () => {
    expect(customerCodeError('  AG7K2M9P4QXZ  ')).toBeNull()
  })

  test('rejects a code that is not AG plus ten characters', () => {
    const message = 'Customer codes are 12 characters starting with AG'
    expect(customerCodeError('AG123')).toBe(message)
    expect(customerCodeError('BG7K2M9P4QXZ')).toBe(message)
    // 12 位但字母数字之外还有别的字符（客户号只会是字母数字）
    expect(customerCodeError('AG7K2M9P4Q-X')).toBe(message)
    // 少一位：最容易犯的抄写错误，必须在提交之前就挡住
    expect(customerCodeError('AG7K2M9P4QX')).toBe(message)
  })

  test('CUSTOMER_CODE_REGEX matches the issued format', () => {
    expect(CUSTOMER_CODE_REGEX.test('AG7K2M9P4QXZ')).toBe(true)
    expect(CUSTOMER_CODE_REGEX.test('AG7K2M9P4QX')).toBe(false)
  })
})
