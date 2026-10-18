import {
  Image,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { AssetTypeBadge } from "@/components/AssetTypeBadge";
import { colors, radius, space } from "@/theme/tokens";
import { fonts } from "@/theme/typography";
import type { IntakeAsset } from "./types";
import { canBeReflection, MAX_INTAKE_ASSETS } from "./types";

type Props = {
  assets: IntakeAsset[];
  onAdd: () => void;
  onRemove: (key: string) => void;
  /** Tap a video/voice tile to designate it as the Reflection (swap). */
  onSetReflection?: (key: string) => void;
};

export function AssetFilmStrip({
  assets,
  onAdd,
  onRemove,
  onSetReflection,
}: Props) {
  const canAdd = assets.length < MAX_INTAKE_ASSETS;

  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.row}
    >
      {assets.map((asset) => {
        const reflectable = canBeReflection(asset.type);
        return (
          <Pressable
            key={asset.key}
            disabled={!reflectable || !onSetReflection}
            onPress={() => onSetReflection?.(asset.key)}
            accessibilityLabel={
              asset.isReflection
                ? "Reflection Asset"
                : reflectable
                  ? "Set as Reflection"
                  : undefined
            }
            style={[styles.tile, asset.isReflection && styles.tileReflection]}
          >
            {asset.type === "voice" ? (
              <View style={styles.voiceTile}>
                <Text style={styles.voiceGlyph}>♪</Text>
              </View>
            ) : (
              <Image source={{ uri: asset.localUri }} style={styles.image} />
            )}
            <View style={styles.badge}>
              {asset.isReflection ? (
                <View style={styles.reflectionBadge}>
                  <Text style={styles.reflectionBadgeText}>REFLECTION</Text>
                </View>
              ) : (
                <AssetTypeBadge type={asset.type} />
              )}
            </View>
            <Pressable
              accessibilityLabel="Remove Asset"
              onPress={() => onRemove(asset.key)}
              style={styles.remove}
            >
              <Text style={styles.removeText}>×</Text>
            </Pressable>
          </Pressable>
        );
      })}
      {canAdd ? (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Add Asset"
          onPress={onAdd}
          style={styles.addTile}
        >
          <Text style={styles.addPlus}>+</Text>
          <Text style={styles.addLabel}>ADD</Text>
        </Pressable>
      ) : null}
    </ScrollView>
  );
}

const TILE = 96;

const styles = StyleSheet.create({
  row: {
    gap: space.sm,
    paddingVertical: space.sm,
  },
  tile: {
    width: TILE,
    height: TILE,
    borderRadius: radius.sm,
    overflow: "hidden",
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
  },
  tileReflection: {
    borderColor: colors.primary,
    borderWidth: 2,
  },
  image: { width: "100%", height: "100%" },
  voiceTile: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.surfaceAlt,
  },
  voiceGlyph: {
    fontFamily: fonts.headline,
    fontSize: 28,
    color: colors.primary,
  },
  badge: {
    position: "absolute",
    left: 4,
    bottom: 4,
  },
  reflectionBadge: {
    backgroundColor: "rgba(18, 16, 14, 0.85)",
    borderRadius: radius.sm,
    paddingHorizontal: space.sm,
    paddingVertical: 2,
    borderWidth: 1,
    borderColor: colors.primary,
  },
  reflectionBadgeText: {
    fontFamily: fonts.label,
    fontSize: 9,
    letterSpacing: 0.8,
    color: colors.primary,
  },
  remove: {
    position: "absolute",
    top: 2,
    right: 2,
    width: 22,
    height: 22,
    borderRadius: radius.sm,
    backgroundColor: "rgba(18,16,14,0.7)",
    alignItems: "center",
    justifyContent: "center",
  },
  removeText: {
    color: colors.onSurface,
    fontSize: 16,
    lineHeight: 18,
  },
  addTile: {
    width: TILE,
    height: TILE,
    borderRadius: radius.sm,
    borderWidth: 1,
    borderColor: colors.outline,
    borderStyle: "dashed",
    alignItems: "center",
    justifyContent: "center",
    gap: 4,
  },
  addPlus: {
    fontFamily: fonts.headline,
    fontSize: 28,
    color: colors.primary,
  },
  addLabel: {
    fontFamily: fonts.label,
    fontSize: 10,
    letterSpacing: 1,
    color: colors.onSurfaceMuted,
  },
});
