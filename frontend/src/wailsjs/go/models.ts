export namespace automation {

	export class ScriptSource {
	    type: string;
	    uri: string;
	    ref: string;
	    path: string;
	    importedAt: string;

	    static createFrom(source: any = {}) {
	        return new ScriptSource(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.uri = source["uri"];
	        this.ref = source["ref"];
	        this.path = source["path"];
	        this.importedAt = source["importedAt"];
	    }
	}
	export class ScriptTargetSelector {
	    code: string;
	    profileId: string;
	    profileName: string;
	    groupId: string;
	    keywords: string[];
	    tags: string[];

	    static createFrom(source: any = {}) {
	        return new ScriptTargetSelector(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.groupId = source["groupId"];
	        this.keywords = source["keywords"];
	        this.tags = source["tags"];
	    }
	}
	export class ScriptTargetConfig {
	    mode: string;
	    selector: ScriptTargetSelector;
	    templateSelector: ScriptTargetSelector;
	    createNameTemplate: string;

	    static createFrom(source: any = {}) {
	        return new ScriptTargetConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.selector = this.convertValues(source["selector"], ScriptTargetSelector);
	        this.templateSelector = this.convertValues(source["templateSelector"], ScriptTargetSelector);
	        this.createNameTemplate = source["createNameTemplate"];
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
	export class ScriptRecord {
	    packageFormat: string;
	    manifestVersion: number;
	    id: string;
	    name: string;
	    description: string;
	    type: string;
	    status: string;
	    entryFile: string;
	    tags: string[];
	    selectorText: string;
	    paramsText: string;
	    scriptText: string;
	    notes: string;
	    targetConfig: ScriptTargetConfig;
	    source: ScriptSource;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new ScriptRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.packageFormat = source["packageFormat"];
	        this.manifestVersion = source["manifestVersion"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.status = source["status"];
	        this.entryFile = source["entryFile"];
	        this.tags = source["tags"];
	        this.selectorText = source["selectorText"];
	        this.paramsText = source["paramsText"];
	        this.scriptText = source["scriptText"];
	        this.notes = source["notes"];
	        this.targetConfig = this.convertValues(source["targetConfig"], ScriptTargetConfig);
	        this.source = this.convertValues(source["source"], ScriptSource);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class ScriptRunRecord {
	    id: string;
	    scriptId: string;
	    scriptName: string;
	    scriptType: string;
	    status: string;
	    summary: string;
	    error: string;
	    resultText: string;
	    startedAt: string;
	    finishedAt: string;
	    durationMs: number;

	    static createFrom(source: any = {}) {
	        return new ScriptRunRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.scriptId = source["scriptId"];
	        this.scriptName = source["scriptName"];
	        this.scriptType = source["scriptType"];
	        this.status = source["status"];
	        this.summary = source["summary"];
	        this.error = source["error"];
	        this.resultText = source["resultText"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.durationMs = source["durationMs"];
	    }
	}
	export class ScriptRunRequest {
	    scriptId: string;
	    selectorText: string;
	    paramsText: string;
	    useScriptSelector: boolean;
	    useScriptParams: boolean;
	    timeoutMs?: number;

	    static createFrom(source: any = {}) {
	        return new ScriptRunRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.scriptId = source["scriptId"];
	        this.selectorText = source["selectorText"];
	        this.paramsText = source["paramsText"];
	        this.useScriptSelector = source["useScriptSelector"];
	        this.useScriptParams = source["useScriptParams"];
	        this.timeoutMs = source["timeoutMs"];
	    }
	}



}

export namespace backend {

	export class BookmarkSyncResult {
	    total: number;
	    synced: number;
	    skipped: number;
	    failed: number;
	    skippedList: string[];
	    failedList: string[];

	    static createFrom(source: any = {}) {
	        return new BookmarkSyncResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.synced = source["synced"];
	        this.skipped = source["skipped"];
	        this.failed = source["failed"];
	        this.skippedList = source["skippedList"];
	        this.failedList = source["failedList"];
	    }
	}
	export class CookieInfo {
	    name: string;
	    value: string;
	    domain: string;
	    path: string;
	    expires: number;
	    httpOnly: boolean;
	    secure: boolean;
	    sameSite: string;

	    static createFrom(source: any = {}) {
	        return new CookieInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	        this.domain = source["domain"];
	        this.path = source["path"];
	        this.expires = source["expires"];
	        this.httpOnly = source["httpOnly"];
	        this.secure = source["secure"];
	        this.sameSite = source["sameSite"];
	    }
	}
	export class LicenseStatus {
	    maxLimit: number;
	    usedCount: number;
	    usedKeys: string[];

	    static createFrom(source: any = {}) {
	        return new LicenseStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxLimit = source["maxLimit"];
	        this.usedCount = source["usedCount"];
	        this.usedKeys = source["usedKeys"];
	    }
	}
	export class ProxyIPHealthResult {
	    proxyId: string;
	    ok: boolean;
	    source: string;
	    error: string;
	    ip: string;
	    fraudScore: number;
	    isResidential: boolean;
	    isBroadcast: boolean;
	    country: string;
	    region: string;
	    city: string;
	    asOrganization: string;
	    rawData: Record<string, any>;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new ProxyIPHealthResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyId = source["proxyId"];
	        this.ok = source["ok"];
	        this.source = source["source"];
	        this.error = source["error"];
	        this.ip = source["ip"];
	        this.fraudScore = source["fraudScore"];
	        this.isResidential = source["isResidential"];
	        this.isBroadcast = source["isBroadcast"];
	        this.country = source["country"];
	        this.region = source["region"];
	        this.city = source["city"];
	        this.asOrganization = source["asOrganization"];
	        this.rawData = source["rawData"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class ProxyTestResult {
	    proxyId: string;
	    ok: boolean;
	    latencyMs: number;
	    error: string;

	    static createFrom(source: any = {}) {
	        return new ProxyTestResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyId = source["proxyId"];
	        this.ok = source["ok"];
	        this.latencyMs = source["latencyMs"];
	        this.error = source["error"];
	    }
	}
	export class ProxyValidationResult {
	    supported: boolean;
	    errorMsg: string;

	    static createFrom(source: any = {}) {
	        return new ProxyValidationResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supported = source["supported"];
	        this.errorMsg = source["errorMsg"];
	    }
	}
	export class SnapshotInfo {
	    snapshotId: string;
	    profileId: string;
	    name: string;
	    sizeMB: number;
	    createdAt: string;
	    filePath?: string;

	    static createFrom(source: any = {}) {
	        return new SnapshotInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.snapshotId = source["snapshotId"];
	        this.profileId = source["profileId"];
	        this.name = source["name"];
	        this.sizeMB = source["sizeMB"];
	        this.createdAt = source["createdAt"];
	        this.filePath = source["filePath"];
	    }
	}

}

export namespace backup {

	export class ManifestEntry {
	    id: string;
	    category: string;
	    entryType: string;
	    required: boolean;
	    archivePath: string;
	    description?: string;

	    static createFrom(source: any = {}) {
	        return new ManifestEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.entryType = source["entryType"];
	        this.required = source["required"];
	        this.archivePath = source["archivePath"];
	        this.description = source["description"];
	    }
	}
	export class ManifestAppInfo {
	    name: string;
	    version: string;

	    static createFrom(source: any = {}) {
	        return new ManifestAppInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	    }
	}
	export class Manifest {
	    format: string;
	    manifestVersion: number;
	    createdAt: string;
	    app: ManifestAppInfo;
	    entries: ManifestEntry[];

	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.manifestVersion = source["manifestVersion"];
	        this.createdAt = source["createdAt"];
	        this.app = this.convertValues(source["app"], ManifestAppInfo);
	        this.entries = this.convertValues(source["entries"], ManifestEntry);
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


	export class ScopeEntry {
	    id: string;
	    category: string;
	    entryType: string;
	    required: boolean;
	    sourcePath: string;
	    archivePath: string;
	    exists: boolean;
	    description?: string;

	    static createFrom(source: any = {}) {
	        return new ScopeEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.category = source["category"];
	        this.entryType = source["entryType"];
	        this.required = source["required"];
	        this.sourcePath = source["sourcePath"];
	        this.archivePath = source["archivePath"];
	        this.exists = source["exists"];
	        this.description = source["description"];
	    }
	}
	export class Scope {
	    format: string;
	    manifestVersion: number;
	    appRoot: string;
	    entries: ScopeEntry[];

	    static createFrom(source: any = {}) {
	        return new Scope(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.manifestVersion = source["manifestVersion"];
	        this.appRoot = source["appRoot"];
	        this.entries = this.convertValues(source["entries"], ScopeEntry);
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

export namespace browser {

	export class CoreExtendedInfo {
	    coreId: string;
	    chromeVersion: string;
	    instanceCount: number;

	    static createFrom(source: any = {}) {
	        return new CoreExtendedInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.coreId = source["coreId"];
	        this.chromeVersion = source["chromeVersion"];
	        this.instanceCount = source["instanceCount"];
	    }
	}
	export class CoreInput {
	    coreId: string;
	    coreName: string;
	    corePath: string;
	    isDefault: boolean;

	    static createFrom(source: any = {}) {
	        return new CoreInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.coreId = source["coreId"];
	        this.coreName = source["coreName"];
	        this.corePath = source["corePath"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class CoreValidateResult {
	    valid: boolean;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new CoreValidateResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.message = source["message"];
	    }
	}
	export class Group {
	    groupId: string;
	    groupName: string;
	    parentId: string;
	    sortOrder: number;
	    createdAt: string;
	    updatedAt: string;

	    static createFrom(source: any = {}) {
	        return new Group(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groupId = source["groupId"];
	        this.groupName = source["groupName"];
	        this.parentId = source["parentId"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class GroupInput {
	    groupName: string;
	    parentId: string;
	    sortOrder: number;

	    static createFrom(source: any = {}) {
	        return new GroupInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groupName = source["groupName"];
	        this.parentId = source["parentId"];
	        this.sortOrder = source["sortOrder"];
	    }
	}
	export class GroupWithCount {
	    groupId: string;
	    groupName: string;
	    parentId: string;
	    sortOrder: number;
	    createdAt: string;
	    updatedAt: string;
	    instanceCount: number;

	    static createFrom(source: any = {}) {
	        return new GroupWithCount(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.groupId = source["groupId"];
	        this.groupName = source["groupName"];
	        this.parentId = source["parentId"];
	        this.sortOrder = source["sortOrder"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.instanceCount = source["instanceCount"];
	    }
	}
	export class Profile {
	    profileId: string;
	    profileName: string;
	    userDataDir: string;
	    coreId: string;
	    fingerprintArgs: string[];
	    proxyId: string;
	    proxyConfig: string;
	    proxyBindSourceId: string;
	    proxyBindSourceUrl: string;
	    proxyBindName: string;
	    proxyBindUpdatedAt: string;
	    launchArgs: string[];
	    tags: string[];
	    keywords: string[];
	    groupId: string;
	    launchCode: string;
	    running: boolean;
	    debugPort: number;
	    debugReady: boolean;
	    pid: number;
	    runtimeWarning: string;
	    lastError: string;
	    createdAt: string;
	    updatedAt: string;
	    lastStartAt: string;
	    lastStopAt: string;

	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileId = source["profileId"];
	        this.profileName = source["profileName"];
	        this.userDataDir = source["userDataDir"];
	        this.coreId = source["coreId"];
	        this.fingerprintArgs = source["fingerprintArgs"];
	        this.proxyId = source["proxyId"];
	        this.proxyConfig = source["proxyConfig"];
	        this.proxyBindSourceId = source["proxyBindSourceId"];
	        this.proxyBindSourceUrl = source["proxyBindSourceUrl"];
	        this.proxyBindName = source["proxyBindName"];
	        this.proxyBindUpdatedAt = source["proxyBindUpdatedAt"];
	        this.launchArgs = source["launchArgs"];
	        this.tags = source["tags"];
	        this.keywords = source["keywords"];
	        this.groupId = source["groupId"];
	        this.launchCode = source["launchCode"];
	        this.running = source["running"];
	        this.debugPort = source["debugPort"];
	        this.debugReady = source["debugReady"];
	        this.pid = source["pid"];
	        this.runtimeWarning = source["runtimeWarning"];
	        this.lastError = source["lastError"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lastStartAt = source["lastStartAt"];
	        this.lastStopAt = source["lastStopAt"];
	    }
	}
	export class ProfileInput {
	    profileName: string;
	    userDataDir: string;
	    coreId: string;
	    fingerprintArgs: string[];
	    proxyId: string;
	    proxyConfig: string;
	    launchArgs: string[];
	    tags: string[];
	    keywords: string[];
	    groupId: string;

	    static createFrom(source: any = {}) {
	        return new ProfileInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profileName = source["profileName"];
	        this.userDataDir = source["userDataDir"];
	        this.coreId = source["coreId"];
	        this.fingerprintArgs = source["fingerprintArgs"];
	        this.proxyId = source["proxyId"];
	        this.proxyConfig = source["proxyConfig"];
	        this.launchArgs = source["launchArgs"];
	        this.tags = source["tags"];
	        this.keywords = source["keywords"];
	        this.groupId = source["groupId"];
	    }
	}
	export class Settings {
	    userDataRoot: string;
	    defaultFingerprintArgs: string[];
	    defaultLaunchArgs: string[];
	    defaultStartUrls: string[];
	    restoreLastSession: boolean;
	    startReadyTimeoutMs: number;
	    startStableWindowMs: number;

	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userDataRoot = source["userDataRoot"];
	        this.defaultFingerprintArgs = source["defaultFingerprintArgs"];
	        this.defaultLaunchArgs = source["defaultLaunchArgs"];
	        this.defaultStartUrls = source["defaultStartUrls"];
	        this.restoreLastSession = source["restoreLastSession"];
	        this.startReadyTimeoutMs = source["startReadyTimeoutMs"];
	        this.startStableWindowMs = source["startStableWindowMs"];
	    }
	}
	export class Tab {
	    tabId: string;
	    title: string;
	    url: string;
	    active: boolean;

	    static createFrom(source: any = {}) {
	        return new Tab(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabId = source["tabId"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.active = source["active"];
	    }
	}

}

export namespace config {

	export class BrowserBookmark {
	    name: string;
	    url: string;
	    openOnStart: boolean;

	    static createFrom(source: any = {}) {
	        return new BrowserBookmark(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.url = source["url"];
	        this.openOnStart = source["openOnStart"];
	    }
	}
	export class BrowserCore {
	    coreId: string;
	    coreName: string;
	    corePath: string;
	    isDefault: boolean;

	    static createFrom(source: any = {}) {
	        return new BrowserCore(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.coreId = source["coreId"];
	        this.coreName = source["coreName"];
	        this.corePath = source["corePath"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class BrowserProxy {
	    proxyId: string;
	    proxyName: string;
	    proxyConfig: string;
	    dnsServers?: string;
	    groupName?: string;
	    sortOrder?: number;
	    sourceId?: string;
	    sourceUrl?: string;
	    sourceNamePrefix?: string;
	    sourceAutoRefresh?: boolean;
	    sourceRefreshIntervalM?: number;
	    sourceLastRefreshAt?: string;
	    lastLatencyMs: number;
	    lastTestOk: boolean;
	    lastTestedAt: string;
	    lastIPHealthJson?: string;

	    static createFrom(source: any = {}) {
	        return new BrowserProxy(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyId = source["proxyId"];
	        this.proxyName = source["proxyName"];
	        this.proxyConfig = source["proxyConfig"];
	        this.dnsServers = source["dnsServers"];
	        this.groupName = source["groupName"];
	        this.sortOrder = source["sortOrder"];
	        this.sourceId = source["sourceId"];
	        this.sourceUrl = source["sourceUrl"];
	        this.sourceNamePrefix = source["sourceNamePrefix"];
	        this.sourceAutoRefresh = source["sourceAutoRefresh"];
	        this.sourceRefreshIntervalM = source["sourceRefreshIntervalM"];
	        this.sourceLastRefreshAt = source["sourceLastRefreshAt"];
	        this.lastLatencyMs = source["lastLatencyMs"];
	        this.lastTestOk = source["lastTestOk"];
	        this.lastTestedAt = source["lastTestedAt"];
	        this.lastIPHealthJson = source["lastIPHealthJson"];
	    }
	}
	export class ProxyCheckTarget {
	    id: string;
	    name: string;
	    type: string;
	    url: string;
	    parser?: string;
	    timeoutMs?: number;
	    expectedStatus?: number[];

	    static createFrom(source: any = {}) {
	        return new ProxyCheckTarget(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.url = source["url"];
	        this.parser = source["parser"];
	        this.timeoutMs = source["timeoutMs"];
	        this.expectedStatus = source["expectedStatus"];
	    }
	}
	export class ProxyCheckConfig {
	    bridgeStartTimeoutMs: number;
	    speedTargetId: string;
	    ipHealthTargetId: string;
	    targets: ProxyCheckTarget[];

	    static createFrom(source: any = {}) {
	        return new ProxyCheckConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bridgeStartTimeoutMs = source["bridgeStartTimeoutMs"];
	        this.speedTargetId = source["speedTargetId"];
	        this.ipHealthTargetId = source["ipHealthTargetId"];
	        this.targets = this.convertValues(source["targets"], ProxyCheckTarget);
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

export namespace launchcode {

	export class LaunchRequestParams {
	    launchArgs: string[];
	    startUrls: string[];
	    skipDefaultStartUrls: boolean;
	    proxyId: string;
	    proxyConfig: string;

	    static createFrom(source: any = {}) {
	        return new LaunchRequestParams(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.launchArgs = source["launchArgs"];
	        this.startUrls = source["startUrls"];
	        this.skipDefaultStartUrls = source["skipDefaultStartUrls"];
	        this.proxyId = source["proxyId"];
	        this.proxyConfig = source["proxyConfig"];
	    }
	}

}

export namespace logger {

	export class MemoryLogEntry {
	    time: string;
	    level: string;
	    component: string;
	    message: string;
	    fields?: Record<string, any>;

	    static createFrom(source: any = {}) {
	        return new MemoryLogEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.component = source["component"];
	        this.message = source["message"];
	        this.fields = source["fields"];
	    }
	}
	export class MethodInterceptor {


	    static createFrom(source: any = {}) {
	        return new MethodInterceptor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);

	    }
	}

}
