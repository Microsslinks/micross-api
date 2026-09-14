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
 * 模型名清单的分隔符：换行、中英文逗号、顿号。
 * 空格不算分隔符——模型名里允许有空格，按空格拆会对不上线路。
 */
const MODEL_LIST_SEPARATOR = /[\n\r,，、]/

/**
 * 把一份粘贴进来的文本拆成模型名：去空白、去重复、保留输入顺序
 * （运营是照着客户的单子一行行核对的，顺序乱了不好对）。
 */
export function splitModelList(raw: string): string[] {
  const seen = new Set<string>()
  const names: string[] = []
  for (const piece of raw.split(MODEL_LIST_SEPARATOR)) {
    const name = piece.trim()
    if (!name || seen.has(name)) {
      continue
    }
    seen.add(name)
    names.push(name)
  }
  return names
}

/**
 * 把新加的模型名并到已有清单后面：同名只留第一个（先来先占位），
 * 顺序不变。清单里的名字一律保留原样大小写——模型名区分大小写。
 */
export function mergeModelNames(base: string[], incoming: string[]): string[] {
  const merged = [...base]
  const seen = new Set(base)
  for (const raw of incoming) {
    const name = raw.trim()
    if (!name || seen.has(name)) {
      continue
    }
    seen.add(name)
    merged.push(name)
  }
  return merged
}
