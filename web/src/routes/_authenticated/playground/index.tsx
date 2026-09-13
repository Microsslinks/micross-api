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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { Main } from '@/components/layout'
import { Playground } from '@/features/playground'
import { isSidebarModuleEnabled } from '@/lib/nav-modules'

export const Route = createFileRoute('/_authenticated/playground/')({
  beforeLoad: () => {
    // 注意：这里查的是 chat.playground，而侧边栏可见性看的是 playground 段，
    // 两者不对称且默认配置里没有 chat 段，所以这个守卫实际上不生效。本次只
    // 统一兜底落点，守卫本身的不对称另记（见 master-plan §「刻意未动的五处」）。
    if (!isSidebarModuleEnabled('chat', 'playground')) {
      throw redirect({ href: '/dashboard/overview' })
    }
  },
  component: PlaygroundPage,
})

function PlaygroundPage() {
  return (
    <Main className='p-0'>
      <Playground />
    </Main>
  )
}
