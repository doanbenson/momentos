import { gatewayFetch, readError } from "./api";
import type {
  CreateMomentRequest,
  Cut,
  EditorPatch,
  Moment,
  MomentListItem,
  MomentStatus,
} from "./types";

const MOMENT_STATUSES: readonly MomentStatus[] = [
  "intake",
  "drafting",
  "rendering",
  "ready",
  "failed",
];

function isMomentStatus(value: string): value is MomentStatus {
  return (MOMENT_STATUSES as readonly string[]).includes(value);
}

function songPickIdFrom(row: Record<string, unknown>): string {
  if (typeof row.songPickId === "string") return row.songPickId;
  const songPick = row.songPick;
  if (songPick && typeof songPick === "object" && "id" in songPick) {
    const id = (songPick as { id?: unknown }).id;
    if (typeof id === "string") return id;
  }
  return "";
}

function parseMomentListItem(raw: unknown): MomentListItem | null {
  if (!raw || typeof raw !== "object") return null;
  const row = raw as Record<string, unknown>;
  if (typeof row.id !== "string") return null;
  if (typeof row.status !== "string" || !isMomentStatus(row.status)) return null;
  return {
    id: row.id,
    status: row.status,
    createdAt: typeof row.createdAt === "string" ? row.createdAt : "",
    songPickId: songPickIdFrom(row),
    renderUrl: typeof row.renderUrl === "string" ? row.renderUrl : undefined,
  };
}

function unwrapMomentList(body: unknown): unknown[] {
  if (Array.isArray(body)) return body;
  if (body && typeof body === "object") {
    const record = body as Record<string, unknown>;
    if (Array.isArray(record.moments)) return record.moments;
    if (Array.isArray(record.items)) return record.items;
  }
  throw new Error("listMoments returned an unexpected payload");
}

function createdAtMillis(iso: string): number {
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
}

/** Moments for the signed-in user, newest first. */
export async function listMoments(): Promise<MomentListItem[]> {
  const res = await gatewayFetch("/v1/moments");
  if (!res.ok) {
    throw new Error(`listMoments failed (${res.status}): ${await readError(res)}`);
  }
  const rows = unwrapMomentList(await res.json());
  return rows
    .map(parseMomentListItem)
    .filter((item): item is MomentListItem => item != null)
    .sort((a, b) => createdAtMillis(b.createdAt) - createdAtMillis(a.createdAt));
}

export async function createMoment(body: CreateMomentRequest): Promise<Moment> {
  const res = await gatewayFetch("/v1/moments", {
    method: "POST",
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(`createMoment failed (${res.status}): ${await readError(res)}`);
  }
  return res.json();
}

export async function getMoment(momentId: string): Promise<Moment> {
  const res = await gatewayFetch(`/v1/moments/${momentId}`);
  if (!res.ok) {
    throw new Error(`getMoment failed (${res.status}): ${await readError(res)}`);
  }
  return res.json();
}

export async function getCut(momentId: string): Promise<Cut> {
  const res = await gatewayFetch(`/v1/moments/${momentId}/cut`);
  if (!res.ok) {
    throw new Error(`getCut failed (${res.status}): ${await readError(res)}`);
  }
  return res.json();
}

export async function patchCut(
  momentId: string,
  patch: EditorPatch,
): Promise<Cut> {
  const res = await gatewayFetch(`/v1/moments/${momentId}/cut`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
  if (!res.ok) {
    throw new Error(`patchCut failed (${res.status}): ${await readError(res)}`);
  }
  return res.json();
}
