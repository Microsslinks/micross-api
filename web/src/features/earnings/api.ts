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
import type {
  ApiResponse,
  AffiliateCodeResponse,
  AffiliateTransferRequest,
  AffiliateTransferResponse,
  CommissionBalanceResponse,
  CommissionRecordsResponse,
  CommissionSummaryResponse,
} from '@/features/wallet/types'
import { api } from '@/lib/api'

// ============================================================================
// Earnings API Functions
//
// Referral commission endpoints. They belong to the customer's own earnings
// surface rather than to the wallet: the wallet page only holds money going
// out (top-up, redemption), while commission is money coming in.
// ============================================================================

/**
 * Get the signed-in user's referral code
 */
export async function getAffiliateCode(): Promise<AffiliateCodeResponse> {
  const res = await api.get('/api/user/aff')
  return res.data
}

/**
 * Transfer accumulated commission to the main balance
 */
export async function transferAffiliateQuota(
  request: AffiliateTransferRequest
): Promise<AffiliateTransferResponse> {
  const res = await api.post('/api/user/aff_transfer', request)
  return res.data
}

/**
 * Get commission wallet balance (task-10 P4 commission core).
 *
 * 独立钱包余额——不能直接消费，需通过 API 调用走主 quota 扣减。
 * 与 getAffiliateCode / transferAffiliateQuota 并列挂在
 * /api/user/aff/commission/*。
 */
export async function getCommissionBalance(): Promise<CommissionBalanceResponse> {
  const res = await api.get('/api/user/aff/commission/balance')
  return res.data
}

/**
 * List commission records with pagination.
 */
export async function listCommissionRecords(
  page = 1,
  pageSize = 20,
): Promise<CommissionRecordsResponse> {
  const res = await api.get('/api/user/aff/commission/records', {
    params: { page, page_size: pageSize },
  })
  return res.data
}

/**
 * Get commission summary (total amount / record count / breach count).
 */
export async function getCommissionSummary(): Promise<CommissionSummaryResponse> {
  const res = await api.get('/api/user/aff/commission/summary')
  return res.data
}

/**
 * Transfer commission wallet balance to main quota.
 *
 * 永远会失败——commission 不可提现（任务文档 §三"资金闭环"）。
 * 前端保留这个函数是为了语义统一（affiliate / commission 各有 transfer，
 * 但 commission 永远报错）。
 */
export async function transferCommissionQuota(
  request: { quota: number },
): Promise<ApiResponse> {
  const res = await api.post('/api/user/aff/commission/transfer', request)
  return res.data
}
