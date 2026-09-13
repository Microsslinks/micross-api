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

export interface ApiResponse<T = unknown> {
  success: boolean
  message?: string
  data?: T
}

/**
 * 我此刻挂在谁名下、按什么价算。
 *
 * parent_agent_id 为 0 表示直属平台（没有归属经销商）；plan_id 为 0 表示此刻没有
 * 生效的折扣方案——两种零值都是正常状态，不是没加载出来。
 */
export interface CustomerBinding {
  parent_agent_id: number
  agent_name: string
  plan_id: number
  plan_name: string
  /** 方案基准折扣（个别模型可能另有专门折扣） */
  plan_discount: string
  /** 折扣的来源：manual / subscription / customer_code 之一 */
  binding_source: string
}
