import { StyleSheet, View } from "react-native";
import { colors, radius } from "@/theme/tokens";

type Props = {
  /** 0–1 */
  progress: number;
};

export function FilmProgressRule({ progress }: Props) {
  const clamped = Math.max(0, Math.min(1, progress));
  return (
    <View style={styles.track}>
      <View style={[styles.fill, { width: `${clamped * 100}%` }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  track: {
    height: 2,
    width: "100%",
    backgroundColor: colors.outline,
    borderRadius: radius.sm,
    overflow: "hidden",
  },
  fill: {
    height: "100%",
    backgroundColor: colors.primary,
  },
});
