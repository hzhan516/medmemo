import { models } from '@wails/go/models'
import type { ProviderConfig, AuthParams } from '@/types/provider'

/**
 * Wails 生成的 ProviderConfig 缺少 models 字段声明。
 * 运行时 JSON 序列化仍能正确传递，这里用 interface merge 补全类型。
 */
export interface WailsProviderConfig extends models.ProviderConfig {
  models?: Array<{ id: string; name: string; enabled: boolean }>
}

/**
 * 将前端 ProviderConfig 转换为 Wails 后端模型。
 */
export function toWailsProviderConfig(config: ProviderConfig): WailsProviderConfig {
  return {
    id: config.id,
    name: config.name,
    apiHost: config.apiHost,
    apiKey: config.apiKey,
    modelId: config.modelId,
    models: config.models,
    temperature: config.temperature,
    timeoutMs: config.timeoutMs,
    maxRetries: config.maxRetries,
    group: config.group,
    enabled: config.enabled,
    sortOrder: config.sortOrder,
    createdAt: config.createdAt,
    updatedAt: config.updatedAt,
    auth_method: config.authMethod,
    auth_params: toWailsAuthParams(config.authParams),
  } as WailsProviderConfig
}

/**
 * 将 Wails 后端模型转换为前端 ProviderConfig。
 */
export function fromWailsProviderConfig(wails: models.ProviderConfig): ProviderConfig {
  const ext = wails as WailsProviderConfig
  return {
    id: wails.id,
    templateId: extractTemplateId(wails.id),
    name: wails.name,
    apiHost: wails.apiHost,
    apiKey: wails.apiKey || '',
    modelId: wails.modelId,
    models: ext.models,
    temperature: wails.temperature,
    timeoutMs: wails.timeoutMs,
    maxRetries: wails.maxRetries,
    group: wails.group,
    enabled: wails.enabled,
    sortOrder: wails.sortOrder,
    createdAt: wails.createdAt,
    updatedAt: wails.updatedAt,
    authMethod: wails.auth_method as ProviderConfig['authMethod'],
    authParams: fromWailsAuthParams(wails.auth_params),
  }
}

function toWailsAuthParams(params: AuthParams): models.AuthParams {
  return {
    api_key: params.apiKey,
    cli_credential_path: params.cliCredentialPath,
    oauth_client_id: params.oauthClientId,
    oauth_auth_url: params.oauthAuthUrl,
    oauth_token_url: params.oauthTokenUrl,
    oauth_expires_at: params.oauthExpiresAt,
    gcp_project_id: params.gcpProjectId,
    gcp_region: params.gcpRegion,
    sa_json: params.saJson,
  }
}

function fromWailsAuthParams(params: models.AuthParams): AuthParams {
  return {
    apiKey: params.api_key,
    cliCredentialPath: params.cli_credential_path,
    oauthClientId: params.oauth_client_id,
    oauthAuthUrl: params.oauth_auth_url,
    oauthTokenUrl: params.oauth_token_url,
    oauthExpiresAt: params.oauth_expires_at,
    gcpProjectId: params.gcp_project_id,
    gcpRegion: params.gcp_region,
    saJson: params.sa_json,
  }
}

/**
 * 从 provider id 中提取 templateId。
 * 当前 id 生成规则: `${templateId}_${Date.now()}_${random}`。
 * 自定义 provider 通常为 custom_...，无下划线时返回原 id。
 */
function extractTemplateId(id: string): string {
  const idx = id.indexOf('_')
  return idx > 0 ? id.slice(0, idx) : id
}
