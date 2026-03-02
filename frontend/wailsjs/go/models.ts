export namespace main {
	
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

}

