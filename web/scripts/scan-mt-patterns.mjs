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
//
// scan-mt-patterns.mjs - task-13 phase 2: machine-translation pattern scan
//
// Scans fr/ru/ja/vi JSON files for six heuristic patterns that suggest
// machine-translation residue. Emits one markdown report per language to
// .docs/task-13-native-i18n/audit/02-mt-pattern-scan.<lang>.md
//
// Patterns:
//   length-anomaly      value vs en.json ratio > 1.5 or < 0.3
//   fullwidth-punct     ru/vi/ja values contain 「，」「。」「！」「？」「：」「；」
//   borrowed-english    value contains English technical loan-words
//   repeated-template   same key-template appears in >= 3 keys with divergent values
//   fr-punctuation      fr value where Latin letter directly precedes ".", ",", ";"
//   ja-kana-heavy       ja value >= 50% Hiragana/Katakana (sign of Google Translate output)
//
// Run from web/: node scripts/scan-mt-patterns.mjs
//
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const AUDIT_DIR = path.resolve('../.docs/task-13-native-i18n/audit')

const LANGS = ['fr', 'ru', 'ja', 'vi']

// Borrowed English technical terms (case-insensitive). Pattern uses word
// boundaries to avoid matching e.g. "cached" -> "cache".
const BORROWED_WORDS_RE = new RegExp(
  '\\b(' +
    [
      'upstream', 'channel', 'channels', 'cache', 'cached', 'caching',
      'token', 'tokens', 'webhook', 'webhooks', 'endpoint', 'endpoints',
      'api', 'sdk', 'cli', 'tls', 'ssl', 'oauth', 'jwt',
      'http', 'https', 'json', 'yaml', 'xml', 'css', 'html',
      'sql', 'prisma', 'drizzle', 'gorm', 'gin', 'cron',
      'sla', 'slo', 'sli', 'sso', 'mfa', '2fa', 'otp', 'totp',
      'hmac', 'rsa', 'sha', 'md5', 'base64', 'utf8', 'ascii',
      'merchant', 'stripe', 'waffo', 'creem', 'pancake',
      'admin', 'dashboard', 'token-limit', 'rate-limit',
    ].join('|') +
    ')\\b',
  'i',
)

// Fullwidth punctuation - ru/vi/ja should not use CJK punctuation
const FULLWIDTH_PUNCT_RE = /[，。；：？！]/

// French: a Latin letter directly followed by "." or "," or ";" or "!" or "?"
// should have a space (or be end-of-sentence). Without space = MT tell.
const FR_PUNCT_NO_SPACE_RE = /[a-zA-ZÀ-ÿÀ-ſ][\.,;!?:](?!\s|$)/u

// Japanese kana range
const KANA_RE = /[\u3040-\u309F\u30A0-\u30FF]/g

async function loadLocale(locale) {
  const raw = await fs.readFile(path.join(LOCALES_DIR, `${locale}.json`), 'utf8')
  const parsed = JSON.parse(raw)
  return parsed.translation || parsed
}

function detectPatterns(enVal, langVal, lang) {
  const flags = []
  const enLen = (enVal || '').length
  const langLen = (langVal || '').length

  if (enLen >= 10) {
    // Skip very short en values - length ratio meaningless
    const ratio = langLen / enLen
    if (ratio > 1.8) flags.push('length-too-long')
    else if (ratio < 0.2 && langLen >= 5) flags.push('length-too-short')
  }

  if (FULLWIDTH_PUNCT_RE.test(langVal)) {
    // ja legitimately uses CJK punctuation - skip
    if (lang === 'ja') {
      // ja 不报警
    } else if (lang === 'fr') {
      flags.push('fullwidth-punct-fr')
    } else {
      flags.push('fullwidth-punct')
    }
  }

  // borrowed-english only fires when English loan-words dominate the value
  // (>= 30% of chars). Single token like "{{token}}" with "token" is normal.
  if (BORROWED_WORDS_RE.test(langVal)) {
    const borrowedChars = (langVal.match(BORROWED_WORDS_RE) || []).join('').length
    if (langLen > 0 && borrowedChars / langLen >= 0.3) {
      flags.push('borrowed-english')
    }
  }

  if (lang === 'fr' && FR_PUNCT_NO_SPACE_RE.test(langVal)) {
    flags.push('fr-punct-no-space')
  }

  if (lang === 'ja' && langLen >= 20) {
    const kanaCount = (langVal.match(KANA_RE) || []).length
    // ja 中长句假名比例高通常意味着 MT 输出；纯汉字或纯片假名反而是规范术语
    if (kanaCount / langLen >= 0.85) {
      flags.push('ja-kana-heavy')
    }
  }

  // Russian: very long adjectives chain (4+ consecutive adjectives)
  if (lang === 'ru' && langLen >= 30) {
    const longAdjChain =
      /(?:\b[а-яА-ЯёЁ]+(?:ый|ая|ое|ые|ий|яя|ее)\b[\s,]+){4,}/.test(langVal)
    if (longAdjChain) flags.push('ru-long-adj-chain')
  }

  return flags
}

function findRepeatedTemplates(entries) {
  // Detect template divergence: same key prefix with {var} placeholders that
  // get translated inconsistently across >=3 occurrences.
  const templateCounts = new Map() // template-pattern -> [{key, value}]
  for (const [key, value] of entries) {
    // extract template: replace {{varName}} with {VAR}
    const tmpl = key.replace(/\{\{[^}]+\}\}/g, '{VAR}')
    if (!templateCounts.has(tmpl)) templateCounts.set(tmpl, [])
    templateCounts.get(tmpl).push({ key, value })
  }
  const divergent = []
  for (const [tmpl, list] of templateCounts) {
    if (list.length >= 3) {
      const distinct = new Set(list.map((x) => x.value))
      if (distinct.size >= 3) {
        divergent.push({ tmpl, samples: list.slice(0, 5) })
      }
    }
  }
  return divergent
}

async function main() {
  await fs.mkdir(AUDIT_DIR, { recursive: true })

  const en = await loadLocale('en')

  for (const lang of LANGS) {
    console.log(`scanning ${lang}...`)
    const data = await loadLocale(lang)

    const entries = Object.entries(data)
    const enEntries = Object.entries(en)

    // length comparison needs parallel iteration
    const flagged = []
    const untranslatedCount = { value: 0 }

    for (const [key, value] of entries) {
      if (typeof value !== 'string') continue

      const enVal = en[key] || ''
      if (value === enVal && enVal !== '') {
        untranslatedCount.value++
        continue // skip untranslated keys - covered by 01-untranslated-leaks.md
      }

      const flags = detectPatterns(enVal, value, lang)
      if (flags.length > 0) {
        flagged.push({ key, value, flags, enLen: enVal.length, langLen: value.length })
      }
    }

    // template divergence scan
    const divergent = findRepeatedTemplates(entries)

    // render markdown
    const lines = []
    lines.push(`# 02 · 机翻特征扫描 · ${lang.toUpperCase()}（task-13 phase 2）`)
    lines.push('')
    lines.push('> 创建：2026-09-17')
    lines.push(`> 数据来源：\`web/src/i18n/locales/${lang}.json\` vs \`en.json\``)
    lines.push(`> 工具：\`web/scripts/scan-mt-patterns.mjs\``)
    lines.push('')
    lines.push('## 摘要')
    lines.push('')
    lines.push(`| 指标 | 值 |`)
    lines.push(`|---|---|`)
    lines.push(`| 总键数 | ${entries.length} |`)
    lines.push(`| 完全未翻译（与 en 相同）| ${untranslatedCount.value} |`)
    lines.push(`| 至少命中一条机翻特征 | ${flagged.length} |`)
    lines.push(`| 句式分叉数 | ${divergent.length} |`)
    lines.push('')

    // grouped by flag type
    const byFlag = new Map()
    for (const f of flagged) {
      for (const flag of f.flags) {
        if (!byFlag.has(flag)) byFlag.set(flag, [])
        byFlag.get(flag).push(f)
      }
    }

    lines.push('## 规则命中统计')
    lines.push('')
    lines.push('| 规则 | 命中数 | 说明 |')
    lines.push('|---|---|---|')
    const RULE_DESC = {
      'length-too-long': 'value 长度 > en 原文 1.5×（典型：过度展开）',
      'length-too-short': 'value 长度 < en 原文 0.3×（典型：翻译过度精简）',
      'fullwidth-punct': 'ru/vi/ja 用了 CJK 全角标点「，」「。」「！」「？」「：」「；」',
      'fullwidth-punct-fr': '法语里混入 CJK 全角标点',
      'borrowed-english': 'value 保留了英文技术借词（upstream / cache / token 等）',
      'fr-punct-no-space': '法语标点前/后缺空格（MT 典型问题）',
      'ja-kana-heavy': '日文假名比例 ≥ 50%（疑似 Google Translate 输出）',
      'ru-long-adj-chain': '俄语连续 3+ 个 -ый/-ая/-ое 形容词堆叠',
    }
    for (const [flag, list] of [...byFlag.entries()].sort((a, b) => b[1].length - a[1].length)) {
      lines.push(`| \`${flag}\` | ${list.length} | ${RULE_DESC[flag] || ''} |`)
    }
    lines.push('')

    // detailed listing per flag
    lines.push('## 详细命中清单（每规则前 30）')
    lines.push('')
    for (const [flag, list] of [...byFlag.entries()].sort((a, b) => b[1].length - a[1].length)) {
      lines.push(`### ${flag}（${list.length} 处，展示前 30）`)
      lines.push('')
      lines.push('| key | en | ' + lang + ' |')
      lines.push('|---|---|---|')
      for (const f of list.slice(0, 30)) {
        const enV = (en[f.key] || '').replace(/\|/g, '\\|').replace(/\n/g, ' ')
        const langV = f.value.replace(/\|/g, '\\|').replace(/\n/g, ' ')
        lines.push(`| \`${f.key}\` | ${enV.slice(0, 80)} | ${langV.slice(0, 80)} |`)
      }
      if (list.length > 30) {
        lines.push(`| ... | （剩余 ${list.length - 30} 处略）| |`)
      }
      lines.push('')
    }

    // divergent templates
    if (divergent.length > 0) {
      lines.push('## 句式分叉（同 key template，value 分裂）')
      lines.push('')
      lines.push('说明：模板形如 `Channel {{name}}` 出现 ≥ 3 次，但 4 语言 value 不一致 → 提示人工对齐。')
      lines.push('')
      for (const d of divergent.slice(0, 30)) {
        lines.push(`### \`${d.tmpl}\``)
        lines.push('')
        for (const s of d.samples) {
          lines.push(`- \`${s.key}\` → ${lang}: \`${s.value}\``)
        }
        lines.push('')
      }
    }

    const outPath = path.join(AUDIT_DIR, `02-mt-pattern-scan.${lang}.md`)
    await fs.writeFile(outPath, lines.join('\n') + '\n', 'utf8')
    console.log(`wrote ${outPath}`)
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})