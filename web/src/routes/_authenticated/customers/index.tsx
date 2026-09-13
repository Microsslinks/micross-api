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

import { DealerCustomers } from '@/features/dealer/customers'
import { CUSTOMER_TYPE } from '@/features/users/constants'
import { useAuthStore } from '@/stores/auth-store'

export const Route = createFileRoute('/_authenticated/customers/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()

    // 「我的客户」只给经销商看。菜单里本来就不会出现，这里挡住的是手敲地址进来的：
    // 普通客户没有下属，看到这张表只会是空的，也容易误会自己有客户。
    if (auth.user?.subject_type !== CUSTOMER_TYPE.AGENT) {
      throw redirect({ to: '/403' })
    }
  },
  component: DealerCustomers,
})
