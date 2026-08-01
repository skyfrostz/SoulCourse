export function policyDocumentPath(province: string, documentId: string) {
  return `/knowledge/${encodeURIComponent(province)}/docs/${documentId}`
}
