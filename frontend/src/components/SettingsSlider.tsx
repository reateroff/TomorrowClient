import {useEffect,useRef,useState} from "react";
export default function SettingsSlider({label,value,min,max,step=1,unit="",format,onCommit}:{label:string;value:number;min:number;max:number;step?:number;unit?:string;format?:(value:number)=>string;onCommit:(value:number)=>void}) {
 const [draft,setDraft]=useState(value);const held=useRef(false),dirty=useRef(false),last=useRef(value),input=useRef<HTMLInputElement>(null),frame=useRef<number|null>(null);
 useEffect(()=>{if(!held.current){setDraft(value);last.current=value;dirty.current=false}},[value]);
 useEffect(()=>()=>{if(frame.current!==null)cancelAnimationFrame(frame.current)},[]);
 const commit=(next:number)=>{held.current=false;if(!dirty.current&&next===last.current)return;dirty.current=false;last.current=next;
  let scroll=input.current?.parentElement;while(scroll&&!/(auto|scroll)/.test(getComputedStyle(scroll).overflowY))scroll=scroll.parentElement;
  const top=input.current?.getBoundingClientRect().top??0;onCommit(next);
  if(frame.current!==null)cancelAnimationFrame(frame.current);frame.current=requestAnimationFrame(()=>{frame.current=null;if(scroll&&input.current)scroll.scrollTop+=input.current.getBoundingClientRect().top-top});
 };
 return <label className="flex items-center gap-4 text-xs text-text-muted"><span className="w-24 shrink-0">{label}</span><input ref={input} aria-label={label} className="min-w-0 flex-1" type="range" min={min} max={max} step={step} value={draft} onPointerDown={()=>held.current=true} onChange={e=>{dirty.current=true;setDraft(e.currentTarget.valueAsNumber)}} onPointerUp={e=>commit(e.currentTarget.valueAsNumber)} onPointerCancel={()=>{held.current=false;dirty.current=false;setDraft(value)}} onKeyUp={e=>commit(e.currentTarget.valueAsNumber)} onBlur={e=>{if(dirty.current)commit(e.currentTarget.valueAsNumber)}}/><output className="w-20 shrink-0 text-right font-mono tabular-nums">{format?format(draft):Number(draft.toFixed(2))}{unit}</output></label>
}
