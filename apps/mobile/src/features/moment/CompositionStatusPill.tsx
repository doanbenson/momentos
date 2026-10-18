import { Pressable, StyleSheet, Text, View } from "react-native";
import { usePathname } from "expo-router";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";
import {
  isCompositionInProgress,
  isOnProcessingRoute,
} from "./compositionState";
import { useMomentJobs } from "./MomentJobsProvider";
import { statusCopy } from "./useMomentPoll";

/** Global Obsidian Sunset chip for in-progress composition (ADR 0018). */
export function CompositionStatusPill() {
  const insets = useSafeAreaInsets();
  const pathname = usePathname();
  const { activeMomentId, moment, readyToastVisible, openProcessing } =
    useMomentJobs();

  const visible = Boolean(
    activeMomentId &&
      moment &&
      !isOnProcessingRoute(pathname) &&
      !readyToastVisible &&
      (isCompositionInProgress(moment.status) || moment.status === "failed"),
  );

  if (!visible || !moment) return null;

  const copy = statusCopy(moment.status);
  const failed = moment.status === "failed";

  return (
    <View
      pointerEvents="box-none"
      style={[styles.host, { bottom: insets.bottom + space.md }]}
    >
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${copy.title}. Open Processing.`}
        onPress={openProcessing}
        style={({ pressed }) => [
          styles.chip,
          failed && styles.chipFailed,
          pressed && styles.pressed,
        ]}
      >
        <View style={[styles.dot, failed && styles.dotFailed]} />
        <View style={styles.copy}>
          <Text style={styles.title} numberOfLines={1}>
            {copy.title}
          </Text>
          <Text style={styles.detail} numberOfLines={1}>
            {failed ? "Tap to review" : "Composing · tap for status"}
          </Text>
        </View>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    left: space.lg,
    right: space.lg,
    zIndex: 40,
    alignItems: "center",
  },
  chip: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.sm,
    maxWidth: 420,
    width: "100%",
    paddingVertical: space.sm + 2,
    paddingHorizontal: space.md,
    borderRadius: radius.full,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outlineStrong,
  },
  chipFailed: {
    borderColor: colors.error,
  },
  pressed: { opacity: 0.9 },
  dot: {
    width: 8,
    height: 8,
    borderRadius: radius.full,
    backgroundColor: colors.primary,
  },
  dotFailed: { backgroundColor: colors.error },
  copy: { flex: 1, gap: 2 },
  title: {
    fontFamily: fonts.labelBold,
    fontSize: 12,
    letterSpacing: 1,
    textTransform: "uppercase",
    color: colors.primary,
  },
  detail: {
    ...typography.caption,
    color: colors.onSurfaceMuted,
  },
});
