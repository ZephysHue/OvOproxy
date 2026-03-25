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

}

export namespace main {
	
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
	export class SubscriptionConflictPreview {
	    sub_id: string;
	    sub_name: string;
	    domains: string[];
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
	        this.total = source["total"];
	        this.truncate = source["truncate"];
	    }
	}

}

