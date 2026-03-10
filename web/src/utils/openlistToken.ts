/**
 * OpenList Token 工具模块
 *
 * 提供打开 OpenList popup 窗口和自动解析 token 的功能，
 * 简化 115 Open 登录流程，避免用户手动复制粘贴两个字段。
 */

/** OpenList 115 页面地址 */
export const OPENLIST_TOKEN_URL = 'https://api.oplist.org/'

/** 解析结果 */
export interface OpenListTokenResult {
  access_token: string
  refresh_token: string
}

/**
 * 打开 OpenList 页面到居中的 popup 窗口。
 */
export function openOpenListPopup(): Window | null {
  const width = 700
  const height = 700
  const left = Math.round((screen.width - width) / 2)
  const top = Math.round((screen.height - height) / 2)
  return window.open(
    OPENLIST_TOKEN_URL,
    'openlist_token',
    `width=${width},height=${height},left=${left},top=${top},resizable=yes,scrollbars=yes`
  )
}

/**
 * 解析用户粘贴的 OpenList token 数据。
 *
 * 支持以下输入格式：
 * 1. 完整 URL（含 hash fragment）：`https://api.oplist.org/#eyJhY2Nlc3Nf...`
 * 2. 仅 hash fragment：`#eyJhY2Nlc3Nf...`
 * 3. 纯 base64 字符串：`eyJhY2Nlc3Nf...`
 * 4. 纯 JSON 字符串：`{"access_token": "...", "refresh_token": "..."}`
 *
 * @throws Error 如果输入不合法或缺少必要字段
 */
export function parseOpenListTokenData(input: string): OpenListTokenResult {
  const trimmed = input.trim()
  if (!trimmed) {
    throw new Error('输入内容为空')
  }

  // 尝试从 URL hash 中提取 base64
  let base64Data: string | null = null

  // 格式 1: 完整 URL with hash
  const hashIndex = trimmed.indexOf('#')
  if (hashIndex !== -1) {
    base64Data = trimmed.substring(hashIndex + 1).trim()
  }

  // 格式 4: 直接是 JSON
  if (!base64Data && trimmed.startsWith('{')) {
    try {
      return extractTokensFromJSON(trimmed)
    } catch {
      throw new Error('JSON 格式无效，请检查是否包含 access_token 和 refresh_token')
    }
  }

  // 格式 3: 纯 base64 (no hash prefix)
  if (!base64Data) {
    base64Data = trimmed
  }

  // 解码 base64
  let decoded: string
  try {
    decoded = atob(base64Data)
  } catch {
    throw new Error('Base64 解码失败，请确认粘贴的内容完整。可以粘贴完整的 URL 或页面上显示的 base64 字符串。')
  }

  try {
    return extractTokensFromJSON(decoded)
  } catch {
    throw new Error('解码后的数据格式不正确，缺少 access_token 或 refresh_token')
  }
}

function extractTokensFromJSON(jsonStr: string): OpenListTokenResult {
  const data = JSON.parse(jsonStr) as Record<string, unknown>
  const accessToken = typeof data.access_token === 'string' ? data.access_token.trim() : ''
  const refreshToken = typeof data.refresh_token === 'string' ? data.refresh_token.trim() : ''

  if (!accessToken || !refreshToken) {
    throw new Error('缺少 access_token 或 refresh_token')
  }

  return { access_token: accessToken, refresh_token: refreshToken }
}
