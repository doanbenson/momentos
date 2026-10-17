import { gatewayFetch, readError } from "./api";
import type { AssetType } from "./types";

/**
 * Week Intake Asset registration via gateway → media-service:
 *   POST /v1/assets → PUT signed uploadUrl → POST /v1/assets/{id}/complete
 */

export type LocalAssetDraft = {
  localUri: string;
  type: AssetType;
  mimeType: string;
  byteSize?: number;
};

export type CreateAssetResponse = {
  asset: { id: string; type: AssetType };
  uploadUrl: string;
  uploadHeaders?: Record<string, string>;
};

export async function createAsset(input: {
  type: AssetType;
  mimeType: string;
  byteSize?: number;
}): Promise<CreateAssetResponse> {
  const res = await gatewayFetch("/v1/assets", {
    method: "POST",
    body: JSON.stringify({
      type: input.type,
      mimeType: input.mimeType,
      byteSize: input.byteSize,
    }),
  });
  if (!res.ok) {
    throw new Error(`Asset create failed (${res.status}): ${await readError(res)}`);
  }
  return res.json();
}

export async function uploadAssetBytes(
  uploadUrl: string,
  localUri: string,
  mimeType: string,
  uploadHeaders?: Record<string, string>,
): Promise<void> {
  // TODO: confirm signed-URL PUT contract with media-service
  const blob = await (await fetch(localUri)).blob();
  const res = await fetch(uploadUrl, {
    method: "PUT",
    headers: {
      "Content-Type": mimeType,
      ...(uploadHeaders ?? {}),
    },
    body: blob,
  });
  if (!res.ok) {
    throw new Error(`Asset upload PUT failed (${res.status})`);
  }
}

export async function completeAsset(assetId: string): Promise<void> {
  const res = await gatewayFetch(`/v1/assets/${assetId}/complete`, {
    method: "POST",
    body: JSON.stringify({}),
  });
  if (!res.ok) {
    throw new Error(`Asset complete failed (${res.status}): ${await readError(res)}`);
  }
}

/** Full local → registered Asset pipeline. */
export async function registerLocalAsset(
  draft: LocalAssetDraft,
): Promise<string> {
  const created = await createAsset({
    type: draft.type,
    mimeType: draft.mimeType,
    byteSize: draft.byteSize,
  });
  await uploadAssetBytes(
    created.uploadUrl,
    draft.localUri,
    draft.mimeType,
    created.uploadHeaders,
  );
  await completeAsset(created.asset.id);
  return created.asset.id;
}
