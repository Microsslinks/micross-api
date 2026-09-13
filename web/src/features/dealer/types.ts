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

/** 通用接口信封，与后端 common.ApiSuccess 一致。 */
export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

/** 台账里的一行：经销商发出去的一个 Key 的用量与花费。 */
export interface AgentLedgerKey {
  token_id: number
  name: string
  /** 1 启用 / 2 禁用 / 3 过期 / 4 额度用尽，取值同 API_KEY_STATUS。 */
  status: number
  /** Key 本身已被删除（软删除），但那笔账还在。 */
  removed: boolean
  /** 平台从他钱包里扣掉的额度。 */
  used_quota: number
  request_count: number
  /** 输入 + 输出 token 合计。 */
  total_tokens: number
  last_used_at: number
}

/**
 * 经销商的自助台账。
 *
 * 三个总数是他账户上的合计（与钱包页同一组数），`keys` 是这些消费按 Key
 * 拆开的样子；`markup_ratio` 是平台给他定的那档毛利（1.1 表示加价 10%），
 * 让他看得见自己的拿货价是按什么算出来的。
 */
export interface AgentLedger {
  quota: number
  used_quota: number
  request_count: number
  markup_ratio: string
  keys: AgentLedgerKey[]
}
