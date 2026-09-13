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
/**
 * 客户号的唯一用法就是这条链接：客户点开注册，号才会落到他账号上（归属与价格一起）。
 *
 * 与推广码 `?aff=` 是同一套做法（见 features/earnings/lib/affiliate.ts）：站点取当前访问的
 * 域名，不写死官方域名——换域名、走测试环境都不用改代码。链接里的参数名 `customer_code`
 * 必须与注册页读的那个一致（routes/__root.tsx 与注册表单各抓一次），
 * 两边一旦走散，经销商拷给客户的就是一条不会带上客户号的链接。
 */
export function buildCustomerInviteLink(code: string): string {
  if (typeof window === 'undefined') return ''
  return `${window.location.origin}/sign-up?customer_code=${encodeURIComponent(code)}`
}
