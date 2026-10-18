import * as FileSystem from "expo-file-system/legacy";
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
  /** Required — media-service rejects missing/zero (defaults to 0 in JSON omit). */
  byteSize: number;
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
  // RN Blob has no arrayBuffer(); Expo fetch also overwrites Content-Type from
  // Blob.type. Native uploadAsync streams the file:// URI with our headers intact.
  const headers: Record<string, string> = {
    ...(uploadHeaders ?? {}),
    "Content-Type": mimeType,
  };
  const result = await FileSystem.uploadAsync(uploadUrl, localUri, {
    httpMethod: "PUT",
    uploadType: FileSystem.FileSystemUploadType.BINARY_CONTENT,
    headers,
  });
  if (result.status < 200 || result.status >= 300) {
    throw new Error(`Asset upload PUT failed (${result.status})`);
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

async function resolveByteSize(
  localUri: string,
  known?: number,
): Promise<number> {
  if (known != null && known > 0) return known;
  const info = await FileSystem.getInfoAsync(localUri);
  if (info.exists && !info.isDirectory && typeof info.size === "number" && info.size > 0) {
    return info.size;
  }
  const blob = await (await fetch(localUri)).blob();
  if (blob.size <= 0) {
    throw new Error("Could not determine Asset byteSize from local file");
  }
  return blob.size;
}

/** Full local → registered Asset pipeline. */
export async function registerLocalAsset(
  draft: LocalAssetDraft,
): Promise<string> {
  const byteSize = await resolveByteSize(draft.localUri, draft.byteSize);
  const created = await createAsset({
    type: draft.type,
    mimeType: draft.mimeType,
    byteSize,
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
