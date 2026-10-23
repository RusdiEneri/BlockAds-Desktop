export namespace dnsengine {
	
	export class Handler {
	
	
	    static createFrom(source: any = {}) {
	        return new Handler(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace filtermanager {
	
	export class UpdateResult {
	    subscription_id: string;
	    status_code: number;
	    status: string;
	    checksum: string;
	    rule_count: number;
	    warnings: number;
	    duration: number;
	    bytes_received: number;
	    error_message?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subscription_id = source["subscription_id"];
	        this.status_code = source["status_code"];
	        this.status = source["status"];
	        this.checksum = source["checksum"];
	        this.rule_count = source["rule_count"];
	        this.warnings = source["warnings"];
	        this.duration = source["duration"];
	        this.bytes_received = source["bytes_received"];
	        this.error_message = source["error_message"];
	    }
	}

}

export namespace filters {
	
	export class Matcher {
	
	
	    static createFrom(source: any = {}) {
	        return new Matcher(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

export namespace models {
	
	export class AppInfo {
	    name: string;
	    version: string;
	    platform: string;
	    milestone: string;
	    goVersion: string;
	    webView2Version: string;
	    dataDir: string;
	    isPortable: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.platform = source["platform"];
	        this.milestone = source["milestone"];
	        this.goVersion = source["goVersion"];
	        this.webView2Version = source["webView2Version"];
	        this.dataDir = source["dataDir"];
	        this.isPortable = source["isPortable"];
	    }
	}
	export class AppSettings {
	    upstreamDohUrl: string;
	    bootstrapDns: string;
	    cacheEnabled: boolean;
	    cacheSize: number;
	    autoStart: boolean;
	    systemDnsIntegrated: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.upstreamDohUrl = source["upstreamDohUrl"];
	        this.bootstrapDns = source["bootstrapDns"];
	        this.cacheEnabled = source["cacheEnabled"];
	        this.cacheSize = source["cacheSize"];
	        this.autoStart = source["autoStart"];
	        this.systemDnsIntegrated = source["systemDnsIntegrated"];
	    }
	}
	export class CatalogFilterItem {
	    id: string;
	    name: string;
	    description: string;
	    category: string;
	    sourceUrl: string;
	    format: string;
	    homepage: string;
	    maintainer: string;
	    license: string;
	    isSubscribed: boolean;
	    subscribedId?: string;
	
	    static createFrom(source: any = {}) {
	        return new CatalogFilterItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.sourceUrl = source["sourceUrl"];
	        this.format = source["format"];
	        this.homepage = source["homepage"];
	        this.maintainer = source["maintainer"];
	        this.license = source["license"];
	        this.isSubscribed = source["isSubscribed"];
	        this.subscribedId = source["subscribedId"];
	    }
	}
	export class CustomRule {
	    id: string;
	    domain: string;
	    action: string;
	    enabled: boolean;
	    createdAt: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new CustomRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.domain = source["domain"];
	        this.action = source["action"];
	        this.enabled = source["enabled"];
	        this.createdAt = source["createdAt"];
	        this.notes = source["notes"];
	    }
	}
	export class DiagnosticCheck {
	    name: string;
	    category: string;
	    status: string;
	    details: string;
	
	    static createFrom(source: any = {}) {
	        return new DiagnosticCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.category = source["category"];
	        this.status = source["status"];
	        this.details = source["details"];
	    }
	}
	export class FilterList {
	    id: string;
	    name: string;
	    sourceUrl: string;
	    enabled: boolean;
	    format: string;
	    ruleCount: number;
	    lastUpdated: string;
	    lastAttemptAt: string;
	    lastSuccessAt: string;
	    lastStatus: string;
	    lastError: string;
	    contentChecksum: string;
	    etag: string;
	    description: string;
	    category: string;
	
	    static createFrom(source: any = {}) {
	        return new FilterList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.sourceUrl = source["sourceUrl"];
	        this.enabled = source["enabled"];
	        this.format = source["format"];
	        this.ruleCount = source["ruleCount"];
	        this.lastUpdated = source["lastUpdated"];
	        this.lastAttemptAt = source["lastAttemptAt"];
	        this.lastSuccessAt = source["lastSuccessAt"];
	        this.lastStatus = source["lastStatus"];
	        this.lastError = source["lastError"];
	        this.contentChecksum = source["contentChecksum"];
	        this.etag = source["etag"];
	        this.description = source["description"];
	        this.category = source["category"];
	    }
	}
	export class FilterPreviewResult {
	    sourceUrl: string;
	    normalizedUrl: string;
	    detectedName: string;
	    description: string;
	    homepage: string;
	    format: string;
	    ruleCount: number;
	    isAlreadySubscribed: boolean;
	    existingSubscription?: string;
	    existingId?: string;
	
	    static createFrom(source: any = {}) {
	        return new FilterPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceUrl = source["sourceUrl"];
	        this.normalizedUrl = source["normalizedUrl"];
	        this.detectedName = source["detectedName"];
	        this.description = source["description"];
	        this.homepage = source["homepage"];
	        this.format = source["format"];
	        this.ruleCount = source["ruleCount"];
	        this.isAlreadySubscribed = source["isAlreadySubscribed"];
	        this.existingSubscription = source["existingSubscription"];
	        this.existingId = source["existingId"];
	    }
	}
	export class ProtectionStatus {
	    isActive: boolean;
	    state: string;
	    upstreamUrl: string;
	    listenAddr: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new ProtectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isActive = source["isActive"];
	        this.state = source["state"];
	        this.upstreamUrl = source["upstreamUrl"];
	        this.listenAddr = source["listenAddr"];
	        this.message = source["message"];
	    }
	}
	export class QueryLogEntry {
	    id: string;
	    timestamp: string;
	    domain: string;
	    clientIp: string;
	    queryType: string;
	    status: string;
	    upstreamLatencyMs: number;
	    filterRule: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new QueryLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.domain = source["domain"];
	        this.clientIp = source["clientIp"];
	        this.queryType = source["queryType"];
	        this.status = source["status"];
	        this.upstreamLatencyMs = source["upstreamLatencyMs"];
	        this.filterRule = source["filterRule"];
	        this.reason = source["reason"];
	    }
	}
	export class SafeSearchProviderSetting {
	    id: string;
	    name: string;
	    domains: string[];
	    target: string;
	    docUrl: string;
	    enabled: boolean;
	    status: string;
	    warning?: string;
	    updatedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new SafeSearchProviderSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.domains = source["domains"];
	        this.target = source["target"];
	        this.docUrl = source["docUrl"];
	        this.enabled = source["enabled"];
	        this.status = source["status"];
	        this.warning = source["warning"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class StartupStatus {
	    supported: boolean;
	    enabled: boolean;
	    executablePath: string;
	    isPathValid: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new StartupStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.enabled = source["enabled"];
	        this.executablePath = source["executablePath"];
	        this.isPathValid = source["isPathValid"];
	        this.message = source["message"];
	    }
	}
	export class TelemetryStats {
	    queriesPerSec: number;
	    peakQueriesPerSec: number;
	    cacheHitRate: number;
	    cachedEntries: number;
	    medianLatencyMs: number;
	    blockedPercent: number;
	    totalAnswers: number;
	    totalBlocked: number;
	    resolverLoad: number;
	    memoryResidentMB: number;
	    uptimeSeconds: number;
	    engineVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new TelemetryStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.queriesPerSec = source["queriesPerSec"];
	        this.peakQueriesPerSec = source["peakQueriesPerSec"];
	        this.cacheHitRate = source["cacheHitRate"];
	        this.cachedEntries = source["cachedEntries"];
	        this.medianLatencyMs = source["medianLatencyMs"];
	        this.blockedPercent = source["blockedPercent"];
	        this.totalAnswers = source["totalAnswers"];
	        this.totalBlocked = source["totalBlocked"];
	        this.resolverLoad = source["resolverLoad"];
	        this.memoryResidentMB = source["memoryResidentMB"];
	        this.uptimeSeconds = source["uptimeSeconds"];
	        this.engineVersion = source["engineVersion"];
	    }
	}

}

export namespace rewrites {
	
	export class Rewriter {
	
	
	    static createFrom(source: any = {}) {
	        return new Rewriter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Rule {
	    id: string;
	    domain: string;
	    type: string;
	    target: string;
	    enabled: boolean;
	    ttl: number;
	
	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.domain = source["domain"];
	        this.type = source["type"];
	        this.target = source["target"];
	        this.enabled = source["enabled"];
	        this.ttl = source["ttl"];
	    }
	}

}

export namespace storage {
	
	export class AsyncLogger {
	
	
	    static createFrom(source: any = {}) {
	        return new AsyncLogger(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}
	export class Storage {
	
	
	    static createFrom(source: any = {}) {
	        return new Storage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

