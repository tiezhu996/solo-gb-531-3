import { ref } from 'vue'
import { exportCoverageEvidencePack } from '../api/coverage-evaluation'
import { downloadJSON, evidencePackFilename } from '../utils/download'

/**
 * 证据包导出：读取服务端证据包并在浏览器端落盘。
 * 任何网络/业务错误都向上抛出，由页面给出明确的失败提示。
 */
export function useEvidenceExport() {
  const exporting = ref(false)

  async function exportEvidencePack(evaluationId: number): Promise<void> {
    exporting.value = true
    try {
      const pack = await exportCoverageEvidencePack(evaluationId)
      downloadJSON(evidencePackFilename(evaluationId), pack)
    } finally {
      exporting.value = false
    }
  }

  return { exporting, exportEvidencePack }
}
