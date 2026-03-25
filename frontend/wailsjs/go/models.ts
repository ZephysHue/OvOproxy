export namespace config {
	
	export class Subscription {
	    id: string;
	    name: string;
	    url: string;
	    enabled: boolean;
	    last_updated?: string;
	    last_status?: string;
	
	    static createFrom(source: any = {}) {
	        return new Subscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.enabled = source["enabled"];
	        this.last_updated = source["last_updated"];
	        this.last_status = source["last_status"];
	    }
	}
	export class SubscriptionRefreshSettings {
	    auto_enabled: boolean;
	    interval_seconds?: number;
	    max_backoff_seconds?: number;
	    history_limit?: number;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionRefreshSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auto_enabled = source["auto_enabled"];
	        this.interval_seconds = source["interval_seconds"];
	        this.max_backoff_seconds = source["max_backoff_seconds"];
	        this.history_limit = source["history_limit"];
	    }
	}

}

export namespace main {
	
	export class AuditLogEntry {
	    time: string;
	    action: string;
	    profile: string;
	    detail: string;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuditLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.action = source["action"];
	        this.profile = source["profile"];
	        this.detail = source["detail"];
	        this.success = source["success"];
	    }
	}
	export class BackupInfo {
	    file_name: string;
	    path: string;
	    size: number;
	    modified: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.file_name = source["file_name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.modified = source["modified"];
	    }
	}
	export class DuplicateDomain {
	    domain: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new DuplicateDomain(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.count = source["count"];
	    }
	}
	export class HostEntry {
	    domain: string;
	    ip: string;
	
	    static createFrom(source: any = {}) {
	        return new HostEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.ip = source["ip"];
	    }
	}
	export class ProfileState {
	    name: string;
	    listen_ip: string;
	    port: number;
	    hosts_file: string;
	    subscriptions?: config.Subscription[];
	    subscription_refresh?: config.SubscriptionRefreshSettings;
	    running: boolean;
	    hosts: Record<string, string>;
	    duplicate_domains: DuplicateDomain[];
	    system_hosts_active: boolean;
	    proxy_active: boolean;
	    proxy_error: string;
	
	    static createFrom(source: any = {}) {
	        return new ProfileState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.listen_ip = source["listen_ip"];
	        this.port = source["port"];
	        this.hosts_file = source["hosts_file"];
	        this.subscriptions = this.convertValues(source["subscriptions"], config.Subscription);
	        this.subscription_refresh = this.convertValues(source["subscription_refresh"], config.SubscriptionRefreshSettings);
	        this.running = source["running"];
	        this.hosts = source["hosts"];
	        this.duplicate_domains = this.convertValues(source["duplicate_domains"], DuplicateDomain);
	        this.system_hosts_active = source["system_hosts_active"];
	        this.proxy_active = source["proxy_active"];
	        this.proxy_error = source["proxy_error"];
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
	export class ProxyLogEntry {
	    time: string;
	    method: string;
	    host: string;
	    resolved_ip: string;
	    success: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyLogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.method = source["method"];
	        this.host = source["host"];
	        this.resolved_ip = source["resolved_ip"];
	        this.success = source["success"];
	        this.error = source["error"];
	    }
	}
	export class SubscriptionConflictItem {
	    domain: string;
	    local_ip: string;
	    remote_ip: string;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionConflictItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.local_ip = source["local_ip"];
	        this.remote_ip = source["remote_ip"];
	    }
	}
	export class SubscriptionConflictPreview {
	    sub_id: string;
	    sub_name: string;
	    domains: string[];
	    items: SubscriptionConflictItem[];
	    total: number;
	    truncate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionConflictPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sub_id = source["sub_id"];
	        this.sub_name = source["sub_name"];
	        this.domains = source["domains"];
	        this.items = this.convertValues(source["items"], SubscriptionConflictItem);
	        this.total = source["total"];
	        this.truncate = source["truncate"];
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
	export class SubscriptionRefreshFailure {
	    sub_id: string;
	    sub_name: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionRefreshFailure(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sub_id = source["sub_id"];
	        this.sub_name = source["sub_name"];
	        this.reason = source["reason"];
	    }
	}
	export class SubscriptionRefreshReport {
	    time: string;
	    source: string;
	    success: boolean;
	    enabled_total: number;
	    success_total: number;
	    failed_total: number;
	    added_total: number;
	    conflict_diff: number;
	    conflict_same: number;
	    failures: SubscriptionRefreshFailure[];
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionRefreshReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.source = source["source"];
	        this.success = source["success"];
	        this.enabled_total = source["enabled_total"];
	        this.success_total = source["success_total"];
	        this.failed_total = source["failed_total"];
	        this.added_total = source["added_total"];
	        this.conflict_diff = source["conflict_diff"];
	        this.conflict_same = source["conflict_same"];
	        this.failures = this.convertValues(source["failures"], SubscriptionRefreshFailure);
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

