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

/**
 * 客户号：经销商发出去的兑换凭证。
 *
 * 一码同时写着两件事——归属哪个经销商、按哪套折扣方案计价——客户绑一下，
 * 这两件事就在他的账号上生效。`plan_id` 为 0 表示这张号只落归属、不带折扣。
 */
export interface CustomerCode {
  id: number
  code: string
  agent_id: number
  plan_id: number
  /** 0 表示不限次数 */
  max_uses: number
  used_count: number
  /** 0 表示不过期 */
  expired_at: number
  /** 1 可用 / 0 已作废 */
  status: number
  remark: string
  created_at: number
  updated_at: number
}

/** 签一批客户号要带上的几件事；`count` 是唯一必填项。 */
export interface CustomerCodeIssuePayload {
  count: number
  /** 0 或不传 = 不绑折扣，只落归属 */
  plan_id?: number
  /** 0 或不传 = 不限次数 */
  max_uses?: number
  /** 0 或不传 = 不过期 */
  expired_at?: number
  remark?: string
}

/**
 * 「我的客户」里的一行：挂在当前经销商名下的一位客户此刻的台账。
 *
 * `plan_id` / `plan_name` / `plan_discount` 是这位客户此刻真正生效的那套价
 * （来源见 `binding_source`），`priced_by_me` 表示这套价是经销商自己定的——
 * 界面上据此决定「撤销我的定价」这个选项出不出现。
 */
export interface AgentCustomer {
  user_id: number
  username: string
  display_name: string
  /** 取值同 USER_STATUS：1 启用 / 2 禁用，-1 已删除。 */
  status: number
  created_at: number
  quota: number
  used_quota: number
  /** 0 表示没有生效方案（按官方标价） */
  plan_id: number
  plan_name: string
  plan_discount: string
  /** manual / agent / subscription / customer_code / migration / default，空表示没有生效方案 */
  binding_source: string
  priced_by_me: boolean
}

/** 货架上的一个折扣方案：经销商挑给客户用哪套价。 */
export interface DealerPlanOption {
  id: number
  name: string
  base_discount: string
  billing_mode: string
  remark: string
}

/** 经销商的货架，以及他自己的零售折扣下限（"0" 表示平台不限）。 */
export interface DealerPlanList {
  min_discount: string
  items: DealerPlanOption[]
}
