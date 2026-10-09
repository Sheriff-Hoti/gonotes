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
  /** Load the list from Go. */
  async refresh(): Promise<void> {
    const list = await listNotes();
    const parsed = z.array(savedItemSchema).safeParse(list ?? []);
    items = parsed.success ? newestFirst(parsed.data) : [];
    emit();
  },
  /** Insert a new card via Go and prepend it (newest first). Real IDs and
   * timestamps always come back from Go; the frontend never mints them. */
  async create(data: { intent: SavedItem["intent"]; summary: string; text: string }): Promise<SavedItem> {
    const stored = await saveNote({ id: "", createdAt: 0, ...data, intent: data.intent as BackendItem["intent"] });
    const parsed = savedItemSchema.parse(stored);
    items = [parsed, ...items];
    emit();
    return parsed;
  },
  /** Update one card via Go, keeping its position in the list. */
  async update(id: string, data: { intent: SavedItem["intent"]; summary: string; text: string }): Promise<SavedItem> {
    const prev = items.find((x) => x.id === id);
    const stored = await saveNote({ id, createdAt: prev?.createdAt ?? 0, ...data, intent: data.intent as BackendItem["intent"] });
    const parsed = savedItemSchema.parse(stored);
    items = items.map((x) => (x.id === id ? parsed : x));
    emit();
    return parsed;
  },
  /** Delete one card via Go. Unknown IDs are not an error. */
  async remove(id: string): Promise<void> {
    await deleteNote(id);
    items = items.filter((x) => x.id !== id);
    emit();
  },
  /**
   * Re-insert a deleted card (Undo). Go cannot reuse a deleted ID, so this
   * saves a fresh copy and splices it back where it was.
   */
  async restoreAt(item: SavedItem, index: number): Promise<SavedItem> {
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
};

let draftCounter = 0;
/** Client-only animation key for the unsaved draft (never persisted). */
export function newDraftKey(): string {
  draftCounter += 1;
  return `draft-${Date.now()}-${draftCounter}`;
}
