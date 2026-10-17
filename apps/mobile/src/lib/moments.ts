import { gatewayFetch, readError } from "./api";
import type { CreateMomentRequest, Cut, EditorPatch, Moment } from "./types";

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
