import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Alert,
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
import type { CutClip, Moment } from "@/lib/types";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";
import { CutClipStrip } from "./CutClipStrip";
import { EditorToolRow } from "./EditorToolRow";
import { MomentPreviewPlayer } from "./MomentPreviewPlayer";

function sortClips(clips: CutClip[]) {
  return [...clips].sort((a, b) => a.order - b.order);
}

export function EditorScreen() {
  const router = useRouter();
  const { momentId } = useLocalSearchParams<{ momentId?: string }>();
  const [moment, setMoment] = useState<Moment | null>(null);
  const [clips, setClips] = useState<CutClip[]>([]);
  const [selectedAssetId, setSelectedAssetId] = useState<string | null>(null);
  const [dirty, setDirty] = useState(false);
  const [pendingReplace, setPendingReplace] = useState<{
    fromAssetId: string;
    toAssetId: string;
  } | null>(null);
  const [removedIds, setRemovedIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [updating, setUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!momentId) return;
    setLoading(true);
    setError(null);
    try {
      const [m, cut] = await Promise.all([getMoment(momentId), getCut(momentId)]);
      setMoment(m);
      const ordered = sortClips(cut.clips);
      setClips(ordered);
      setSelectedAssetId(ordered[0]?.assetId ?? null);
      setDirty(false);
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
    setDirty(true);
  }

  function removeSelected() {
    if (!selectedAssetId) return;
    const next = orderedIds.filter((id) => id !== selectedAssetId);
    setRemovedIds((prev) => [...prev, selectedAssetId]);
    setClips(clipsInOrder(next));
    setSelectedAssetId(next[0] ?? null);
    setDirty(true);
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
        mimeType:
          picked.mimeType ?? (isVideo ? "video/mp4" : "image/jpeg"),
      });
      setPendingReplace({ fromAssetId: selectedAssetId, toAssetId });
      setClips((prev) =>
        sortClips(prev).map((c) =>
          c.assetId === selectedAssetId ? { ...c, assetId: toAssetId } : c,
        ),
      );
      setSelectedAssetId(toAssetId);
      setDirty(true);
    } catch (e) {
      Alert.alert(
        "Replace blocked",
        e instanceof Error ? e.message : "Asset upload API missing.",
      );
    }
  }

  async function updateMoment() {
    if (!momentId || !dirty) return;
    setUpdating(true);
    try {
      // Batch local edits; only the final PATCH requests reDraft/reRender (GRILL Q4)
      if (pendingReplace) {
        await patchCut(momentId, { replace: pendingReplace });
      }
      for (const removeAssetId of removedIds) {
        await patchCut(momentId, { removeAssetId });
      }
      await patchCut(momentId, {
        reorderAssetIds: orderedIds,
        reDraft: true,
        reRender: true,
      });
      setDirty(false);
      setPendingReplace(null);
      setRemovedIds([]);
      Alert.alert("Updating", "Cut patched — returning to Processing while Render runs.");
      router.replace({
        pathname: "/(processing)",
        params: { momentId },
      });
    } catch (e) {
      Alert.alert(
        "Update failed",
        e instanceof Error ? e.message : String(e),
      );
    } finally {
      setUpdating(false);
    }
  }

  function goShare() {
    if (!momentId) return;
    if (dirty) {
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
    Alert.alert(
      "Leave Editor?",
      "Start another Moment from Week Intake? Unsaved strip edits will be lost.",
      [
        { text: "Stay", style: "cancel" },
        {
          text: "Week Intake",
          style: "destructive",
          onPress: () => router.replace("/(onboarding)"),
        },
      ],
    );
  }

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.header}>
          <GhostButton label="Back" onPress={confirmBack} style={styles.back} />
          <Text style={styles.title}>Editor</Text>
        </View>

        {loading ? <Text style={styles.muted}>Loading Cut…</Text> : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}

        <MomentPreviewPlayer renderUrl={moment?.renderUrl} />
        <SongPickCaption compact />

        <Text style={styles.section}>Cut strip</Text>
        <CutClipStrip
          clips={clips}
          selectedAssetId={selectedAssetId}
          onSelect={setSelectedAssetId}
        />
        <EditorToolRow
          disabled={!selectedAssetId}
          onReorderLeft={() => moveSelected(-1)}
          onReorderRight={() => moveSelected(1)}
          onReplace={() => void replaceSelected()}
          onRemove={removeSelected}
        />

        <BrassButton
          label="Update Moment"
          onPress={() => void updateMoment()}
          disabled={!dirty}
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
  content: {
    padding: space.lg,
    gap: space.md,
    paddingBottom: space.xl,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.md,
  },
  back: { minWidth: 88, minHeight: 40, paddingVertical: space.sm },
  title: { ...typography.title, flex: 1 },
  section: { ...typography.label, marginTop: space.sm },
  muted: { ...typography.caption },
  error: { ...typography.caption, color: colors.error },
  cta: { marginTop: space.sm },
});
