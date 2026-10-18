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
  reflectionAssetId?: string;
  reflectionQuestion?: string;
  cutId?: string;
  renderUrl?: string;
};

export type CutClipRole = "reflection" | "montage" | "hero_outro";

export type CutClip = {
  assetId: string;
  order: number;
  startMs?: number;
  endMs?: number;
  role: CutClipRole;
};

export type CutCaption = {
  text: string;
  startMs: number;
  endMs: number;
};

export type CutMarkerType = "j_cut" | "chorus_drop" | "flash" | "sfx";

export type CutMarker = {
  type: CutMarkerType;
  atMs: number;
  payload?: Record<string, unknown>;
};

export type Cut = {
  id: string;
  momentId: string;
  version: number;
  songPickId?: string;
  clips: CutClip[];
  captions?: CutCaption[];
  markers?: CutMarker[];
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
  reflectionAssetId: string;
  reflectionQuestion: string;
  songPickId?: string;
};
