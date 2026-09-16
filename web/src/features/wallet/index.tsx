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
import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'

import { SectionPageLayout } from '@/components/layout'
import { useStatus } from '@/hooks/use-status'
import { AffiliateRewardsCard } from '@/features/earnings/components/affiliate-rewards-card'
import { CommissionRecordsTable } from '@/features/earnings/components/commission-records-table'
import { TransferDialog } from '@/features/earnings/components/dialogs/transfer-dialog'
import { useAffiliate, useCommission } from '@/features/earnings/hooks'
import { getSelf } from '@/lib/api'
import { ROLE } from '@/lib/roles'

import { AmountDiscountManageDialog } from './components/dialogs/amount-discount-manage-dialog'
import { AmountOptionsManageDialog } from './components/dialogs/amount-options-manage-dialog'
import { BillingHistoryDialog } from './components/dialogs/billing-history-dialog'
import { CreemConfirmDialog } from './components/dialogs/creem-confirm-dialog'
import { PaymentConfirmDialog } from './components/dialogs/payment-confirm-dialog'
import { CheckinAdminCard } from './components/checkin-admin-card'
import { RechargeFormCard } from './components/recharge-form-card'
import { WalletStatsCard } from './components/wallet-stats-card'
import { DEFAULT_DISCOUNT_RATE, PAYMENT_TYPES } from './constants'
import {
  useTopupInfo,
  usePayment,
  useRedemption,
  useCreemPayment,
  useWaffoPayment,
  useWaffoPancakePayment,
} from './hooks'
import {
  getDefaultPaymentType,
  getMinTopupAmount,
  dispatchSelectedPayment,
} from './lib'
import type {
  UserWalletData,
  PaymentMethod,
  PresetAmount,
  CreemProduct,
  WaffoPayMethod,
} from './types'

interface WalletProps {
  initialShowHistory?: boolean
}

/**
 * 「钱包」页
 *
 * 只装**钱本身**：余额、充值、兑换码、支付方式、账单历史，
 * 以及页面底部的「推荐收益」（推广佣金：邀请链接、待结算、累计、
 * 邀请数、转出到余额——原先的独立 /earnings 页，因为内容只有
 * 一行推广链接，不值得单独占一个入口）。
 * 最下面还有管理员可见的「签到奖励」配置卡——签到发的就是钱包里的
 * 每日额度，原先在业务管理 → 计费与定价的分节，同样不值得单开入口。
 * 「卖给谁」在 `/plans`（套餐）。
 */
export function Wallet(props: WalletProps) {
  const { t } = useTranslation()
  const [user, setUser] = useState<UserWalletData | null>(null)
  const [userLoading, setUserLoading] = useState(true)
  const [topupAmount, setTopupAmount] = useState(0)
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null)
  const [selectedPaymentMethod, setSelectedPaymentMethod] =
    useState<PaymentMethod>()
  const [selectedWaffoMethodIndex, setSelectedWaffoMethodIndex] = useState<
    number | null
  >(null)
  const [paymentLoading, setPaymentLoading] = useState<string | null>(null)
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false)
  const [billingDialogOpen, setBillingDialogOpen] = useState(false)
  const [redemptionCode, setRedemptionCode] = useState('')
  const [creemDialogOpen, setCreemDialogOpen] = useState(false)
  const [selectedCreemProduct, setSelectedCreemProduct] =
    useState<CreemProduct | null>(null)
  const [amountOptionsDialogOpen, setAmountOptionsDialogOpen] = useState(false)
  const [discountManageDialogOpen, setDiscountManageDialogOpen] =
    useState(false)
  const [transferDialogOpen, setTransferDialogOpen] = useState(false)

  const { status } = useStatus()
  const {
    topupInfo,
    presetAmounts,
    loading: topupLoading,
    refetch: refetchTopupInfo,
  } = useTopupInfo()

  const {
    affiliateLink,
    loading: affiliateLoading,
    transferQuota,
    transferring,
  } = useAffiliate()

  const {
    records: commissionRecords,
    loading: commissionLoading,
    page: commissionPage,
    total: commissionTotal,
    pageSize: commissionPageSize,
    setPage: setCommissionPage,
  } = useCommission(20)

  // 管理员才在快捷充值区域装配「添加充值金额 / 折扣管理」入口
  const isAdminUser = !!user && (user.role ?? 0) >= ROLE.ADMIN

  const {
    amount: paymentAmount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
  } = usePayment()
  const { redeeming, redeemCode } = useRedemption()
  const { processing: creemProcessing, processCreemPayment } = useCreemPayment()
  const { processing: waffoProcessing, processWaffoPayment } = useWaffoPayment()
  const { processing: pancakeProcessing, processWaffoPancakePayment } =
    useWaffoPancakePayment()

  // Fetch and refresh user data
  const fetchUser = useCallback(async () => {
    try {
      setUserLoading(true)
      const response = await getSelf()
      if (response.success && response.data) {
        setUser(response.data as UserWalletData)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch user data:', error)
    } finally {
      setUserLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  useEffect(() => {
    if (props.initialShowHistory) {
      setBillingDialogOpen(true)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [props.initialShowHistory])

  // Initialize topup amount when topup info is loaded
  const topupAmountInitializedRef = useRef(false)
  useEffect(() => {
    if (topupInfo && !topupAmountInitializedRef.current) {
      topupAmountInitializedRef.current = true
      const minTopup = getMinTopupAmount(topupInfo)
      setTopupAmount(minTopup)

      // Calculate initial payment amount with default payment type
      const defaultPaymentType = getDefaultPaymentType(topupInfo)
      calculatePaymentAmount(minTopup, defaultPaymentType)
    }
  }, [topupInfo, calculatePaymentAmount])

  // Get current payment type (selected or default)
  const getCurrentPaymentType = useCallback(() => {
    return selectedPaymentMethod?.type || getDefaultPaymentType(topupInfo)
  }, [selectedPaymentMethod, topupInfo])

  // Handle preset selection
  const handleSelectPreset = (preset: PresetAmount) => {
    setTopupAmount(preset.value)
    setSelectedPreset(preset.value)
    calculatePaymentAmount(preset.value, getCurrentPaymentType())
  }

  // Handle topup amount change
  const handleTopupAmountChange = (amount: number) => {
    setTopupAmount(amount)
    setSelectedPreset(null)
    calculatePaymentAmount(amount, getCurrentPaymentType())
  }

  // Handle payment method selection
  const handlePaymentMethodSelect = async (method: PaymentMethod) => {
    setSelectedPaymentMethod(method)
    setSelectedWaffoMethodIndex(null)
    setPaymentLoading(method.type)

    try {
      // Validate minimum topup
      const minTopup = getMinTopupAmount(topupInfo)
      if (topupAmount < minTopup) {
        return
      }

      // Calculate payment amount and show confirmation dialog
      await calculatePaymentAmount(topupAmount, method.type)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  // Handle payment confirmation
  const handlePaymentConfirm = async () => {
    if (!selectedPaymentMethod) return

    const success = await dispatchSelectedPayment(
      selectedPaymentMethod,
      topupAmount,
      selectedWaffoMethodIndex,
      {
        regular: processPayment,
        waffo: processWaffoPayment,
        waffoPancake: processWaffoPancakePayment,
      }
    )

    if (success) {
      setConfirmDialogOpen(false)
      await fetchUser()
    }
  }

  // Handle redemption
  const handleRedeem = async () => {
    if (!redemptionCode) return

    const success = await redeemCode(redemptionCode)
    if (success) {
      setRedemptionCode('')
      await fetchUser()
    }
  }

  // Handle Creem product selection
  const handleCreemProductSelect = (product: CreemProduct) => {
    setSelectedCreemProduct(product)
    setCreemDialogOpen(true)
  }

  // Handle Creem payment confirmation
  const handleCreemConfirm = async () => {
    if (!selectedCreemProduct) return

    const success = await processCreemPayment(selectedCreemProduct.productId)
    if (success) {
      setCreemDialogOpen(false)
      setSelectedCreemProduct(null)
      await fetchUser()
    }
  }

  // Handle referral rewards transfer to balance
  const handleAffiliateTransfer = async (amount: number) => {
    const success = await transferQuota(amount)
    if (success) {
      await fetchUser()
    }
    return success
  }

  const handleWaffoMethodSelect = async (
    method: WaffoPayMethod,
    index: number
  ) => {
    const loadingKey = `waffo-${index}`
    setSelectedPaymentMethod({
      name: method.name,
      type: PAYMENT_TYPES.WAFFO,
      icon: method.icon,
    })
    setSelectedWaffoMethodIndex(index)
    setPaymentLoading(loadingKey)

    try {
      await calculatePaymentAmount(topupAmount, PAYMENT_TYPES.WAFFO)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  // Get discount rate for current topup amount
  const getDiscountRate = useCallback(() => {
    return topupInfo?.discount?.[topupAmount] || DEFAULT_DISCOUNT_RATE
  }, [topupInfo, topupAmount])

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Wallet')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='flex w-full flex-col gap-4'>
            <WalletStatsCard user={user} loading={userLoading} />

            <div id='wallet-add-funds' className='scroll-mt-4'>
              <RechargeFormCard
                topupInfo={topupInfo}
                presetAmounts={presetAmounts}
                selectedPreset={selectedPreset}
                onSelectPreset={handleSelectPreset}
                topupAmount={topupAmount}
                onTopupAmountChange={handleTopupAmountChange}
                paymentAmount={paymentAmount}
                calculating={calculating}
                onPaymentMethodSelect={handlePaymentMethodSelect}
                paymentLoading={paymentLoading}
                redemptionCode={redemptionCode}
                onRedemptionCodeChange={setRedemptionCode}
                onRedeem={handleRedeem}
                redeeming={redeeming}
                topupLink={topupInfo?.topup_link}
                loading={topupLoading}
                priceRatio={(status?.price as number) || 1}
                onOpenBilling={() => setBillingDialogOpen(true)}
                creemProducts={topupInfo?.creem_products}
                enableCreemTopup={topupInfo?.enable_creem_topup}
                onCreemProductSelect={handleCreemProductSelect}
                enableWaffoTopup={topupInfo?.enable_waffo_topup}
                waffoPayMethods={topupInfo?.waffo_pay_methods}
                waffoMinTopup={topupInfo?.waffo_min_topup}
                onWaffoMethodSelect={handleWaffoMethodSelect}
                enableWaffoPancakeTopup={topupInfo?.enable_waffo_pancake_topup}
                isAdmin={isAdminUser}
                onOpenAmountOptions={() => setAmountOptionsDialogOpen(true)}
                onOpenDiscountManage={() => setDiscountManageDialogOpen(true)}
              />
            </div>

            <AffiliateRewardsCard
              user={user}
              affiliateLink={affiliateLink}
              onTransfer={() => setTransferDialogOpen(true)}
              complianceConfirmed={
                topupInfo?.payment_compliance_confirmed !== false
              }
              loading={affiliateLoading || userLoading}
            />

            <CommissionRecordsTable
              records={commissionRecords}
              page={commissionPage}
              total={commissionTotal}
              pageSize={commissionPageSize}
              loading={commissionLoading}
              onPageChange={setCommissionPage}
            />

            {isAdminUser && <CheckinAdminCard />}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <PaymentConfirmDialog
        open={confirmDialogOpen}
        onOpenChange={setConfirmDialogOpen}
        onConfirm={handlePaymentConfirm}
        topupAmount={topupAmount}
        paymentAmount={paymentAmount}
        paymentMethod={selectedPaymentMethod}
        calculating={calculating}
        processing={processing || waffoProcessing || pancakeProcessing}
        discountRate={getDiscountRate()}
      />

      <BillingHistoryDialog
        open={billingDialogOpen}
        onOpenChange={setBillingDialogOpen}
      />

      <TransferDialog
        open={transferDialogOpen}
        onOpenChange={setTransferDialogOpen}
        onConfirm={handleAffiliateTransfer}
        availableQuota={user?.aff_quota ?? 0}
        transferring={transferring}
      />

      <CreemConfirmDialog
        open={creemDialogOpen}
        onOpenChange={setCreemDialogOpen}
        onConfirm={handleCreemConfirm}
        product={selectedCreemProduct}
        processing={creemProcessing}
      />

      {isAdminUser && (
        <>
          <AmountOptionsManageDialog
            open={amountOptionsDialogOpen}
            onOpenChange={setAmountOptionsDialogOpen}
            onSaved={refetchTopupInfo}
          />
          <AmountDiscountManageDialog
            open={discountManageDialogOpen}
            onOpenChange={setDiscountManageDialogOpen}
            onSaved={refetchTopupInfo}
          />
        </>
      )}
    </>
  )
}
