export namespace auth {
	
	export class CLIDetectResult {
	    provider_type: string;
	    detected: boolean;
	    credential_path: string;
	    logged_in: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new CLIDetectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider_type = source["provider_type"];
	        this.detected = source["detected"];
	        this.credential_path = source["credential_path"];
	        this.logged_in = source["logged_in"];
	        this.error = source["error"];
	    }
	}

}

export namespace feedback {
	
	export class SystemInfo {
	    app_version: string;
	    go_version: string;
	    os: string;
	    arch: string;
	    build_time?: string;

	    static createFrom(source: any = {}) {
	        return new SystemInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_version = source["app_version"];
	        this.go_version = source["go_version"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.build_time = source["build_time"];
	    }
	}

}

export namespace entity {
	
	export class VersionNote {
	    version: string;
	    title: string;
	    features: string[];
	    fixes: string[];
	
	    static createFrom(source: any = {}) {
	        return new VersionNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.title = source["title"];
	        this.features = source["features"];
	        this.fixes = source["fixes"];
	    }
	}

}

export namespace main {
	
	export class AuthMethodDetectStatus {
	    method: string;
	    available: boolean;
	    connected: boolean;
	    tier: number;
	    provider_type?: string;
	    detail?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthMethodDetectStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.available = source["available"];
	        this.connected = source["connected"];
	        this.tier = source["tier"];
	        this.provider_type = source["provider_type"];
	        this.detail = source["detail"];
	        this.error = source["error"];
	    }
	}
	export class AuthDetectResult {
	    results: AuthMethodDetectStatus[];
	    recommended: string;
	    all_unavailable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuthDetectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], AuthMethodDetectStatus);
	        this.recommended = source["recommended"];
	        this.all_unavailable = source["all_unavailable"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class ConversationSummary {
	    id: string;
	    title: string;
	    updated_at: string;
	    deleted_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversationSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.updated_at = source["updated_at"];
	        this.deleted_at = source["deleted_at"];
	    }
	}
	export class DeviceFlowStartResponse {
	    user_code: string;
	    verification_uri: string;
	    device_code: string;
	    expires_in: number;
	    interval: number;
	    redirect_uri?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFlowStartResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.user_code = source["user_code"];
	        this.verification_uri = source["verification_uri"];
	        this.device_code = source["device_code"];
	        this.expires_in = source["expires_in"];
	        this.interval = source["interval"];
	        this.redirect_uri = source["redirect_uri"];
	    }
	}
	export class DeviceFlowStatusResponse {
	    device_code: string;
	    provider_type: string;
	    status: string;
	    error?: string;
	    provider_id?: string;
	    provider_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFlowStatusResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device_code = source["device_code"];
	        this.provider_type = source["provider_type"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.provider_id = source["provider_id"];
	        this.provider_name = source["provider_name"];
	    }
	}
	export class DisclaimerStatus {
	    required: boolean;
	    text: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new DisclaimerStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.required = source["required"];
	        this.text = source["text"];
	        this.version = source["version"];
	    }
	}
	export class DownloadUpdateRequest {
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadUpdateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	    }
	}
	export class EmbeddingStatusResponse {
	    available: boolean;
	    model_present: boolean;
	    engine_available: boolean;
	    runtime_lib_present: boolean;
	    runtime_lib_path: string;
	    failure_reason: string;
	    model_path: string;
	    model_name: string;
	    download_url: string;

	    static createFrom(source: any = {}) {
	        return new EmbeddingStatusResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.model_present = source["model_present"];
	        this.engine_available = source["engine_available"];
	        this.runtime_lib_present = source["runtime_lib_present"];
	        this.runtime_lib_path = source["runtime_lib_path"];
	        this.failure_reason = source["failure_reason"];
	        this.model_path = source["model_path"];
	        this.model_name = source["model_name"];
	        this.download_url = source["download_url"];
	    }
	}
	export class EmergencyResult {
	    level: string;
	    message: string;
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new EmergencyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.message = source["message"];
	        this.action = source["action"];
	    }
	}
	export class HealthResultResponse {
	    provider_id: string;
	    status: string;
	    latency_ms: number;
	    checked_at: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HealthResultResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider_id = source["provider_id"];
	        this.status = source["status"];
	        this.latency_ms = source["latency_ms"];
	        this.checked_at = source["checked_at"];
	        this.error = source["error"];
	    }
	}
	export class MessageResponse {
	    id: string;
	    role: string;
	    content: string;
	    timestamp: string;
	    prompt_tokens: number;
	    completion_tokens: number;
	    total_tokens: number;
	    confidence?: any;

	    static createFrom(source: any = {}) {
	        return new MessageResponse(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	        this.prompt_tokens = source["prompt_tokens"];
	        this.completion_tokens = source["completion_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.confidence = source["confidence_result"];
	    }
	}
	export class ModelInfo {
	    id: string;
	    name: string;
	    provider: string;
	
	    static createFrom(source: any = {}) {
	        return new ModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.provider = source["provider"];
	    }
	}
	export class OAuthDeviceFlowProviderInfo {
	    provider_type: string;
	    name: string;
	    available: boolean;
	    configured: boolean;
	    detail: string;
	
	    static createFrom(source: any = {}) {
	        return new OAuthDeviceFlowProviderInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider_type = source["provider_type"];
	        this.name = source["name"];
	        this.available = source["available"];
	        this.configured = source["configured"];
	        this.detail = source["detail"];
	    }
	}
	export class OllamaDetectResult {
	    installed: boolean;
	    running: boolean;
	    has_smollm2: boolean;
	    install_guide?: string;
	    server_starting?: boolean;
	    pull_progress?: string;
	
	    static createFrom(source: any = {}) {
	        return new OllamaDetectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.running = source["running"];
	        this.has_smollm2 = source["has_smollm2"];
	        this.install_guide = source["install_guide"];
	        this.server_starting = source["server_starting"];
	        this.pull_progress = source["pull_progress"];
	    }
	}
	export class SendMessageRequest {
	    conversation_id: string;
	    messages: models.Message[];
	    model: string;
	    provider_id: string;
	
	    static createFrom(source: any = {}) {
	        return new SendMessageRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.conversation_id = source["conversation_id"];
	        this.messages = this.convertValues(source["messages"], models.Message);
	        this.model = source["model"];
	        this.provider_id = source["provider_id"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SendMessageResponse {
	    reply: string;
	    confidence_result?: any;
	    warnings: string[];
	
	    static createFrom(source: any = {}) {
	        return new SendMessageResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reply = source["reply"];
	        this.confidence_result = source["confidence_result"];
	        this.warnings = source["warnings"];
	    }
	}
	export class TestAPIKeyResult {
	    valid: boolean;
	    message: string;
	    models?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TestAPIKeyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.message = source["message"];
	        this.models = source["models"];
	    }
	}
	export class UpdateInfoResponse {
	    version: string;
	    name: string;
	    body: string;
	    published_at: string;
	    mandatory: boolean;
	    channel: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfoResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.name = source["name"];
	        this.body = source["body"];
	        this.published_at = source["published_at"];
	        this.mandatory = source["mandatory"];
	        this.channel = source["channel"];
	    }
	}
	export class UpdateSettingsResponse {
	    check_enabled: boolean;
	    channel: string;
	    skip_version: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.check_enabled = source["check_enabled"];
	        this.channel = source["channel"];
	        this.skip_version = source["skip_version"];
	    }
	}
	export class MemoryItem {
	    fact_id: string;
	    subject: string;
	    predicate: string;
	    object: string;
	    confidence?: any;
	    status: string;
	    created_at: number;

	    static createFrom(source: any = {}) {
	        return new MemoryItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fact_id = source["fact_id"];
	        this.subject = source["subject"];
	        this.predicate = source["predicate"];
	        this.object = source["object"];
	        this.confidence = source["confidence_result"];
	        this.status = source["status"];
	        this.created_at = source["created_at"];
	    }
	}
	export class MemoryStats {
	    total: number;
	    approved: number;
	    rejected: number;
	    pending: number;

	    static createFrom(source: any = {}) {
	        return new MemoryStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.approved = source["approved"];
	        this.rejected = source["rejected"];
	        this.pending = source["pending"];
	    }
	}

}

export namespace models {
	
	export class AuthParams {
	    api_key?: string;
	    cli_credential_path?: string;
	    oauth_client_id?: string;
	    oauth_auth_url?: string;
	    oauth_token_url?: string;
	    oauth_expires_at?: number;
	    gcp_project_id?: string;
	    gcp_region?: string;
	    sa_json?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.cli_credential_path = source["cli_credential_path"];
	        this.oauth_client_id = source["oauth_client_id"];
	        this.oauth_auth_url = source["oauth_auth_url"];
	        this.oauth_token_url = source["oauth_token_url"];
	        this.oauth_expires_at = source["oauth_expires_at"];
	        this.gcp_project_id = source["gcp_project_id"];
	        this.gcp_region = source["gcp_region"];
	        this.sa_json = source["sa_json"];
	    }
	}
	export class Message {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class ProviderConfig {
	    id: string;
	    name: string;
	    apiHost: string;
	    apiKey?: string;
	    modelId: string;
	    temperature: number;
	    timeoutMs: number;
	    maxRetries: number;
    maxTokens: number;
	    group: string;
	    enabled: boolean;
	    sortOrder: number;
	    createdAt: number;
	    updatedAt: number;
	    auth_method: string;
	    auth_params: AuthParams;
	
	    static createFrom(source: any = {}) {
	        return new ProviderConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.apiHost = source["apiHost"];
	        this.apiKey = source["apiKey"];
	        this.modelId = source["modelId"];
	        this.temperature = source["temperature"];
	        this.timeoutMs = source["timeoutMs"];
	        this.maxRetries = source["maxRetries"];
        this.maxTokens = source["maxTokens"];
	        this.group = source["group"];
	        this.enabled = source["enabled"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.auth_method = source["auth_method"];
	        this.auth_params = this.convertValues(source["auth_params"], AuthParams);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}
