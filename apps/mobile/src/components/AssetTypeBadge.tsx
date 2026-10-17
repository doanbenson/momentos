import { StyleSheet, Text, View } from "react-native";
import { colors, radius, space } from "@/theme/tokens";
import { fonts } from "@/theme/typography";

export type AssetType = "photo" | "video" | "voice";

const LABEL: Record<AssetType, string> = {
  photo: "PHOTO",
  video: "VIDEO",
  voice: "VOICE",
};

type Props = { type: AssetType };

export function AssetTypeBadge({ type }: Props) {
  return (
    <View style={styles.badge}>
      <Text style={styles.text}>{LABEL[type]}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    backgroundColor: "rgba(18, 16, 14, 0.72)",
    borderRadius: radius.sm,
    paddingHorizontal: space.sm,
    paddingVertical: 2,
  },
  text: {
    fontFamily: fonts.label,
    fontSize: 10,
    letterSpacing: 1,
    color: colors.primary,
  },
});
