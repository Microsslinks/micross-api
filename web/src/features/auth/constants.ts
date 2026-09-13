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
import { z } from 'zod'

// ============================================================================
// Form Schemas
// ============================================================================

export const loginFormSchema = z.object({
  username: z.string().min(1, 'Please enter your username or email'),
  password: z.string().min(1, 'Please enter your password'),
})

export const registerFormSchema = z
  .object({
    username: z.string().min(1, 'Please enter your username'),
    email: z.string().optional(),
    password: z
      .string()
      .min(1, 'Please enter your password')
      .min(8, 'Password must be between 8 and 20 characters')
      .max(20, 'Password must be at most 20 characters long'),
    confirmPassword: z.string().min(1, 'Please confirm your password'),
    // 客户号选填：填了就在注册这一趟落归属与折扣，没填就是普通客户。
    // 格式在这里先挡一道，客户抄错一位时当场就知道，不必等提交报错。
    customerCode: z
      .string()
      .trim()
      .optional()
      .refine(
        (value) => !value || CUSTOMER_CODE_REGEX.test(value.toUpperCase()),
        { message: 'Customer codes are 12 characters starting with AG' }
      ),
  })
  .refine((data) => data.password === data.confirmPassword, {
    message: "Passwords don't match.",
    path: ['confirmPassword'],
  })

export const forgotPasswordFormSchema = z.object({
  email: z.string().email({
    message: 'Please enter a valid email address',
  }),
})

export const otpFormSchema = z.object({
  otp: z.string().min(1, 'Please enter a code.'),
})

// ============================================================================
// Validation Constants
// ============================================================================

export const PASSWORD_MIN_LENGTH = 8
export const PASSWORD_MAX_LENGTH = 20
export const OTP_LENGTH = 6
export const BACKUP_CODE_LENGTH = 9 // XXXX-XXXX format
export const BACKUP_CODE_REGEX = /^[A-Z0-9]{4}-[A-Z0-9]{4}$/i
export const OTP_REGEX = /^\d{6}$/
// 客户号：经销商签发、客户拿它来归到经销商名下的一串号码。
// 与后端 model/agent_code.go 的签发规则一致（前缀 AG + 10 位大写字母数字）。
export const CUSTOMER_CODE_REGEX = /^AG[A-Z0-9]{10}$/i

// ============================================================================
// Countdown Constants
// ============================================================================

export const EMAIL_VERIFICATION_COUNTDOWN = 30 // seconds
export const PASSWORD_RESET_COUNTDOWN = 30 // seconds

// ============================================================================
// OAuth Constants
// ============================================================================

export const OAUTH_BIND_CALLBACK_MESSAGE = 'oauth:binding:callback'
export const OAUTH_BIND_RESULT_MESSAGE = 'oauth:binding:result'
export const TELEGRAM_BIND_RESULT_MESSAGE = 'telegram:binding:result'
