export namespace models {
	
	export class Connection {
	    id: string;
	    type: string;
	    ip: string;
	    port: string;
	    user: string;
	    pass: string;
	    status: string;
	    message: string;
	    result: string;
	    logs: string[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    connected_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new Connection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.pass = source["pass"];
	        this.status = source["status"];
	        this.message = source["message"];
	        this.result = source["result"];
	        this.logs = source["logs"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.connected_at = this.convertValues(source["connected_at"], null);
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
	export class ConnectionRequest {
	    type: string;
	    ip: string;
	    port: string;
	    user: string;
	    pass: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.pass = source["pass"];
	    }
	}
	export class ProxyConfig {
	    enabled: boolean;
	    type: string;
	    host: string;
	    port: string;
	    user: string;
	    pass: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.pass = source["pass"];
	    }
	}
	export class VulnerabilityInfo {
	    id: string;
	    service_type: string;
	    name: string;
	    level: string;
	    description: string;
	    repair: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	
	    static createFrom(source: any = {}) {
	        return new VulnerabilityInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.service_type = source["service_type"];
	        this.name = source["name"];
	        this.level = source["level"];
	        this.description = source["description"];
	        this.repair = source["repair"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
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

export namespace services {
	
	export class VulnerabilityData {
	    name: string;
	    level: string;
	    target: string;
	    describe: string;
	    images: string[];
	    repair: string;
	
	    static createFrom(source: any = {}) {
	        return new VulnerabilityData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.level = source["level"];
	        this.target = source["target"];
	        this.describe = source["describe"];
	        this.images = source["images"];
	        this.repair = source["repair"];
	    }
	}
	export class ExportReportRequest {
	    connectionIds: string[];
	    vulnerabilities: VulnerabilityData[];
	    outputPath: string;
	
	    static createFrom(source: any = {}) {
	        return new ExportReportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionIds = source["connectionIds"];
	        this.vulnerabilities = this.convertValues(source["vulnerabilities"], VulnerabilityData);
	        this.outputPath = source["outputPath"];
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

