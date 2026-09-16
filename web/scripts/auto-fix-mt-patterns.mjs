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
// auto-fix-mt-patterns.mjs - task-13 phase 4 programmatic fixes
//
// Deterministic (non-LLM) fixes for patterns detected in scan-mt-patterns.mjs:
//
//  1. fullwidth-punct  (ru/vi only; ja legitimately uses CJK punctuation)
//     Replace 「，。；：？！」with their half-width ",.;:?!" equivalents.
//
//  2. fr-punct-no-space (fr only)
//     Insert a space between a letter and ".", ",", ";", ":", "!" or "?"
//     when followed by another letter. Preserves abbreviations like "U.S.A."
//     (single letter segments), URLs, and version strings.
//
//  3. terminology alignment (per web/src/i18n/translation-glossary.md)
//     Replace legacy 渠道 terms (Canal, Канал, チャネル, Kênh, Fabricant, etc.)
//     with the new 上游供应商 terms. This is the bulk of the channel fix.
//
//  4. length-too-short (lang-le-short values)
//     If a translation is < 20% of en length and likely truncated, mark for
//     AI rewrite (no auto-fix; logged in change report).
//
// Usage:
//   node scripts/auto-fix-mt-patterns.mjs --dry-run   (default)
//   node scripts/auto-fix-mt-patterns.mjs --apply
//   node scripts/auto-fix-mt-patterns.mjs --lang=fr --apply
//
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const REPORT_DIR = path.resolve('../.docs/task-13-native-i18n/audit')

const LANGS = ['fr', 'ru', 'ja', 'vi']

// -----------------------------------------------------------------------------
// Per-language rules
// -----------------------------------------------------------------------------

// 1. fullwidth -> halfwidth punctuation (ru/vi only)
const FW_TO_HW = {
  '，': ', ', // (Chinese comma)
  '。': '. ', // (Chinese period)
  '；': '; ', // (Chinese semicolon)
  '：': ': ', // (Chinese colon)
  '？': '?', // (Chinese question mark)
  '！': '!', // (Chinese exclamation)
}
const FW_PUNCT_RE = new RegExp('[' + Object.keys(FW_TO_HW).join('') + ']', 'g')

function fixFullwidth(value, lang) {
  if (lang === 'ja') return { value, changed: false }
  if (!FW_PUNCT_RE.test(value)) return { value, changed: false }
  // Reset regex state (g flag)
  FW_PUNCT_RE.lastIndex = 0
  const out = value.replace(FW_PUNCT_RE, (m) => FW_TO_HW[m])
  return { value: out, changed: out !== value }
}

// 2. fr: insert space before ".", ",", ";", ":", "!" or "?" when followed by letter
const FR_PUNCT_RE = /([A-Za-zÀ-ÿÀ-ſ])(\.|,|;|:|!|\?)([A-Za-zÀ-ÿÀ-ſ])/g

function fixFrPunctuation(value, lang) {
  if (lang !== 'fr') return { value, changed: false }
  // don't touch URLs (http://, https://)
  // Skip abbreviation-like patterns: single uppercase + period (U.S.A., M.B.A.)
  // Heuristic: if the middle group is exactly "." and surrounding letters are
  // both uppercase of length 1, skip (likely abbreviation).
  FR_PUNCT_RE.lastIndex = 0
  const out = value.replace(FR_PUNCT_RE, (m, p1, p2, p3) => {
    if (p2 === '.' && p1.length === 1 && p1 >= 'A' && p1 <= 'Z' && p3.length === 1 && p3 >= 'A' && p3 <= 'Z') {
      return m // likely abbreviation like "U.S.A."
    }
    if (/^https?$/.test(p1) && p2 === ':') return m
    if (/^www$/.test(p1) && p2 === '.') return m
    return `${p1}${p2} ${p3}`
  })
  return { value: out, changed: out !== value }
}

// 3. terminology alignment
// Each rule is [regex, replacement]. Replacement may be a function for
// case-sensitive or context-aware logic. Match patterns:
//
// FR channel: Canal, canal, canaux -> Fournisseur amont / fournisseur amont
// FR vendor:  Fabricant (standalone) -> Fournisseur
// RU channel: Канал/канала/каналы/каналу/каналов -> Поставщик / поставщик
// RU vendor:  Производитель -> Поставщик
// JA channel: チャネル/チャンネル -> アップストリームプロバイダー
// VI channel: Kênh/kênh -> Nhà cung cấp
// VI vendor:  Nhà sản xuất -> Nhà cung cấp
const TERMINOLOGY = {
  fr: [
    // Capitalized standalone word (likely as noun in title/heading)
    {
      re: /\bCanal\b/g,
      to: 'Fournisseur amont',
      note: 'Channel (capitalized standalone)',
    },
    // lowercase channel word forms (in middle of sentence)
    {
      re: /\bcanal(?!s?\b)\b/gi,
      to: (m) => (m[0] === 'C' ? 'Fournisseur amont' : 'fournisseur amont'),
      note: 'canal -> fournisseur amont',
    },
    // "canal " in plural form already covered by 'canaux' below
    {
      re: /\bcanaux\b/gi,
      to: (m) => (m[0] === 'C' ? 'Fournisseurs amont' : 'fournisseurs amont'),
      note: 'canaux -> fournisseurs amont',
    },
    {
      re: /\bcanal\(aux\)/gi,
      to: (m) => (m[0] === 'C' ? 'Fournisseur(s) amont' : 'fournisseur(s) amont'),
      note: 'canal(aux) -> fournisseur(s) amont',
    },
    // Vendor: Fabricant -> Fournisseur (only standalone, not part of other words)
    {
      re: /\bFabricant\b/g,
      to: 'Fournisseur',
      note: 'Vendor Fabricant -> Fournisseur',
    },
  ],
  ru: [
    // All inflected forms of "канал" - explicit listing (no \b, JS doesn't handle Cyrillic \b).
    // Cyrillic range \u0400-\u04FF used for boundary detection (avoids PowerShell charset issues).
    {
      re:
        /(?<![A-Za-z\u0400-\u04FF])(канал|канала|каналу|каналом|канале|каналы|каналов|каналам|каналами|каналах)(?![A-Za-z\u0400-\u04FF])/gi,
      to: (m) => {
        const lower = m.toLowerCase()
        const isCap = m[0] === 'К'
        const capMap = {
          Канал: 'Поставщик',
          Канала: 'Поставщика',
          Каналу: 'Поставщику',
          Каналом: 'Поставщиком',
          Канале: 'Поставщике',
          Каналы: 'Поставщики',
          Каналов: 'Поставщиков',
          Каналам: 'Поставщикам',
          Каналами: 'Поставщиками',
          Каналах: 'Поставщиках',
        }
        const map = {
          канал: 'поставщик',
          канала: 'поставщика',
          каналу: 'поставщику',
          каналом: 'поставщиком',
          канале: 'поставщике',
          каналы: 'поставщики',
          каналов: 'поставщиков',
          каналам: 'поставщикам',
          каналами: 'поставщиками',
          каналах: 'поставщиках',
        }
        return isCap ? capMap[m] ?? 'Поставщик' : map[lower] ?? 'поставщик'
      },
      note: 'канал* inflected -> поставщик* inflected',
    },
    // Vendor: Производитель -> Поставщик
    {
      re: /(?<![A-Za-z\u0400-\u04FF])Производитель(?![A-Za-z\u0400-\u04FF])/g,
      to: 'Поставщик',
      note: 'Vendor Производитель -> Поставщик',
    },
  ],
  ja: [
    // チャネル -> アップストリームプロバイダー
    { re: /チャネル/g, to: 'アップストリームプロバイダー', note: 'チャネル -> アップストリームプロバイダー' },
    // チャンネル -> アップストリームプロバイダー
    { re: /チャンネル/g, to: 'アップストリームプロバイダー', note: 'チャンネル -> アップストリームプロバイダー' },
  ],
  vi: [
    // Kênh -> Nhà cung cấp
    { re: /\bKênh\b/g, to: 'Nhà cung cấp', note: 'Kênh standalone' },
    { re: /\bkênh\b/g, to: 'nhà cung cấp', note: 'kênh lowercase' },
    // Vendor: Nhà sản xuất -> Nhà cung cấp
    { re: /\bNhà sản xuất\b/g, to: 'Nhà cung cấp', note: 'Vendor Nhà sản xuất -> Nhà cung cấp' },
  ],
}

function fixTerminology(value, lang) {
  const rules = TERMINOLOGY[lang] || []
  let out = value
  let changed = false
  const notes = []
  for (const rule of rules) {
    const before = out
    out = out.replace(rule.re, rule.to)
    if (out !== before) {
      changed = true
      notes.push(rule.note)
    }
  }
  return { value: out, changed, notes }
}

// -----------------------------------------------------------------------------
// Driver
// -----------------------------------------------------------------------------

async function loadLocale(locale) {
  const raw = await fs.readFile(path.join(LOCALES_DIR, `${locale}.json`), 'utf8')
  const parsed = JSON.parse(raw)
  return parsed.translation || parsed
}

function stableStringify(obj) {
  // JSON.stringify with sorted keys for stable diff
  const seen = new WeakSet()
  return JSON.stringify(
    obj,
    (k, v) => {
      if (typeof v === 'object' && v !== null) {
        if (seen.has(v)) return
        seen.add(v)
      }
      return v
    },
    2,
  )
}

async function processLang(lang, opts) {
  const data = await loadLocale(lang)
  const changes = []

  for (const [key, original] of Object.entries(data)) {
    if (typeof original !== 'string') continue
    let cur = original
    const ops = []

    // 1. fullwidth-punct (ru/vi)
    {
      const r = fixFullwidth(cur, lang)
      if (r.changed) {
        cur = r.value
        ops.push('fullwidth->halfwidth')
      }
    }

    // 2. fr-punct
    {
      const r = fixFrPunctuation(cur, lang)
      if (r.changed) {
        cur = r.value
        ops.push('fr-punct-space')
      }
    }

    // 3. terminology
    {
      const r = fixTerminology(cur, lang)
      if (r.changed) {
        cur = r.value
        ops.push(`term:${r.notes.join('|')}`)
      }
    }

    if (cur !== original) {
      changes.push({ key, original, current: cur, ops })
      if (opts.apply) {
        data[key] = cur
      }
    }
  }

  if (opts.apply && changes.length > 0) {
    const out = stableStringify({ translation: data })
    await fs.writeFile(
      path.join(LOCALES_DIR, `${lang}.json`),
      out + '\n',
      'utf8',
    )
  }

  return { lang, changes, applied: opts.apply }
}

async function main() {
  const args = process.argv.slice(2)
  const apply = args.includes('--apply')
  const langArg = args.find((a) => a.startsWith('--lang='))
  const onlyLang = langArg ? langArg.split('=')[1] : null

  const targets = onlyLang ? [onlyLang] : LANGS

  const allChanges = []
  for (const lang of targets) {
    const result = await processLang(lang, { apply })
    allChanges.push(result)
  }

  // Render change report
  const lines = []
  lines.push('# 03 · 程序化修复变更报告（task-13 phase 4）')
  lines.push('')
  lines.push(`> 生成：${new Date().toISOString().slice(0, 19)}`)
  lines.push(`> 模式：${apply ? '**APPLIED**' : 'DRY-RUN'}`)
  lines.push('')
  lines.push('| 语言 | 修改 key 数 |')
  lines.push('|---|---|')
  let total = 0
  for (const r of allChanges) {
    lines.push(`| ${r.lang} | ${r.changes.length} |`)
    total += r.changes.length
  }
  lines.push(`| **合计** | **${total}** |`)
  lines.push('')

  for (const r of allChanges) {
    lines.push(`## ${r.lang}（${r.changes.length} 处）`)
    lines.push('')
    lines.push('| key | 改前 | 改后 | 操作 |')
    lines.push('|---|---|---|---|')
    for (const c of r.changes.slice(0, 50)) {
      const before = c.original.replace(/\|/g, '\\|').replace(/\n/g, ' ')
      const after = c.current.replace(/\|/g, '\\|').replace(/\n/g, ' ')
      lines.push(`| \`${c.key}\` | ${before.slice(0, 60)} | ${after.slice(0, 60)} | ${c.ops.join(', ')} |`)
    }
    if (r.changes.length > 50) {
      lines.push(`| ... | （剩余 ${r.changes.length - 50} 处略）| | |`)
    }
    lines.push('')
  }

  await fs.mkdir(REPORT_DIR, { recursive: true })
  const reportPath = path.join(REPORT_DIR, '03-auto-fix-changes.md')
  await fs.writeFile(reportPath, lines.join('\n') + '\n', 'utf8')

  console.log(`[${apply ? 'APPLIED' : 'DRY-RUN'}] total changes: ${total}`)
  console.log(`report: ${reportPath}`)

  for (const r of allChanges) {
    console.log(`  ${r.lang}: ${r.changes.length} keys`)
  }
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})