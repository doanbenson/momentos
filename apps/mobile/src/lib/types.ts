/** Shapes aligned with packages/contracts/openapi.yaml */

export type MomentStatus =
  | "intake"
  | "drafting"
  | "rendering"
  | "ready"
  | "failed";

export type AssetType = "photo" | "video" | "voice";

export type SongPick = {
  id: string;
  title: string;
  platformSongId?: string;
};

export type Moment = {
  id: string;
  status: MomentStatus;
  songPick: SongPick;
  assetIds: string[];
  cutId?: string;
  renderUrl?: string;
};

export type CutClip = {
  assetId: string;
  order: number;
  startMs?: number;
  endMs?: number;
};

export type Cut = {
  id: string;
  momentId: string;
  version: number;
  songPickId?: string;
  clips: CutClip[];
};

export type EditorPatch = {
  reorderAssetIds?: string[];
  replace?: { fromAssetId: string; toAssetId: string };
  removeAssetId?: string;
  reDraft?: boolean;
  reRender?: boolean;
};

export type CreateMomentRequest = {
  assetIds: string[];
  songPickId?: string;
};
