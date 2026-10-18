import type { AssetType } from "@/lib/types";

export type IntakeAsset = {
  /** Local-only id until upload registers a server assetId */
  key: string;
  localUri: string;
  type: AssetType;
  mimeType: string;
  /** Exactly one intake Asset should be the Reflection (video or voice). */
  isReflection?: boolean;
  /** Set after successful registerLocalAsset */
  assetId?: string;
};

export const MAX_INTAKE_ASSETS = 10;
export const MIN_INTAKE_ASSETS = 1;

export function canBeReflection(type: AssetType): boolean {
  return type === "video" || type === "voice";
}

export function reflectionAsset(assets: IntakeAsset[]): IntakeAsset | undefined {
  return assets.find((a) => a.isReflection);
}
