import { useEffect, useRef, useState } from "react";
import { X, Pipette, Plus, Check } from "lucide-react";
import { isHex, hexToHsv, hsvToHex } from "../theme";
import type { HSV } from "../theme";
import ModalPortal from "./ModalPortal";

// clamp01 keeps a normalised drag position inside the control.
const clamp01 = (n: number) => (n < 0 ? 0 : n > 1 ? 1 : n);

// ColorPickerModal is a self-contained picker drawn in the app's own language:
// a saturation/value field, a hue rail, a hex field and quick shades. It
// deliberately avoids <input type="color">, whose popup is the browser's own
// widget and looks nothing like the rest of the client.
export default function ColorPickerModal({
  initial,
  title = "Цвет акцента",
  saved,
  onClose,
  onApply,
  onSave,
  onRemove,
}: {
  initial: string;
  title?: string;
  saved: string[];
  onClose: () => void;
  onApply: (hex: string) => void;
  onSave: (hex: string) => void;
  onRemove: (hex: string) => void;
}) {
  // HSV is authoritative while picking; hex is derived. Going through hex on
  // every drag would lose the hue as soon as the value reached black or white.
  const [hsv, setHsv] = useState(
    () => hexToHsv(initial) ?? { h: 230, s: 0.45, v: 1 },
  );
  const [text, setText] = useState(initial.toLowerCase());

  const current=useRef(hsv),frame=useRef<number|null>(null),pending=useRef<HSV|null>(null);current.current=hsv;
  useEffect(()=>()=>{if(frame.current!==null)cancelAnimationFrame(frame.current)},[]);
  const hex = hsvToHex(hsv.h, hsv.s, hsv.v);
  const typedValid = isHex(text);
  const alreadySaved = saved.includes(hex.toLowerCase());

  const svRef = useRef<HTMLDivElement>(null);
  const hueRef = useRef<HTMLDivElement>(null);

  // Committing through here keeps the hex field in step with the handles.
  const commit = (next: HSV) => {
    current.current=next;pending.current=next;
    if(frame.current===null)frame.current=requestAnimationFrame(()=>{frame.current=null;const value=pending.current;if(value){setHsv(value);setText(hsvToHex(value.h,value.s,value.v))}});
  };

  const pickFromHex = (v: string) => {
    setText(v);
    const parsed = hexToHsv(v);
    if (parsed) setHsv(parsed);
  };

  // Pointer capture means a drag that leaves the control still tracks.
  const drag =
    (
      ref: React.RefObject<HTMLDivElement | null>,
      apply: (x: number, y: number) => void,
    ) =>
    (e: React.PointerEvent) => {
      if (e.type === "pointermove" && e.buttons !== 1) return;
      const el = ref.current;
      if (!el) return;
      if (e.type === "pointerdown") {
        el.setPointerCapture(e.pointerId);
      }
      const r = el.getBoundingClientRect();
      apply(
        clamp01((e.clientX - r.left) / r.width),
        clamp01((e.clientY - r.top) / r.height),
      );
    };

  const onSV = drag(svRef, (x, y) => commit({ ...current.current, s: x, v: 1 - y }));
  const onHue = drag(hueRef, (x) => commit({ ...current.current, h: x * 360 }));

  // Chromium ships an eyedropper; offer it only where it actually exists.
  const eyeDropper = (window as any).EyeDropper;
  const pickFromScreen = async () => {
    try {
      const res = await new eyeDropper().open();
      if (res?.sRGBHex) pickFromHex(res.sRGBHex);
    } catch {
      /* the user dismissed the eyedropper */
    }
  };

  return (
    <ModalPortal label={title} onClose={onClose}>
      <div
        className="fixed inset-0 z-50 grid place-items-center bg-black/60 p-6 backdrop-blur-sm"
        onClick={onClose}
      >
        <div
          className="animate-view w-full max-w-xs rounded-xl border border-border bg-surface p-4 shadow-2xl"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-medium text-text">{title}</h2>
            <button
              onClick={onClose}
              className="no-drag text-text-faint transition hover:text-text"
            >
              <X size={16} />
            </button>
          </div>

          {/* Saturation (x) × value (y) over the current hue */}
          <div
            ref={svRef}
            onPointerDown={onSV}
            onPointerMove={onSV}
            className="relative mb-3 h-40 w-full cursor-crosshair touch-none overflow-hidden rounded-lg border border-border"
            style={{
              background: `linear-gradient(to top, #000, transparent), linear-gradient(to right, #fff, hsl(${hsv.h} 100% 50%))`,
            }}
          >
            <span
              className="tc-color-handle pointer-events-none absolute h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_rgba(0,0,0,0.45)]"
              style={{
                left: `${hsv.s * 100}%`,
                top: `${(1 - hsv.v) * 100}%`,
                background: hex,
              }}
            />
          </div>

          {/* Hue rail */}
          <div
            ref={hueRef}
            onPointerDown={onHue}
            onPointerMove={onHue}
            className="relative mb-3 h-3 w-full cursor-pointer touch-none rounded-full border border-border"
            style={{
              background:
                "linear-gradient(to right,#f00 0%,#ff0 17%,#0f0 33%,#0ff 50%,#00f 67%,#f0f 83%,#f00 100%)",
            }}
          >
            <span
              className="tc-color-handle pointer-events-none absolute top-1/2 h-4 w-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_rgba(0,0,0,0.45)]"
              style={{
                left: `${(hsv.h / 360) * 100}%`,
                background: `hsl(${hsv.h} 100% 50%)`,
              }}
            />
          </div>

          {/* Swatch + hex + eyedropper */}
          <div className="mb-3 flex items-center gap-2">
            <span
              className="h-9 w-9 shrink-0 rounded-lg border border-border"
              style={{ background: hex }}
            />
            <input
              value={text}
              onChange={(e) => pickFromHex(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && typedValid && onApply(hex)}
              placeholder="#ffffff"
              spellCheck={false}
              className={`min-w-0 flex-1 rounded-lg border bg-bg px-3 py-2 font-mono text-sm text-text outline-none transition placeholder:text-text-faint focus:border-accent/60 ${
                text.length > 1 && !typedValid
                  ? "border-danger/50"
                  : "border-border"
              }`}
            />
            {eyeDropper && (
              <button
                onClick={pickFromScreen}
                title="Взять цвет с экрана"
                className="shrink-0 rounded-lg border border-border p-2 text-text-faint transition hover:bg-surface-2 hover:text-text"
              >
                <Pipette size={15} />
              </button>
            )}
          </div>

          {/* The user's own palette. The ready-made shades that used to sit here
            were the accent palette again, one screen up. */}
          <div className="mb-4">
            <div className="mb-1.5 flex items-center justify-between">
              <span className="font-mono text-[11px] uppercase tracking-wide text-text-faint">
                Сохранённые
              </span>
              <button
                onClick={() => onSave(hex)}
                disabled={alreadySaved}
                title={alreadySaved ? "Уже сохранён" : "Сохранить текущий цвет"}
                className="flex items-center gap-1 rounded-md border border-border px-1.5 py-0.5 text-[11px] text-text-muted transition hover:bg-surface-2 hover:text-text disabled:cursor-not-allowed disabled:opacity-40"
              >
                {alreadySaved ? <Check size={11} /> : <Plus size={11} />}
                {alreadySaved ? "Сохранён" : "Сохранить"}
              </button>
            </div>

            {saved.length === 0 ? (
              <div className="rounded-lg border border-dashed border-border px-3 py-2.5 text-center text-[11px] text-text-faint">
                Пока пусто — подберите цвет и нажмите «Сохранить»
              </div>
            ) : (
              <div className="grid grid-cols-8 gap-1.5">
                {saved.map((c) => (
                  <div key={c} className="group relative">
                    <button
                      onClick={() => pickFromHex(c)}
                      title={c}
                      className={`h-6 w-full rounded-md border transition ${
                        hex.toLowerCase() === c
                          ? "border-text"
                          : "border-transparent hover:border-border"
                      }`}
                      style={{ background: c }}
                    />
                    <button
                      onClick={() => onRemove(c)}
                      title="Убрать"
                      className="absolute -right-1 -top-1 hidden h-3.5 w-3.5 place-items-center rounded-full border border-border bg-surface text-text-faint transition hover:text-danger group-hover:grid"
                    >
                      <X size={8} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="flex gap-2">
            <button
              onClick={onClose}
              className="flex-1 rounded-lg border border-border py-2 text-sm text-text-muted transition hover:bg-surface-2"
            >
              Отмена
            </button>
            <button
              onClick={() => onApply(hex)}
              className="flex-1 rounded-lg bg-accent py-2 text-sm font-medium text-on-accent transition hover:bg-accent-soft"
            >
              Применить
            </button>
          </div>
        </div>
      </div>
    </ModalPortal>
  );
}
