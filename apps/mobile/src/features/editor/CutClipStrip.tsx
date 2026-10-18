import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import type { CutClip, CutClipRole } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { fonts } from "@/theme/typography";

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
        const reflection = clip.role === "reflection";
        return (
          <Pressable
            key={`${clip.assetId}-${clip.order}`}
            onPress={() => onSelect(clip.assetId)}
            style={[
              styles.tile,
              selected && styles.tileSelected,
              reflection && styles.tileReflection,
            ]}
          >
            <Text style={styles.order}>{index + 1}</Text>
            <Text style={styles.id} numberOfLines={1}>
              {clip.assetId.slice(0, 8)}
            </Text>
            <View
              style={[styles.badge, reflection && styles.badgeReflection]}
            >
              <Text
                style={[styles.badgeText, reflection && styles.badgeTextReflection]}
                numberOfLines={1}
              >
                {ROLE_LABEL[clip.role]}
              </Text>
            </View>
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
    minHeight: 88,
    alignItems: "center",
  },
  tile: {
    width: 80,
    height: 88,
    borderRadius: radius.sm,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
    gap: 2,
    paddingBottom: space.sm,
  },
  tileSelected: {
    borderColor: colors.primary,
    borderWidth: 2,
  },
  tileReflection: {
    borderColor: colors.primary,
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
  badge: {
    marginTop: 2,
    borderRadius: radius.sm,
    paddingHorizontal: 4,
    paddingVertical: 1,
    backgroundColor: "rgba(18, 16, 14, 0.72)",
  },
  badgeReflection: {
    borderWidth: 1,
    borderColor: colors.primary,
    backgroundColor: "rgba(18, 16, 14, 0.85)",
  },
  badgeText: {
    fontFamily: fonts.label,
    fontSize: 8,
    letterSpacing: 0.6,
    color: colors.secondary,
  },
  badgeTextReflection: {
    color: colors.primary,
  },
  empty: {
    fontFamily: fonts.body,
    color: colors.onSurfaceMuted,
  },
});
