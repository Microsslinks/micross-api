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
// ============================================================================
// 兼容重导出（task-11 UI 改名）
//
// 原 affiliate-rewards-card 的实现已迁移到 commission-rewards-card.tsx。
// 这里只做别名重导出，保证上游 merge 时不会冲突：
//   - wallet/index.tsx 等老调用方仍可
//     `import { AffiliateRewardsCard } from './affiliate-rewards-card'`，
//     实际指向 CommissionRewardsCard（commission 数据源、删 transfer 按钮）。
//   - 新代码请直接 import CommissionRewardsCard：
//     `import { CommissionRewardsCard } from './commission-rewards-card'`。
//
// 后续 Phase（task-list 标记）会彻底删除 affiliate-rewards-card.tsx，
// 同时把 wallet/index.tsx 的 import 改为 commission-rewards-card 路径。
// ============================================================================

export { CommissionRewardsCard as AffiliateRewardsCard } from './commission-rewards-card'