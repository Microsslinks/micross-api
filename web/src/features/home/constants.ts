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
 * Home page constants
 * All hardcoded data for home page sections
 */
import type { TFunction } from 'i18next'

// Layout - Main base classes
export const MAIN_BASE_CLASSES = 'bg-background text-foreground w-full'

/**
 * Vertical rhythm shared by the three content sections (S3 promotions /
 * S4 how-it-works / S5 cta). The page-level gap is two of these added
 * together, so the whole page is retuned by changing this one string.
 * See `.docs/frontend/layout-system.md` §五 垂直节奏.
 */
export const SECTION_PY = 'py-12 md:py-14 lg:py-16'

// Hero section - AI Applications (Left side)
export const AI_APPLICATIONS = [
  'LobeHub.Color',
  'Dify.Color',
  'OpenWebUI',
  'Cline',
] as const

// Hero section - AI Models (Right side)
export const AI_MODELS = [
  'Qwen.Color',
  'DeepSeek.Color',
  'Doubao.Color',
  'OpenAI',
  'Claude.Color',
  'Gemini.Color',
] as const

// Hero section - Gateway Features
export const GATEWAY_FEATURES = [
  'Cost Tracking',
  'Model Access',
  'Guardrails',
  'Observability',
  'Budgets',
  'Load Balancing',
  'Rate Limiting',
  'Token Mgmt',
  'Prompt Caching',
  'Pass-Through',
] as const

// Stats section - Default statistics
export const DEFAULT_STATS = [
  {
    value: '33',
    suffix: '+',
    description: 'upstream token factories',
  },
  {
    value: '120',
    suffix: '+',
    description: 'large models at your service',
  },
  {
    value: '20',
    suffix: '+',
    description: 'API routing rules, all compatible',
  },
  {
    value: '100',
    suffix: '+',
    description: 'skills to power your AI journey',
  },
] as const

// Promotions section - 3 + 3 热卖模型（标题、说明、折扣与牌价均为 i18n key 或展示字面量）
export type PromoBrand =
  | 'anthropic'
  | 'openai'
  | 'gemini'
  | 'kimi'
  | 'deepseek'
  | 'glm'

export interface PromoModelPlan {
  id: string
  input: string
  output: string
  cache: string
}

export interface PromoModel {
  brand: PromoBrand
  name: string
  maker: string
  promo: string
  plans: PromoModelPlan[]
}

export interface PromoGroup {
  id: string
  label: string
  models: PromoModel[]
}

export const PROMO_GROUPS: PromoGroup[] = [
  {
    id: 'global',
    label: 'Global models',
    models: [
      {
        brand: 'anthropic',
        name: 'Claude',
        maker: 'Anthropic · The ceiling for code',
        promo: '60% off',
        plans: [
          { id: 'claude-fable-5', input: '$10', output: '$50', cache: '$1' },
          { id: 'claude-opus-5', input: '$5', output: '$25', cache: '$0.5' },
          { id: 'claude-sonnet-5', input: '$2', output: '$10', cache: '$0.2' },
        ],
      },
      {
        brand: 'openai',
        name: 'GPT',
        maker: 'OpenAI · Where AI began',
        promo: '65% off',
        plans: [
          { id: 'gpt-5.6-sol', input: '$5', output: '$30', cache: '$0.5' },
          { id: 'gpt-5.6-terra', input: '$2.5', output: '$15', cache: '$0.25' },
          { id: 'gpt-5.6-luna', input: '$1', output: '$6', cache: '$0.1' },
        ],
      },
      {
        brand: 'gemini',
        name: 'Gemini',
        maker: 'Google · Multimodal powerhouse',
        promo: '70% off',
        plans: [
          { id: 'gemini-3.1-pro', input: '$2', output: '$12', cache: '$0.2' },
          {
            id: 'gemini-3.6-flash',
            input: '$1.5',
            output: '$7.5',
            cache: '$0.15',
          },
          {
            id: 'gemini-3.5-flash-lite',
            input: '$0.3',
            output: '$2.5',
            cache: '$0.03',
          },
        ],
      },
    ],
  },
  {
    id: 'china',
    label: 'China models',
    models: [
      {
        brand: 'kimi',
        name: 'Kimi K3',
        maker: 'Moonshot AI · Domestic front-runner',
        promo: '30% off',
        plans: [
          { id: 'kimi-k3', input: '¥20', output: '¥100', cache: '¥2' },
          { id: 'kimi-k2.7-code', input: '¥6.5', output: '¥27', cache: '¥1.3' },
          { id: 'kimi-k2.6', input: '¥6.5', output: '¥27', cache: '¥1.1' },
        ],
      },
      {
        brand: 'deepseek',
        name: 'DeepSeek',
        maker: 'DeepSeek · The price cutter',
        promo: '32% off',
        plans: [
          {
            id: 'deepseek-v4-pro',
            input: '¥3.1',
            output: '¥6.2',
            cache: '¥0.03',
          },
          {
            id: 'deepseek-v4-flash',
            input: '¥1',
            output: '¥2',
            cache: '¥0.02',
          },
          { id: 'deepseek-v3.2-pro', input: '¥2', output: '¥3', cache: '¥0.2' },
        ],
      },
      {
        brand: 'glm',
        name: 'GLM',
        maker: 'Zhipu AI · Built by Tsinghua',
        promo: '34% off',
        plans: [
          { id: 'glm-5.3', input: '¥8', output: '¥28', cache: '¥2' },
          { id: 'glm-5.2', input: '¥8', output: '¥28', cache: '¥2' },
          {
            id: 'glm-5.3-flash',
            input: '¥0.4',
            output: '¥0.8',
            cache: '¥0.115',
          },
        ],
      },
    ],
  },
]

export function getGatewayFeatures(t: TFunction) {
  return GATEWAY_FEATURES.map((feature) => t(feature))
}

export function getDefaultStats(t: TFunction) {
  return DEFAULT_STATS.map((stat) => ({
    ...stat,
    description: stat.description ? t(stat.description) : undefined,
  }))
}

// 智能体生态 - 15 个可直连本平台的 AI 智能体（featured 的 6 个在 Hero 展示，其余在弹窗里）
export type AgentId =
  | 'claude-code'
  | 'codex'
  | 'cursor'
  | 'openclaw'
  | 'hermes'
  | 'n8n'
  | 'trae'
  | 'codebuddy'
  | 'cherry-studio'
  | 'coze'
  | 'dify'
  | 'doubao'
  | 'yuanbao'
  | 'qwen'
  | 'workbuddy'

export interface AgentItem {
  id: AgentId
  name: string
  url: string
  group: 'global' | 'china'
  desc: string
  featured?: boolean
}

export const AGENTS: AgentItem[] = [
  // 国外 - 编程智能体三巨头
  {
    id: 'claude-code',
    name: 'Claude Code',
    url: 'https://claude.com/claude-code',
    group: 'global',
    desc: 'The Anthropic veteran that lives in your terminal.',
    featured: true,
  },
  {
    id: 'codex',
    name: 'Codex',
    url: 'https://openai.com/codex/',
    group: 'global',
    desc: 'OpenAI official CLI: one command and it ships.',
    featured: true,
  },
  {
    id: 'cursor',
    name: 'Cursor',
    url: 'https://cursor.com',
    group: 'global',
    desc: 'The editor writes the code, you sip the coffee.',
    featured: true,
  },
  {
    id: 'openclaw',
    name: 'OpenClaw',
    url: 'https://openclaw.ai',
    group: 'global',
    desc: 'Never asks for a raise, never clocks off.',
  },
  {
    id: 'hermes',
    name: 'Hermes',
    url: 'https://hermes-agent.nousresearch.com',
    group: 'global',
    desc: 'The open-source courier: hand it work and it runs.',
  },
  {
    id: 'n8n',
    name: 'n8n',
    url: 'https://n8n.io',
    group: 'global',
    desc: 'Open-source automation: wire the nodes, it works.',
  },
  // 国内 - 客户端与 IDE
  {
    id: 'codebuddy',
    name: 'CodeBuddy',
    url: 'https://copilot.tencent.com',
    group: 'china',
    desc: 'Tencent Cloud coding agent: plugin, IDE and CLI.',
    featured: true,
  },
  {
    id: 'trae',
    name: 'Trae',
    url: 'https://www.trae.cn',
    group: 'china',
    desc: 'ByteDance AI IDE, built for Chinese developers.',
    featured: true,
  },
  {
    id: 'cherry-studio',
    name: 'Cherry Studio',
    url: 'https://cherry-ai.com',
    group: 'china',
    desc: 'The sweet little desktop AI client.',
    featured: true,
  },
  {
    id: 'coze',
    name: '扣子 Coze',
    url: 'https://www.coze.cn',
    group: 'china',
    desc: 'Drag, drop, launch a bot. Zero threshold.',
  },
  {
    id: 'dify',
    name: 'Dify',
    url: 'https://dify.ai',
    group: 'china',
    desc: 'Drag and drop, and your LLM app goes live.',
  },
  {
    id: 'doubao',
    name: '豆包 Doubao',
    url: 'https://www.doubao.com',
    group: 'china',
    desc: 'ByteDance everyday assistant: anyone can start.',
  },
  {
    id: 'yuanbao',
    name: '元宝 Yuanbao',
    url: 'https://yuanbao.tencent.com',
    group: 'china',
    desc: 'Tencent Yuanbao, the AI helper inside WeChat.',
  },
  {
    id: 'qwen',
    name: '通义千问 Qwen',
    url: 'https://tongyi.aliyun.com',
    group: 'china',
    desc: 'Alibaba Qwen: long-form Chinese is its home turf.',
  },
  {
    id: 'workbuddy',
    name: 'WorkBuddy',
    url: 'https://www.workbuddy.cn',
    group: 'china',
    desc: 'Tencent desktop AI assistant: chores, delegated.',
  },
]
