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
	export class ProfileState {
	    name: string;
	    listen_ip: string;
	    port: number;
	    hosts_file: string;
	    running: boolean;
	    hosts: Record<string, string>;
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
	        this.system_hosts_active = source["system_hosts_active"];
	        this.proxy_active = source["proxy_active"];
	        this.proxy_error = source["proxy_error"];
	    }
	}
	export class SubscriptionResult {
	    status: string;
	    message: string;
	    last_fetch: string;
	    entry_count: number;
	
	    static createFrom(source: any = {}) {
	        return new SubscriptionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.message = source["message"];
	        this.last_fetch = source["last_fetch"];
	        this.entry_count = source["entry_count"];
	    }
	}

}

