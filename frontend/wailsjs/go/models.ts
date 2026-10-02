export namespace cores {

	export class Info {
	    id: string;
	    name: string;
	    version: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.version = source["version"];
	        this.description = source["description"];
	    }
	}

}

export namespace device {

	export class Info {
	    hwid: string;
	    os: string;
	    osVersion: string;
	    model: string;
	    userAgent: string;

	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hwid = source["hwid"];
	        this.os = source["os"];
	        this.osVersion = source["osVersion"];
	        this.model = source["model"];
	        this.userAgent = source["userAgent"];
	    }
	}

}

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
	export class ApplicationTraffic {
	    process: string;
	    upload: number;
	    download: number;
	    connections: number;

	    static createFrom(source: any = {}) {
	        return new ApplicationTraffic(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.process = source["process"];
	        this.upload = source["upload"];
	        this.download = source["download"];
	        this.connections = source["connections"];
	    }
	}
	export class ConnectionMetadata {
	    network: string;
	    host: string;
	    destinationIP: string;
	    destinationPort: string;
	    process: string;
	    processPath: string;

	    static createFrom(source: any = {}) {
	        return new ConnectionMetadata(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.network = source["network"];
	        this.host = source["host"];
	        this.destinationIP = source["destinationIP"];
	        this.destinationPort = source["destinationPort"];
	        this.process = source["process"];
	        this.processPath = source["processPath"];
	    }
	}
	export class LiveConnection {
	    id: string;
	    metadata: ConnectionMetadata;
	    upload: number;
	    download: number;
	    start: string;
	    chains: string[];
	    rule: string;

	    static createFrom(source: any = {}) {
	        return new LiveConnection(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.metadata = this.convertValues(source["metadata"], ConnectionMetadata);
	        this.upload = source["upload"];
	        this.download = source["download"];
	        this.start = source["start"];
	        this.chains = source["chains"];
	        this.rule = source["rule"];
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
	export class ConnectionSnapshot {
	    connections: LiveConnection[];
	    applications: ApplicationTraffic[];
	    uploadTotal: number;
	    downloadTotal: number;
	    memory: number;

	    static createFrom(source: any = {}) {
	        return new ConnectionSnapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connections = this.convertValues(source["connections"], LiveConnection);
	        this.applications = this.convertValues(source["applications"], ApplicationTraffic);
	        this.uploadTotal = source["uploadTotal"];
	        this.downloadTotal = source["downloadTotal"];
	        this.memory = source["memory"];
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
	export class CoreSupport {
	    core: string;
	    supported: boolean;
	    reason?: string;

	    static createFrom(source: any = {}) {
	        return new CoreSupport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.core = source["core"];
	        this.supported = source["supported"];
	        this.reason = source["reason"];
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
	    path: string;
	    application: boolean;
	    icon: string;

	    static createFrom(source: any = {}) {
	        return new ProcessInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.application = source["application"];
	        this.icon = source["icon"];
	    }
	}
	export class ProfileCores {
	    selected: string;
	    error?: string;
	    cores: CoreSupport[];

	    static createFrom(source: any = {}) {
	        return new ProfileCores(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selected = source["selected"];
	        this.error = source["error"];
	        this.cores = this.convertValues(source["cores"], CoreSupport);
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
	export class RuntimeStats {
	    goVersion: string;
	    goroutines: number;
	    cpus: number;
	    heapBytes: number;
	    heapObjects: number;
	    systemBytes: number;
	    gcCount: number;
	    gcPauseMs: number;
	    uptimeSeconds: number;
	    pid: number;
	    status: model.Status;

	    static createFrom(source: any = {}) {
	        return new RuntimeStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.goVersion = source["goVersion"];
	        this.goroutines = source["goroutines"];
	        this.cpus = source["cpus"];
	        this.heapBytes = source["heapBytes"];
	        this.heapObjects = source["heapObjects"];
	        this.systemBytes = source["systemBytes"];
	        this.gcCount = source["gcCount"];
	        this.gcPauseMs = source["gcPauseMs"];
	        this.uptimeSeconds = source["uptimeSeconds"];
	        this.pid = source["pid"];
	        this.status = this.convertValues(source["status"], model.Status);
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
	export class UpdateInfo {
	    version: string;
	    available: boolean;
	    url: string;
	    assetUrl: string;
	    assetName: string;
	    digest: string;
	    downloadPath: string;
	    downloading: boolean;
	    progress: number;
	    error: string;
	    checkedAt: number;

	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.available = source["available"];
	        this.url = source["url"];
	        this.assetUrl = source["assetUrl"];
	        this.assetName = source["assetName"];
	        this.digest = source["digest"];
	        this.downloadPath = source["downloadPath"];
	        this.downloading = source["downloading"];
	        this.progress = source["progress"];
	        this.error = source["error"];
	        this.checkedAt = source["checkedAt"];
	    }
	}

}

export namespace model {

	export class Point {
	    x: number;
	    y: number;

	    static createFrom(source: any = {}) {
	        return new Point(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class RouteNode {
	    id: string;
	    type: string;
	    name: string;
	    values: string[];
	    icons?: Record<string, string>;
	    action: string;
	    x: number;
	    y: number;

	    static createFrom(source: any = {}) {
	        return new RouteNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.name = source["name"];
	        this.values = source["values"];
	        this.icons = source["icons"];
	        this.action = source["action"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class RouteNote {
	    id: string;
	    text: string;
	    x: number;
	    y: number;

	    static createFrom(source: any = {}) {
	        return new RouteNote(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.text = source["text"];
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class RouteGraph {
	    notes?: RouteNote[];
	    nodes: RouteNode[];
	    finalHidden: boolean;
	    final: string;
	    layout: Record<string, Point>;

	    static createFrom(source: any = {}) {
	        return new RouteGraph(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.notes = this.convertValues(source["notes"], RouteNote);
	        this.nodes = this.convertValues(source["nodes"], RouteNode);
	        this.finalHidden = source["finalHidden"];
	        this.final = source["final"];
	        this.layout = this.convertValues(source["layout"], Point, true);
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
	export class CustomTheme {
	    id: string;
	    name: string;
	    colors: Record<string, string>;
	    accent: string;

	    static createFrom(source: any = {}) {
	        return new CustomTheme(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.colors = source["colors"];
	        this.accent = source["accent"];
	    }
	}
	export class AppSettings {
	    clientPreset: string;
	    clientVersion: string;
	    customThemes: CustomTheme[];
	    uiScale: number;
	    density: string;
	    customRadius: number;
	    autoUpdate: boolean;
	    settingsVersion: number;
	    core: string;
	    activeProfileId: string;
	    dns: string;
	    dnsFallback: string;
	    tunName: string;
	    stack: string;
	    mtu: number;
	    rules: RoutingRule[];
	    routingMode: string;
	    simpleFinal?: string;
	    graph: RouteGraph;
	    ipv6: boolean;
	    strictRoute: boolean;
	    sniff: boolean;
	    pingMethod: string;
	    pingUrl: string;
	    pingTimeout: number;
	    hwidEnabled: boolean;
	    hwid: string;
	    deviceOs: string;
	    osVersion: string;
	    deviceModel: string;
	    userAgent: string;
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
	        this.clientPreset = source["clientPreset"];
	        this.clientVersion = source["clientVersion"];
	        this.customThemes = this.convertValues(source["customThemes"], CustomTheme);
	        this.uiScale = source["uiScale"];
	        this.density = source["density"];
	        this.customRadius = source["customRadius"];
	        this.autoUpdate = source["autoUpdate"];
	        this.settingsVersion = source["settingsVersion"];
	        this.core = source["core"];
	        this.activeProfileId = source["activeProfileId"];
	        this.dns = source["dns"];
	        this.dnsFallback = source["dnsFallback"];
	        this.tunName = source["tunName"];
	        this.stack = source["stack"];
	        this.mtu = source["mtu"];
	        this.rules = this.convertValues(source["rules"], RoutingRule);
	        this.routingMode = source["routingMode"];
	        this.simpleFinal = source["simpleFinal"];
	        this.graph = this.convertValues(source["graph"], RouteGraph);
	        this.ipv6 = source["ipv6"];
	        this.strictRoute = source["strictRoute"];
	        this.sniff = source["sniff"];
	        this.pingMethod = source["pingMethod"];
	        this.pingUrl = source["pingUrl"];
	        this.pingTimeout = source["pingTimeout"];
	        this.hwidEnabled = source["hwidEnabled"];
	        this.hwid = source["hwid"];
	        this.deviceOs = source["deviceOs"];
	        this.osVersion = source["osVersion"];
	        this.deviceModel = source["deviceModel"];
	        this.userAgent = source["userAgent"];
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


	export class ProfileSpeed {
	    downloadMbps: number;
	    bytes: number;
	    durationMs: number;
	    core: string;
	    measuredAt: number;

	    static createFrom(source: any = {}) {
	        return new ProfileSpeed(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloadMbps = source["downloadMbps"];
	        this.bytes = source["bytes"];
	        this.durationMs = source["durationMs"];
	        this.core = source["core"];
	        this.measuredAt = source["measuredAt"];
	    }
	}
	export class Profile {
	    speed?: ProfileSpeed;
	    id: string;
	    name: string;
	    protocol: string;
	    address: string;
	    port: number;
	    uuid?: string;
	    username?: string;
	    password?: string;
	    method?: string;
	    alterId?: number;
	    encryption?: string;
	    packetEncoding?: string;
	    network?: string;
	    security?: string;
	    sni?: string;
	    alpn?: string;
	    fingerprint?: string;
	    allowInsecure?: boolean;
	    flow?: string;
	    publicKey?: string;
	    shortId?: string;
	    spiderX?: string;
	    path?: string;
	    host?: string;
	    serviceName?: string;
	    headerType?: string;
	    seed?: string;
	    mode?: string;
	    extra?: string;
	    plugin?: string;
	    pluginOpts?: string;
	    obfs?: string;
	    obfsPassword?: string;
	    upMbps?: number;
	    downMbps?: number;
	    congestion?: string;
	    udpRelayMode?: string;
	    ports?: string;
	    privateKey?: string;
	    preSharedKey?: string;
	    localAddress?: string;
	    reserved?: string;
	    mtu?: number;
	    raw?: string;
	    subId?: string;

	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.speed = this.convertValues(source["speed"], ProfileSpeed);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.uuid = source["uuid"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.method = source["method"];
	        this.alterId = source["alterId"];
	        this.encryption = source["encryption"];
	        this.packetEncoding = source["packetEncoding"];
	        this.network = source["network"];
	        this.security = source["security"];
	        this.sni = source["sni"];
	        this.alpn = source["alpn"];
	        this.fingerprint = source["fingerprint"];
	        this.allowInsecure = source["allowInsecure"];
	        this.flow = source["flow"];
	        this.publicKey = source["publicKey"];
	        this.shortId = source["shortId"];
	        this.spiderX = source["spiderX"];
	        this.path = source["path"];
	        this.host = source["host"];
	        this.serviceName = source["serviceName"];
	        this.headerType = source["headerType"];
	        this.seed = source["seed"];
	        this.mode = source["mode"];
	        this.extra = source["extra"];
	        this.plugin = source["plugin"];
	        this.pluginOpts = source["pluginOpts"];
	        this.obfs = source["obfs"];
	        this.obfsPassword = source["obfsPassword"];
	        this.upMbps = source["upMbps"];
	        this.downMbps = source["downMbps"];
	        this.congestion = source["congestion"];
	        this.udpRelayMode = source["udpRelayMode"];
	        this.ports = source["ports"];
	        this.privateKey = source["privateKey"];
	        this.preSharedKey = source["preSharedKey"];
	        this.localAddress = source["localAddress"];
	        this.reserved = source["reserved"];
	        this.mtu = source["mtu"];
	        this.raw = source["raw"];
	        this.subId = source["subId"];
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

export namespace vpn {

	export class SpeedResult {
	    downloadMbps: number;
	    bytes: number;
	    durationMs: number;
	    core: string;

	    static createFrom(source: any = {}) {
	        return new SpeedResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.downloadMbps = source["downloadMbps"];
	        this.bytes = source["bytes"];
	        this.durationMs = source["durationMs"];
	        this.core = source["core"];
	    }
	}

}

