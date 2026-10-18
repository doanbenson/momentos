import { Pressable, StyleSheet, Text, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";
import { useMomentJobs } from "./MomentJobsProvider";

/** Film Grain ready notice when composition finishes off Processing (ADR 0018). */
export function ReadyToast() {
  const insets = useSafeAreaInsets();
  const { readyToastVisible, openReadyMoment, dismissReadyToast } =
    useMomentJobs();

  if (!readyToastVisible) return null;

  return (
    <View
      pointerEvents="box-none"
      style={[styles.host, { top: insets.top + space.md }]}
    >
      <View style={styles.toast}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Moment ready. Open Editor."
          onPress={openReadyMoment}
          style={styles.body}
        >
          <Text style={styles.title}>Moment ready</Text>
          <Text style={styles.detail}>Tap to open the Editor</Text>
        </Pressable>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Dismiss"
          onPress={dismissReadyToast}
          hitSlop={8}
          style={styles.dismiss}
        >
          <Text style={styles.dismissText}>×</Text>
        </Pressable>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  host: {
    position: "absolute",
    left: space.lg,
    right: space.lg,
    zIndex: 50,
    alignItems: "center",
  },
  toast: {
    flexDirection: "row",
    alignItems: "center",
    maxWidth: 420,
    width: "100%",
    borderRadius: radius.sm,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.primary,
    paddingLeft: space.md,
    paddingRight: space.sm,
    paddingVertical: space.sm,
  },
  body: { flex: 1, gap: 2, paddingVertical: space.xs },
  title: {
    fontFamily: fonts.labelBold,
    fontSize: 13,
    letterSpacing: 1,
    textTransform: "uppercase",
    color: colors.primary,
  },
  detail: {
    ...typography.caption,
    color: colors.onSurface,
  },
  dismiss: {
    width: 36,
    height: 36,
    alignItems: "center",
    justifyContent: "center",
    borderRadius: radius.sm,
  },
  dismissText: {
    fontFamily: fonts.headline,
    fontSize: 22,
    color: colors.onSurfaceMuted,
    lineHeight: 24,
  },
});
