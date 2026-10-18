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

/** Row from GET /v1/moments. Newest first when listed on Home. */
export type MomentListItem = {
  id: string;
  status: MomentStatus;
  createdAt: string;
  songPickId: string;
  renderUrl?: string;
};

export type CutClipRole = "reflection" | "montage" | "hero_outro";

export type CutClipAct = 1 | 2 | 3;

export type CutMotionPreset =
  | "hook_slow_push"
  | "whip_punch_in"
  | "handheld_super8_drift"
  | "native_action_crop";

export type CutVisualStyle = "clean_916" | "super8_gate";

export type CutTransitionIn = "none" | "hard_cut" | "white_flash";

export type CutClip = {
  assetId: string;
  order: number;
  startMs?: number;
  endMs?: number;
  /** Inclusive file offset in ms. Photos are not file seeks. */
  sourceInMs?: number;
  /** Exclusive file offset in ms. Duration is sourceOutMs - sourceInMs when both are set. */
  sourceOutMs?: number;
  role: CutClipRole;
  act: CutClipAct;
  /** Required on picture clips; omit for voice-only Reflection slate. */
  motionPreset?: CutMotionPreset;
  visualStyle: CutVisualStyle;
  transitionIn: CutTransitionIn;
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

export type EditorTrim = {
  assetId: string;
  sourceInMs: number;
  sourceOutMs: number;
};

export type EditorPatch = {
  reorderAssetIds?: string[];
  replace?: { fromAssetId: string; toAssetId: string };
  removeAssetId?: string;
  trims?: EditorTrim[];
  reDraft?: boolean;
  reRender?: boolean;
};

export type CreateMomentRequest = {
  /** Including Reflection; OpenAPI maxItems 20 (ADR 0016). */
  assetIds: string[];
  reflectionAssetId: string;
  reflectionQuestion: string;
  songPickId?: string;
};
