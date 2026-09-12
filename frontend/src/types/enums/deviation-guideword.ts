export const deviationGuidewords = ['no', 'more', 'less', 'reverse', 'other'] as const
export type DeviationGuideword = (typeof deviationGuidewords)[number]

export const deviationGuidewordLabels: Record<DeviationGuideword, string> = {
  no: '无 / NO',
  more: '过量 / MORE',
  less: '不足 / LESS',
  reverse: '逆向 / REVERSE',
  other: '其他 / OTHER',
}
