export function shouldReportHighTTFT(
  p99Ms: number | null | undefined,
  thresholdMs: number | null | undefined,
  sampleCount: number | null | undefined
): boolean {
  return p99Ms != null && thresholdMs != null && (sampleCount ?? 0) > 0 && p99Ms >= thresholdMs
}
