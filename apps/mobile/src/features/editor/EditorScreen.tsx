import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useLocalSearchParams, useRouter } from "expo-router";
import * as ImagePicker from "expo-image-picker";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrassButton } from "@/components/BrassButton";
import { GhostButton } from "@/components/GhostButton";
import { SongPickCaption } from "@/components/SongPickCaption";
import { registerLocalAsset } from "@/lib/assets";
import { getCut, getMoment, patchCut } from "@/lib/moments";
import type { CutClip, EditorTrim, Moment } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";
import { statusCopy } from "@/features/moment/useMomentPoll";
import { CutClipStrip } from "./CutClipStrip";
import { EditorToolRow } from "./EditorToolRow";
import { MomentPreviewPlayer } from "./MomentPreviewPlayer";
import { TrimControl } from "./TrimControl";

type SourceWindow = {
  sourceInMs?: number;
  sourceOutMs?: number;
};

function sortClips(clips: CutClip[]) {
  return [...clips].sort((a, b) => a.order - b.order);
}

function sameIds(a: string[], b: string[]) {
  return a.length === b.length && a.every((id, index) => id === b[index]);
}

function windowSnap(clip: CutClip): SourceWindow {
  return { sourceInMs: clip.sourceInMs, sourceOutMs: clip.sourceOutMs };
}

export function EditorScreen() {
  const router = useRouter();
  const { momentId } = useLocalSearchParams<{ momentId?: string }>();
  const [moment, setMoment] = useState<Moment | null>(null);
  const [clips, setClips] = useState<CutClip[]>([]);
  const [baselineOrder, setBaselineOrder] = useState<string[]>([]);
  const [baselineWindows, setBaselineWindows] = useState<
    Record<string, SourceWindow>
  >({});
  const [selectedAssetId, setSelectedAssetId] = useState<string | null>(null);
  const [pendingReplace, setPendingReplace] = useState<{
    fromAssetId: string;
    toAssetId: string;
  } | null>(null);
  const [removedIds, setRemovedIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!momentId) {
      setLoading(false);
      setError("Missing momentId.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [m, cut] = await Promise.all([getMoment(momentId), getCut(momentId)]);
      const ordered = sortClips(cut.clips);
      setMoment(m);
      setClips(ordered);
      setBaselineOrder(ordered.map((clip) => clip.assetId));
      setBaselineWindows(
        Object.fromEntries(ordered.map((clip) => [clip.assetId, windowSnap(clip)])),
      );
      setSelectedAssetId(ordered[0]?.assetId ?? null);
      setPendingReplace(null);
      setRemovedIds([]);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [momentId]);

  useEffect(() => {
    void load();
  }, [load]);

  const orderedIds = useMemo(
    () => sortClips(clips).map((c) => c.assetId),
    [clips],
  );

  const selectedClip =
    clips.find((clip) => clip.assetId === selectedAssetId) ?? null;

  function sourceWindowChanged(clip: CutClip) {
    const base = baselineWindows[clip.assetId];
    if (!base) {
      return clip.sourceInMs != null && clip.sourceOutMs != null;
    }
    return (
      clip.sourceInMs !== base.sourceInMs || clip.sourceOutMs !== base.sourceOutMs
    );
  }

  const trims: EditorTrim[] = sortClips(clips).flatMap((clip) => {
    if (!sourceWindowChanged(clip)) return [];
    if (clip.sourceInMs == null || clip.sourceOutMs == null) return [];
    if (clip.sourceOutMs <= clip.sourceInMs) return [];
    return [
      {
        assetId: clip.assetId,
        sourceInMs: clip.sourceInMs,
        sourceOutMs: clip.sourceOutMs,
      },
    ];
  });

  const canUpdate =
    !loading &&
    (pendingReplace != null ||
      removedIds.length > 0 ||
      !sameIds(orderedIds, baselineOrder) ||
      trims.length > 0);

  /** Preserve act / role / motion / look / transitionIn on local reorder. */
  function clipsInOrder(ids: string[]): CutClip[] {
    const byId = new Map(clips.map((c) => [c.assetId, c]));
    return ids.map((assetId, order) => {
      const prev = byId.get(assetId);
      if (!prev) {
        throw new Error(`Cut clip missing for Asset ${assetId}`);
      }
      return { ...prev, order };
    });
  }

  function moveSelected(dir: -1 | 1) {
    if (!selectedAssetId) return;
    const ids = [...orderedIds];
    const idx = ids.indexOf(selectedAssetId);
    const next = idx + dir;
    if (idx < 0 || next < 0 || next >= ids.length) return;
    [ids[idx], ids[next]] = [ids[next], ids[idx]];
    setClips(clipsInOrder(ids));
  }

  function removeSelected() {
    if (!selectedAssetId) return;
    const next = orderedIds.filter((id) => id !== selectedAssetId);
    setRemovedIds((prev) => [...prev, selectedAssetId]);
    setClips(clipsInOrder(next));
    setSelectedAssetId(next[0] ?? null);
  }

  function applyTrim(next: { sourceInMs: number; sourceOutMs: number }) {
    if (!selectedAssetId) return;
    if (next.sourceOutMs <= next.sourceInMs) return;
    setClips((prev) =>
      prev.map((clip) =>
        clip.assetId === selectedAssetId ? { ...clip, ...next } : clip,
      ),
    );
  }

  async function replaceSelected() {
    if (!selectedAssetId) return;
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      Alert.alert("Library", "Permission required to replace an Asset.");
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images", "videos"],
      quality: 0.9,
      allowsMultipleSelection: false,
    });
    if (result.canceled || !result.assets[0]) return;
    const picked = result.assets[0];
    const isVideo = picked.type === "video";
    try {
      const toAssetId = await registerLocalAsset({
        localUri: picked.uri,
        type: isVideo ? "video" : "photo",
        mimeType: picked.mimeType ?? (isVideo ? "video/mp4" : "image/jpeg"),
      });
      const fromAssetId = selectedAssetId;
      setPendingReplace({ fromAssetId, toAssetId });
      setClips((prev) =>
        sortClips(prev).map((c) =>
          c.assetId === fromAssetId ? { ...c, assetId: toAssetId } : c,
        ),
      );
      setBaselineWindows((prev) => {
        const copied = prev[fromAssetId] ?? {};
        return { ...prev, [toAssetId]: copied };
      });
      setSelectedAssetId(toAssetId);
    } catch (e) {
      Alert.alert(
        "Replace blocked",
        e instanceof Error ? e.message : "Asset upload API missing.",
      );
    }
  }

  async function updateMoment() {
    if (!momentId || !canUpdate) return;
    setUpdating(true);
    try {
      if (pendingReplace) {
        await patchCut(momentId, { replace: pendingReplace });
      }
      for (const removeAssetId of removedIds) {
        await patchCut(momentId, { removeAssetId });
      }
      await patchCut(momentId, {
        reorderAssetIds: orderedIds,
        ...(trims.length > 0 ? { trims } : {}),
        reRender: true,
      });
      setPendingReplace(null);
      setRemovedIds([]);
      Alert.alert(
        "Updating",
        "Cut patched — returning to Processing while Render runs.",
      );
      router.replace({
        pathname: "/(processing)",
        params: { momentId },
      });
    } catch (e) {
      Alert.alert("Update failed", e instanceof Error ? e.message : String(e));
    } finally {
      setUpdating(false);
    }
  }

  function goShare() {
    if (!momentId) return;
    if (canUpdate) {
      Alert.alert(
        "Unsaved changes",
        "Update Moment before sharing, or discard local edits?",
        [
          { text: "Cancel", style: "cancel" },
          {
            text: "Discard",
            style: "destructive",
            onPress: () =>
              router.push({ pathname: "/(share)", params: { momentId } }),
          },
          { text: "Update Moment", onPress: () => void updateMoment() },
        ],
      );
      return;
    }
    router.push({ pathname: "/(share)", params: { momentId } });
  }

  function confirmBack() {
    const leave = () => router.replace("/(home)");
    if (!canUpdate) {
      leave();
      return;
    }
    Alert.alert("Leave Editor?", "Unsaved Cut edits will be lost.", [
      { text: "Stay", style: "cancel" },
      { text: "Moments", style: "destructive", onPress: leave },
    ]);
  }

  const meta = moment ? statusCopy(moment.status).title : "Editor";

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
      <View style={styles.header}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Back to Moments"
          onPress={confirmBack}
          hitSlop={8}
          style={styles.back}
        >
          <Text style={styles.backChevron}>‹</Text>
          <Text style={styles.backLabel}>Moments</Text>
        </Pressable>
        <View style={styles.headerCenter}>
          <Text style={styles.headerTitle} numberOfLines={1}>
            Moment
          </Text>
          <Text style={styles.headerMeta}>{meta}</Text>
        </View>
        <View style={styles.headerSpacer} />
      </View>

      <ScrollView
        contentContainerStyle={styles.content}
        keyboardShouldPersistTaps="handled"
      >
        {loading ? <Text style={styles.muted}>Loading Cut…</Text> : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}

        <MomentPreviewPlayer renderUrl={moment?.renderUrl} />

        <View style={styles.track}>
          <SongPickCaption compact />
        </View>

        <View style={styles.sectionHead}>
          <Text style={styles.section}>Cut</Text>
          <Text style={styles.sectionHint}>Durations</Text>
        </View>
        <CutClipStrip
          clips={clips}
          selectedAssetId={selectedAssetId}
          onSelect={setSelectedAssetId}
        />

        {selectedClip ? (
          <TrimControl clip={selectedClip} onChange={applyTrim} />
        ) : (
          <Text style={styles.muted}>Select an Asset to trim its source window.</Text>
        )}

        <EditorToolRow
          disabled={!selectedAssetId || updating}
          onReorderLeft={() => moveSelected(-1)}
          onReorderRight={() => moveSelected(1)}
          onReplace={() => void replaceSelected()}
          onRemove={removeSelected}
        />

        <BrassButton
          label="Update Moment"
          onPress={() => void updateMoment()}
          disabled={!canUpdate}
          loading={updating}
          style={styles.cta}
        />
        <GhostButton label="Done" onPress={goShare} />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  header: {
    flexDirection: "row",
    alignItems: "center",
    paddingHorizontal: space.md,
    minHeight: 56,
    borderBottomWidth: 1,
    borderBottomColor: colors.outline,
    gap: space.sm,
  },
  back: {
    flexDirection: "row",
    alignItems: "center",
    minHeight: 44,
    minWidth: 88,
    paddingRight: space.sm,
  },
  backChevron: {
    fontFamily: fonts.headlineRegular,
    fontSize: 22,
    color: colors.onSurfaceMuted,
    marginRight: 2,
  },
  backLabel: {
    fontFamily: fonts.bodyMedium,
    fontSize: 14,
    color: colors.onSurface,
  },
  headerCenter: {
    flex: 1,
    alignItems: "center",
  },
  headerTitle: {
    fontFamily: fonts.headlineRegular,
    fontSize: 14,
    color: colors.onSurface,
  },
  headerMeta: {
    fontFamily: fonts.label,
    fontSize: 10,
    letterSpacing: 0.6,
    color: colors.onSurfaceMuted,
  },
  headerSpacer: { width: 88 },
  content: {
    padding: space.md,
    gap: space.md,
    paddingBottom: space.xl,
  },
  track: {
    backgroundColor: colors.surface,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.outline,
    paddingHorizontal: space.md,
    paddingVertical: space.sm,
  },
  sectionHead: {
    flexDirection: "row",
    alignItems: "baseline",
    justifyContent: "space-between",
  },
  section: {
    fontFamily: fonts.headlineRegular,
    fontSize: 16,
    color: colors.onSurface,
  },
  sectionHint: {
    ...typography.label,
  },
  muted: { ...typography.caption },
  error: { ...typography.caption, color: colors.error },
  cta: { marginTop: space.sm },
});
