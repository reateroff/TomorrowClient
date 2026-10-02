import {useEffect,useState} from 'react';
import {ArrowDownToLine,X} from 'lucide-react';
import {GetUpdateInfo,DownloadUpdate,ShowUpdateFolder} from '../backend';
import type {UpdateInfo} from '../backend';
import {EventsOn} from '../../wailsjs/runtime/runtime';
import {push} from './Toasts';
export default function UpdateBanner(){
 const [info,setInfo]=useState<UpdateInfo|null>(null),[dismissed,setDismissed]=useState(''),[busy,setBusy]=useState(false);
 useEffect(()=>{GetUpdateInfo().then(setInfo).catch(()=>{});return EventsOn('app:update',(u:UpdateInfo)=>setInfo(u))},[]);
 if(!info?.available||dismissed===info.version)return null;
 const action=async()=>{setBusy(true);try{info.downloadPath?await ShowUpdateFolder():await DownloadUpdate()}catch(e){push(String(e),'error')}finally{setBusy(false)}};
 return <div className="absolute right-4 top-4 z-20 max-w-xs rounded-lg border border-accent/40 bg-surface p-3 shadow-lg"><div className="flex items-center gap-2 text-xs"><ArrowDownToLine size={14} className="text-accent"/><span className="flex-1">Доступна {info.version}</span><button aria-label="Скрыть уведомление" onClick={()=>setDismissed(info.version)}><X size={13}/></button></div><button disabled={busy||info.downloading||!info.assetUrl} className="mt-2 w-full rounded-md bg-accent px-3 py-1.5 text-xs text-on-accent disabled:opacity-50" onClick={action}>{info.downloading?`Скачивание ${info.progress}%`:info.downloadPath?'Открыть скачанное обновление':'Скачать обновление'}</button>{info.error&&<p className="mt-2 text-[10px] text-danger">{info.error}</p>}<p className="mt-2 text-[10px] text-text-faint">Установка вручную. Приложение не заменяется в фоне.</p></div>
}
