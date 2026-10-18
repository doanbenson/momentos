import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import type { CutClip, CutClipRole } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { fonts } from "@/theme/typography";
import { clipDurationMs, formatDurationMs } from "./clipWindow";

type Props = {
  clips: CutClip[];
  selectedAssetId: string | null;
  onSelect: (assetId: string) => void;
};

const ROLE_LABEL: Record<CutClipRole, string> = {
  reflection: "REFLECTION",
  montage: "MONTAGE",
  hero_outro: "HERO",
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
        const duration = clipDurationMs(clip);
        const durationLabel = duration != null ? formatDurationMs(duration) : "—";
        return (
          <Pressable
            key={`${clip.assetId}-${clip.order}`}
            accessibilityRole="button"
            accessibilityState={{ selected }}
            accessibilityLabel={`Clip ${index + 1}, ${ROLE_LABEL[clip.role]}, ${durationLabel}`}
            onPress={() => onSelect(clip.assetId)}
            style={[styles.tile, selected && styles.tileSelected]}
          >
            {selected ? <View style={styles.playhead} /> : null}
            <Text style={[styles.order, selected && styles.orderSelected]}>
              {index + 1}
            </Text>
            <Text style={[styles.duration, selected && styles.durationSelected]}>
              {durationLabel}
            </Text>
            <View style={[styles.badge, selected && styles.badgeSelected]}>
              <Text style={styles.badgeText} numberOfLines={1}>
                {ROLE_LABEL[clip.role]}
              </Text>
            </View>
          </Pressable>
        );
      })}
      {ordered.length === 0 ? (
        <Text style={styles.empty}>No clips in this Cut yet</Text>
      ) : null}
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  row: {
    gap: space.sm,
    paddingVertical: space.sm,
    minHeight: 108,
    alignItems: "center",
  },
  tile: {
    width: 84,
    height: 104,
    borderRadius: radius.md,
    backgroundColor: colors.surfaceContainer,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
    gap: 4,
    overflow: "hidden",
  },
  tileSelected: {
    borderColor: colors.primary,
    borderWidth: 2,
    backgroundColor: colors.surface,
  },
  playhead: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    height: 3,
    backgroundColor: colors.primary,
  },
  order: {
    fontFamily: fonts.labelBold,
    fontSize: 16,
    color: colors.onSurfaceMuted,
  },
  orderSelected: {
    color: colors.primary,
  },
  duration: {
    fontFamily: fonts.label,
    fontSize: 12,
    letterSpacing: 0.4,
    color: colors.onSurface,
  },
  durationSelected: {
    color: colors.primarySoft,
  },
  badge: {
    marginTop: 2,
    borderRadius: radius.full,
    paddingHorizontal: 6,
    paddingVertical: 2,
    backgroundColor: colors.background,
    maxWidth: 76,
  },
  badgeSelected: {
    borderWidth: 1,
    borderColor: colors.outlineStrong,
  },
  badgeText: {
    fontFamily: fonts.label,
    fontSize: 8,
    letterSpacing: 0.6,
    color: colors.onSurfaceMuted,
  },
  empty: {
    fontFamily: fonts.body,
    color: colors.onSurfaceMuted,
  },
});
