import { z } from "zod";
import { INTENT_KEYS } from "@/lib/jev/types";
import { Delete as deleteNote, List as listNotes, Save as saveNote } from "../../bindings/changeme/notesservice.js";
import type { SavedItem as BackendItem } from "../../bindings/changeme/models.js";

export const savedItemSchema = z.object({
  id: z.string(),
  intent: z.enum(INTENT_KEYS).exclude(["none"]),
  summary: z.string(),
  text: z.string(),
  createdAt: z.number(),
});
export type SavedItem = z.infer<typeof savedItemSchema>;

const EMPTY: SavedItem[] = [];
let items: SavedItem[] = [];
/** Demo mode records into memory only, so it never touches the user's saved list. */
let ephemeral = false;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

function newestFirst(list: SavedItem[]): SavedItem[] {
  return [...list].sort((a, b) => b.createdAt - a.createdAt || (a.id < b.id ? 1 : -1));
}

export const savedItems = {
  subscribe(listener: () => void) {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
  getSnapshot(): SavedItem[] {
    return items;
  },
  getServerSnapshot(): SavedItem[] {
    return EMPTY;
  },
  /** Load the list from Go. No-op in demo (ephemeral) mode. */
  async refresh(): Promise<void> {
    if (ephemeral) return;
    const list = await listNotes();
    const parsed = z.array(savedItemSchema).safeParse(list ?? []);
    items = parsed.success ? newestFirst(parsed.data) : [];
    emit();
  },
  /** Insert a new card via Go and prepend it (newest first). Real IDs and
   * timestamps always come back from Go; the frontend never mints them. */
  async create(data: { intent: SavedItem["intent"]; summary: string; text: string }): Promise<SavedItem> {
    if (ephemeral) {
      const item: SavedItem = { ...data, id: newEphemeralId(), createdAt: Date.now() };
      items = [item, ...items].slice(0, 9);
      emit();
      return item;
    }
    const stored = await saveNote({ id: "", createdAt: 0, ...data, intent: data.intent as BackendItem["intent"] });
    const parsed = savedItemSchema.parse(stored);
    items = [parsed, ...items];
    emit();
    return parsed;
  },
  /** Update one card via Go, keeping its position in the list. */
  async update(id: string, data: { intent: SavedItem["intent"]; summary: string; text: string }): Promise<SavedItem> {
    if (ephemeral) {
      let next: SavedItem | undefined;
      items = items.map((x) => {
        if (x.id !== id) return x;
        next = { ...x, ...data };
        return next;
      });
      if (!next) throw new Error(`unknown id ${id}`);
      emit();
      return next;
    }
    const prev = items.find((x) => x.id === id);
    const stored = await saveNote({ id, createdAt: prev?.createdAt ?? 0, ...data, intent: data.intent as BackendItem["intent"] });
    const parsed = savedItemSchema.parse(stored);
    items = items.map((x) => (x.id === id ? parsed : x));
    emit();
    return parsed;
  },
  /** Delete one card via Go. Unknown IDs are not an error. */
  async remove(id: string): Promise<void> {
    if (ephemeral) {
      items = items.filter((x) => x.id !== id);
      emit();
      return;
    }
    await deleteNote(id);
    items = items.filter((x) => x.id !== id);
    emit();
  },
  /**
   * Re-insert a deleted card (Undo). Go cannot reuse a deleted ID, so this
   * saves a fresh copy and splices it back where it was.
   */
  async restoreAt(item: SavedItem, index: number): Promise<SavedItem> {
    if (ephemeral) {
      if (items.some((x) => x.id === item.id)) return item;
      const next = [...items];
      next.splice(Math.min(index, next.length), 0, item);
      items = next.slice(0, 9);
      emit();
      return item;
    }
    const stored = await saveNote({
      id: "",
      createdAt: 0,
      intent: item.intent as BackendItem["intent"],
      summary: item.summary,
      text: item.text,
    });
    const parsed = savedItemSchema.parse(stored);
    const next = [...items];
    next.splice(Math.min(index, next.length), 0, parsed);
    items = next;
    emit();
    return parsed;
  },
  setEphemeral(on: boolean) {
    if (ephemeral === on) return;
    ephemeral = on;
    items = [];
    emit();
  },
  isEphemeral(): boolean {
    return ephemeral;
  },
};

let draftCounter = 0;
/** Client-only animation key for the unsaved draft (never persisted). */
export function newDraftKey(): string {
  draftCounter += 1;
  return `draft-${Date.now()}-${draftCounter}`;
}

let ephemeralCounter = 0;
/**
 * Demo-only placeholder ID (`?demo=1` never calls Go, so there is no stored
 * row to own the ID). Never sent to the backend and never mistaken for a
 * Go-minted UUIDv7.
 */
function newEphemeralId(): string {
  ephemeralCounter += 1;
  return `demo-${Date.now()}-${ephemeralCounter}`;
}
