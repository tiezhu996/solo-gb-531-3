/**
 * 将已在前端获取到的 JSON 证据保存为本地文件。
 * 后端统一返回响应信封，错误已经由 api/client 抛出，因此这里只负责落盘。
 */
export function downloadJSON(filename: string, content: unknown): void {
  const json = JSON.stringify(content, null, 2)
  const blob = new Blob([json], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  try {
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = filename
    document.body.appendChild(anchor)
    anchor.click()
    anchor.remove()
  } finally {
    URL.revokeObjectURL(url)
  }
}

export function evidencePackFilename(evaluationId: number): string {
  return `coverage-evidence-pack-${evaluationId}.json`
}
