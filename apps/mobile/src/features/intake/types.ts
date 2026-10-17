import type { AssetType } from "@/lib/types";

export type IntakeAsset = {
  /** Local-only id until upload registers a server assetId */
  key: string;
  localUri: string;
  type: AssetType;
  mimeType: string;
  /** Set after successful registerLocalAsset */
  assetId?: string;
};

export const MAX_INTAKE_ASSETS = 10;
export const MIN_INTAKE_ASSETS = 1;
