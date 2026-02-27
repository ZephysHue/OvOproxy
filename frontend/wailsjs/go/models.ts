export namespace main {
	
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
	    }
	}

}

