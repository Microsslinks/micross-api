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
// apply-ai-rewrites.mjs - task-13 phase 4-6: apply AI rewrite batches
//
// Reads rewrite JSON from .docs/task-13-native-i18n/rewrites/ and applies
// them to the locale JSON files. Each rewrite file is per-language and
// contains a flat key -> new_value map. Preserves key set (adds 0, removes 0).
//
// Usage:
//   node scripts/apply-ai-rewrites.mjs --dry-run
//   node scripts/apply-ai-rewrites.mjs --apply
//   node scripts/apply-ai-rewrites.mjs --lang=fr --apply
//
import fs from 'node:fs/promises'
import path from 'node:path'

const LOCALES_DIR = path.resolve('src/i18n/locales')
const REWRITES_DIR = path.resolve('../.docs/task-13-native-i18n/rewrites')

const LANGS = ['fr', 'ru', 'ja', 'vi']

async function loadLocale(locale) {
  const raw = await fs.readFile(path.join(LOCALES_DIR, `${locale}.json`), 'utf8')
  const parsed = JSON.parse(raw)
  return parsed.translation || parsed
}

function stableStringify(obj) {
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
  const rewritePath = opts.rewriteFile || path.join(REWRITES_DIR, `${lang}.json`)
  let rewrites
  try {
    rewrites = JSON.parse(await fs.readFile(rewritePath, 'utf8'))
  } catch (e) {
    console.error(`[${lang}] no rewrite file or parse error: ${e.message}`)
    return { lang, applied: 0, skipped: 0 }
  }

  let applied = 0
  let skipped = 0
  const changes = []

  for (const [key, newValue] of Object.entries(rewrites)) {
    if (typeof newValue !== 'string') {
      skipped++
      continue
    }
    if (!(key in data)) {
      console.warn(`[${lang}] WARN: key '${key}' not in locale file`)
      skipped++
      continue
    }
    const oldValue = data[key]
    if (oldValue === newValue) {
      skipped++
      continue
    }
    changes.push({ key, old: oldValue, neu: newValue })
    if (opts.apply) data[key] = newValue
    applied++
  }

  if (opts.apply && applied > 0) {
    await fs.writeFile(
      path.join(LOCALES_DIR, `${lang}.json`),
      stableStringify({ translation: data }) + '\n',
      'utf8',
    )
  }
  return { lang, applied, skipped, changes }
}

async function main() {
  const args = process.argv.slice(2)
  const apply = args.includes('--apply')
  const langArg = args.find((a) => a.startsWith('--lang='))
  const onlyLang = langArg ? langArg.split('=')[1] : null
  const fileArg = args.find((a) => a.startsWith('--rewrite-file='))
  const rewriteFile = fileArg ? path.resolve(fileArg.split('=')[1]) : null

  const targets = onlyLang ? [onlyLang] : LANGS

  await fs.mkdir(REWRITES_DIR, { recursive: true })

  const results = []
  for (const lang of targets) {
    const r = await processLang(lang, { apply, rewriteFile })
    results.push(r)
  }

  console.log(`[${apply ? 'APPLIED' : 'DRY-RUN'}] summary:`)
  let total = 0
  for (const r of results) {
    console.log(`  ${r.lang}: ${r.applied} applied, ${r.skipped} skipped`)
    total += r.applied
  }
  console.log(`  TOTAL: ${total}`)
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})