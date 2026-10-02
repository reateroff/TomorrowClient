import {useEffect,useRef} from 'react';
import {createPortal} from 'react-dom';
import type {ReactNode} from 'react';
// All dialogs are rooted in body, outside animated/transformed view containers.
// Only the topmost dialog handles Escape/Tab; nested process pickers stay safe.
export default function ModalPortal({children,onClose,label}:{children:ReactNode;onClose:()=>void;label?:string}){
 const ref=useRef<HTMLDivElement>(null);const close=useRef(onClose);close.current=onClose;
 useEffect(()=>{const previous=document.activeElement as HTMLElement|null;const focusable=()=>Array.from(ref.current?.querySelectorAll<HTMLElement>('button:not(:disabled),input:not(:disabled),textarea:not(:disabled),select:not(:disabled),[tabindex="0"]')??[]).filter(e=>e.getClientRects().length>0);
 const frame=requestAnimationFrame(()=>{if(!ref.current?.contains(document.activeElement))focusable()[0]?.focus()});
 const key=(e:KeyboardEvent)=>{const all=document.querySelectorAll('[data-tc-modal]');if(all[all.length-1]!==ref.current)return;if(e.key==='Escape'){e.preventDefault();e.stopImmediatePropagation();if(document.querySelector("[data-tc-select]"))window.dispatchEvent(new Event("tc:close-popups"));else if(document.querySelector('[role="tooltip"]'))window.dispatchEvent(new Event('tc:hide-tooltips'));else close.current()}if(e.key==='Tab'){const f=focusable();if(!f.length)return;const at=f.indexOf(document.activeElement as HTMLElement);if(e.shiftKey&&(at<=0)){e.preventDefault();f[f.length-1].focus()}else if(!e.shiftKey&&(at===f.length-1||at<0)){e.preventDefault();f[0].focus()}}};
 document.addEventListener('keydown',key,true);return()=>{cancelAnimationFrame(frame);document.removeEventListener('keydown',key,true);if(previous?.isConnected)previous.focus()};},[]);
 return createPortal(<div ref={ref} data-tc-modal role="dialog" aria-modal="true" aria-label={label} className="contents">{children}</div>,document.body)
}
