import fs from 'fs'
import path from 'path'

// Guards against UTF-8 text re-saved as Windows-1252 (e.g. "—" -> "â€”"),
// which renders as garbage in the UI.
const MOJIBAKE = /â€|â†|â‰|Ã[\u0080-\u00BF]|Â[\u0080-\u00BF]/

function walk(dir: string, out: string[] = []): string[] {
  for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, e.name)
    if (e.isDirectory()) walk(p, out)
    else if (/\.(tsx?|jsx?)$/.test(e.name)) out.push(p)
  }
  return out
}

describe('source files contain no mojibake', () => {
  const files = ['app', 'components', 'lib', 'hooks']
    .map((d) => path.join(process.cwd(), d))
    .filter((d) => fs.existsSync(d))
    .flatMap((d) => walk(d))

  it.each(files)('%s', (file) => {
    const bad = fs
      .readFileSync(file, 'utf8')
      .split('\n')
      .map((l, i) => (MOJIBAKE.test(l) ? `${i + 1}: ${l.trim()}` : null))
      .filter(Boolean)
    expect(bad).toEqual([])
  })
})
