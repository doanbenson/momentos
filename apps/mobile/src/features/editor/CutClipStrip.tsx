import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import type { CutClip } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { fonts } from "@/theme/typography";

type Props = {
  clips: CutClip[];
  selectedAssetId: string | null;
  onSelect: (assetId: string) => void;
};

export function CutClipStrip({ clips, selectedAssetId, onSelect }: Props) {
  const ordered = [...clips].sort((a, b) => a.order - b.order);

  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.row}
    >
      {ordered.map((clip, index) => {
        const selected = clip.assetId === selectedAssetId;
        return (
          <Pressable
            key={`${clip.assetId}-${clip.order}`}
            onPress={() => onSelect(clip.assetId)}
            style={[styles.tile, selected && styles.tileSelected]}
          >
            <Text style={styles.order}>{index + 1}</Text>
            <Text style={styles.id} numberOfLines={1}>
              {clip.assetId.slice(0, 8)}
            </Text>
          </Pressable>
        );
      })}
      {ordered.length === 0 ? (
        <Text style={styles.empty}>No clips in Cut yet</Text>
      ) : null}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  row: {
    gap: space.sm,
    paddingVertical: space.sm,
    minHeight: 72,
    alignItems: "center",
  },
  tile: {
    width: 72,
    height: 72,
    borderRadius: radius.sm,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
    gap: 4,
  },
  tileSelected: {
    borderColor: colors.primary,
    borderWidth: 2,
  },
  order: {
    fontFamily: fonts.labelBold,
    fontSize: 16,
    color: colors.primary,
  },
  id: {
    fontFamily: fonts.label,
    fontSize: 10,
    color: colors.onSurfaceMuted,
    paddingHorizontal: 4,
  },
  empty: {
    fontFamily: fonts.body,
    color: colors.onSurfaceMuted,
  },
});
