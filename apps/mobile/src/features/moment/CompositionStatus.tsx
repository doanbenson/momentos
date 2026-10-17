import { StyleSheet, Text, View } from "react-native";
import type { MomentStatus } from "@/lib/types";
import { colors, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";
import { statusCopy } from "./useMomentPoll";

const STEPS: { key: MomentStatus; label: string }[] = [
  { key: "drafting", label: "Drafting Cut" },
  { key: "rendering", label: "Syncing · Rendering" },
  { key: "ready", label: "Ready" },
];

function stepActive(current: MomentStatus | undefined, step: MomentStatus) {
  const order: MomentStatus[] = ["intake", "drafting", "rendering", "ready"];
  if (!current || current === "failed") return false;
  return order.indexOf(current) >= order.indexOf(step);
}

type Props = { status?: MomentStatus };

export function CompositionStatus({ status }: Props) {
  const copy = statusCopy(status);
  return (
    <View style={styles.wrap}>
      <Text style={styles.title}>{copy.title}</Text>
      <Text style={styles.detail}>{copy.detail}</Text>
      <View style={styles.steps}>
        {STEPS.map((s) => {
          const on = stepActive(status, s.key) || status === s.key;
          return (
            <Text
              key={s.key}
              style={[styles.step, on && styles.stepOn, status === s.key && styles.stepCurrent]}
            >
              {s.label}
            </Text>
          );
        })}
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: space.sm },
  title: { ...typography.title },
  detail: { ...typography.body },
  steps: { gap: space.xs, marginTop: space.md },
  step: {
    fontFamily: fonts.label,
    fontSize: 12,
    letterSpacing: 1,
    color: colors.onSurfaceMuted,
    opacity: 0.45,
  },
  stepOn: { opacity: 0.75, color: colors.secondary },
  stepCurrent: { opacity: 1, color: colors.primary },
});
