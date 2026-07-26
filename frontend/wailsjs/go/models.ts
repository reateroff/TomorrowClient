export namespace main {
	
	export class AppInfo {
	    version: string;
	    copyright: string;
	    builtWith: string;
	    license: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.copyright = source["copyright"];
	        this.builtWith = source["builtWith"];
	        this.license = source["license"];
	    }
	}
	export class PingResult {
	    latencyMs: number;
	    ok: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PingResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.latencyMs = source["latencyMs"];
	        this.ok = source["ok"];
	    }
	}
	export class ProcessInfo {
	    name: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.icon = source["icon"];
	    }
	}

}

export namespace model {
	
	export class RoutingRule {
	    type: string;
	    value: string;
	    action: string;
	    icon: string;
	
	    static createFrom(source: any = {}) {
	        return new RoutingRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.value = source["value"];
	        this.action = source["action"];
	        this.icon = source["icon"];
	    }
	}
	export class AppSettings {
	    settingsVersion: number;
	    core: string;
	    activeProfileId: string;
	    dns: string;
	    dnsFallback: string;
	    tunName: string;
	    stack: string;
	    mtu: number;
	    rules: RoutingRule[];
	    autoConnect: boolean;
	    launchAtStartup: boolean;
	    minimizeToTray: boolean;
	    devMode: boolean;
	    demoMode: boolean;
	    theme: string;
	    accent: string;
	    savedColors: string[];
	    font: string;
	    radius: string;
	    navPosition: string;
	    animation: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settingsVersion = source["settingsVersion"];
	        this.core = source["core"];
	        this.activeProfileId = source["activeProfileId"];
	        this.dns = source["dns"];
	        this.dnsFallback = source["dnsFallback"];
	        this.tunName = source["tunName"];
	        this.stack = source["stack"];
	        this.mtu = source["mtu"];
	        this.rules = this.convertValues(source["rules"], RoutingRule);
	        this.autoConnect = source["autoConnect"];
	        this.launchAtStartup = source["launchAtStartup"];
	        this.minimizeToTray = source["minimizeToTray"];
	        this.devMode = source["devMode"];
	        this.demoMode = source["demoMode"];
	        this.theme = source["theme"];
	        this.accent = source["accent"];
	        this.savedColors = source["savedColors"];
	        this.font = source["font"];
	        this.radius = source["radius"];
	        this.navPosition = source["navPosition"];
	        this.animation = source["animation"];
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
	export class Profile {
	    id: string;
	    name: string;
	    protocol: string;
	    address: string;
	    port: number;
	    uuid?: string;
	    password?: string;
	    method?: string;
	    alterId?: number;
	    network?: string;
	    security?: string;
	    sni?: string;
	    alpn?: string;
	    fingerprint?: string;
	    flow?: string;
	    publicKey?: string;
	    shortId?: string;
	    path?: string;
	    host?: string;
	    serviceName?: string;
	    obfs?: string;
	    obfsPassword?: string;
	    upMbps?: number;
	    downMbps?: number;
	    congestion?: string;
	    udpRelayMode?: string;
	    raw?: string;
	    subId?: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.uuid = source["uuid"];
	        this.password = source["password"];
	        this.method = source["method"];
	        this.alterId = source["alterId"];
	        this.network = source["network"];
	        this.security = source["security"];
	        this.sni = source["sni"];
	        this.alpn = source["alpn"];
	        this.fingerprint = source["fingerprint"];
	        this.flow = source["flow"];
	        this.publicKey = source["publicKey"];
	        this.shortId = source["shortId"];
	        this.path = source["path"];
	        this.host = source["host"];
	        this.serviceName = source["serviceName"];
	        this.obfs = source["obfs"];
	        this.obfsPassword = source["obfsPassword"];
	        this.upMbps = source["upMbps"];
	        this.downMbps = source["downMbps"];
	        this.congestion = source["congestion"];
	        this.udpRelayMode = source["udpRelayMode"];
	        this.raw = source["raw"];
	        this.subId = source["subId"];
	    }
	}
	
	export class Stats {
	    upload: number;
	    download: number;
	    uploadSpeed: number;
	    downloadSpeed: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.upload = source["upload"];
	        this.download = source["download"];
	        this.uploadSpeed = source["uploadSpeed"];
	        this.downloadSpeed = source["downloadSpeed"];
	    }
	}
	export class Status {
	    state: string;
	    core: string;
	    activeProfile?: Profile;
	    error?: string;
	    stats: Stats;
	    connectedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.core = source["core"];
	        this.activeProfile = this.convertValues(source["activeProfile"], Profile);
	        this.error = source["error"];
	        this.stats = this.convertValues(source["stats"], Stats);
	        this.connectedAt = source["connectedAt"];
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
	export class Subscription {
	    id: string;
	    name: string;
	    url: string;
	    updatedAt: number;
	    count: number;
	    upload: number;
	    download: number;
	    total: number;
	    expire: number;
	
	    static createFrom(source: any = {}) {
	        return new Subscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.updatedAt = source["updatedAt"];
	        this.count = source["count"];
	        this.upload = source["upload"];
	        this.download = source["download"];
	        this.total = source["total"];
	        this.expire = source["expire"];
	    }
	}

}

