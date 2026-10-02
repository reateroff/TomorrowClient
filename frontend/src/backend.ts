// Explicit typed bridge for new Wails methods; generated bindings remain Wails-owned.
import type { AppSettings } from './types';
export interface UpdateInfo {version:string;available:boolean;url:string;assetUrl:string;assetName:string;digest:string;downloadPath:string;downloading:boolean;progress:number;error:string;checkedAt:number}
export interface Connection {id:string;metadata:{network:string;host:string;destinationIP:string;destinationPort:string;process:string;processPath:string};upload:number;download:number;start:string;chains:string[];rule:string}
export interface AppTraffic {process:string;upload:number;download:number;connections:number}
export interface Connections {connections:Connection[];applications:AppTraffic[];uploadTotal:number;downloadTotal:number;memory:number}
export interface RuntimeStats {goVersion:string;goroutines:number;cpus:number;heapBytes:number;heapObjects:number;systemBytes:number;gcCount:number;gcPauseMs:number;uptimeSeconds:number;pid:number;status:{state:string;core:string}}
export interface SpeedResult {downloadMbps:number;bytes:number;durationMs:number;core:string}
function call<T>(name:string,...args:unknown[]):Promise<T> {const w=window as unknown as {go:{main:{App:Record<string,(...a:unknown[])=>Promise<T>>}}};return w.go.main.App[name](...args)}
export const GetConnections=()=>call<Connections>('GetConnections');
export const CloseConnection=(id:string)=>call<void>('CloseConnection',id);
export const GetRuntimeStats=()=>call<RuntimeStats>('GetRuntimeStats');
export const GetGoroutineDump=()=>call<string>('GetGoroutineDump');
export const CollectGarbage=()=>call<void>('CollectGarbage');
export const ExportHeapProfile=()=>call<string>('ExportHeapProfile');
export const TestProfileSpeed=(id:string)=>call<SpeedResult>('TestProfileSpeed',id);
export const ImportRouting=()=>call<AppSettings|null>('ImportRouting');
export const ExportRouting=()=>call<string>('ExportRouting');
export const GetUpdateInfo=()=>call<UpdateInfo>('GetUpdateInfo');
export const CheckUpdates=()=>call<UpdateInfo>('CheckUpdates');
export const DownloadUpdate=()=>call<string>('DownloadUpdate');
export const ShowUpdateFolder=()=>call<void>('ShowUpdateFolder');
