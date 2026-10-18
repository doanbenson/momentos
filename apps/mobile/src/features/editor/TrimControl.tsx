import { useEffect, useState } from "react";
import { Pressable, StyleSheet, Text, TextInput, View } from "react-native";
import type { CutClip } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";
import { clipDurationMs, formatDurationMs } from "./clipWindow";

type Props = {
  clip: CutClip;
  onChange: (next: { sourceInMs: number; sourceOutMs: number }) => void;
};

const STEP_MS = 100;

function digitsOnly(text: string) {
  return text.replace(/[^\d]/g, "");
}

function parseMs(text: string): number | null {
  if (!/^\d+$/.test(text)) return null;
  const n = Number(text);
  return Number.isInteger(n) ? n : null;
}

export function TrimControl({ clip, onChange }: Props) {
  const [inText, setInText] = useState(
    clip.sourceInMs != null ? String(clip.sourceInMs) : "",
  );
  const [outText, setOutText] = useState(
    clip.sourceOutMs != null ? String(clip.sourceOutMs) : "",
  );
  const [reject, setReject] = useState<string | null>(null);

  useEffect(() => {
    setInText(clip.sourceInMs != null ? String(clip.sourceInMs) : "");
    setOutText(clip.sourceOutMs != null ? String(clip.sourceOutMs) : "");
    setReject(null);
  }, [clip.assetId, clip.sourceInMs, clip.sourceOutMs]);

  function commit(nextIn: string, nextOut: string) {
    const inDigits = digitsOnly(nextIn);
    const outDigits = digitsOnly(nextOut);
    setInText(inDigits);
    setOutText(outDigits);
    if (inDigits === "" || outDigits === "") {
      setReject(null);
      return;
    }
    const sourceInMs = parseMs(inDigits);
    const sourceOutMs = parseMs(outDigits);
    if (sourceInMs == null || sourceOutMs == null) {
      setReject("Use whole milliseconds.");
      return;
    }
    if (sourceOutMs <= sourceInMs) {
      setReject("Out must be after in.");
      return;
    }
    setReject(null);
    if (sourceInMs === clip.sourceInMs && sourceOutMs === clip.sourceOutMs) return;
    onChange({ sourceInMs, sourceOutMs });
  }

  function nudge(field: "in" | "out", delta: number) {
    const parsedIn = parseMs(inText);
    const parsedOut = parseMs(outText);
    const currentIn = parsedIn ?? clip.sourceInMs ?? 0;
    const timeline = clipDurationMs(clip);
    const fallbackOut =
      clip.sourceOutMs ??
      (timeline != null ? (clip.sourceInMs ?? 0) + timeline : currentIn + 1000);
    const currentOut = parsedOut ?? fallbackOut;
    const nextIn = Math.max(0, field === "in" ? currentIn + delta : currentIn);
    const nextOut = Math.max(0, field === "out" ? currentOut + delta : currentOut);
    commit(String(nextIn), String(nextOut));
  }

  const duration = clipDurationMs(clip);

  return (
    <View style={styles.card}>
      <Text style={styles.kicker}>TRIM</Text>
      <Text style={styles.title}>Source in and out</Text>
      <Field
        label="In"
        accessibilityLabel="Source in milliseconds"
        value={inText}
        placeholder="0"
        onChangeText={(text) => commit(text, outText)}
        onMinus={() => nudge("in", -STEP_MS)}
        onPlus={() => nudge("in", STEP_MS)}
      />
      <Field
        label="Out"
        accessibilityLabel="Source out milliseconds"
        value={outText}
        placeholder="ms"
        onChangeText={(text) => commit(inText, text)}
        onMinus={() => nudge("out", -STEP_MS)}
        onPlus={() => nudge("out", STEP_MS)}
      />
      <Text style={styles.duration}>
        {duration != null
          ? `Duration ${formatDurationMs(duration)}`
          : "Duration uses the source window, or the Cut span until you trim."}
      </Text>
      {reject ? (
        <Text accessibilityLiveRegion="polite" style={styles.reject}>
          {reject}
        </Text>
      ) : null}
    </View>
  );
}

function Field({
  label,
  accessibilityLabel,
  value,
  placeholder,
  onChangeText,
  onMinus,
  onPlus,
}: {
  label: string;
  accessibilityLabel: string;
  value: string;
  placeholder: string;
  onChangeText: (text: string) => void;
  onMinus: () => void;
  onPlus: () => void;
}) {
  return (
    <View style={styles.field}>
      <Text style={styles.fieldLabel}>{label}</Text>
      <Stepper label={`Decrease ${label}`} glyph="−" onPress={onMinus} />
      <TextInput
        accessibilityLabel={accessibilityLabel}
        keyboardType="number-pad"
        value={value}
        placeholder={placeholder}
        placeholderTextColor={colors.onSurfaceMuted}
        onChangeText={onChangeText}
        style={styles.input}
      />
      <Stepper label={`Increase ${label}`} glyph="+" onPress={onPlus} />
    </View>
  );
}

function Stepper({
  label,
  glyph,
  onPress,
}: {
  label: string;
  glyph: string;
  onPress: () => void;
}) {
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [styles.step, pressed && styles.stepPressed]}
    >
      <Text style={styles.stepGlyph}>{glyph}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.outline,
    padding: space.md,
    gap: space.sm,
  },
  kicker: {
    ...typography.label,
    color: colors.primary,
  },
  title: {
    fontFamily: fonts.headlineRegular,
    fontSize: 16,
    color: colors.onSurface,
  },
  field: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.sm,
  },
  fieldLabel: {
    width: 36,
    fontFamily: fonts.labelBold,
    fontSize: 12,
    letterSpacing: 0.6,
    color: colors.onSurfaceMuted,
  },
  input: {
    flex: 1,
    minHeight: 44,
    borderRadius: radius.full,
    backgroundColor: colors.surfaceContainer,
    borderWidth: 1,
    borderColor: colors.outlineStrong,
    paddingHorizontal: space.md,
    color: colors.onSurface,
    fontFamily: fonts.label,
    fontSize: 16,
    textAlign: "center",
  },
  step: {
    width: 44,
    height: 44,
    borderRadius: radius.full,
    backgroundColor: colors.surfaceAlt,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
  },
  stepPressed: { opacity: 0.85 },
  stepGlyph: {
    fontFamily: fonts.headlineRegular,
    fontSize: 20,
    color: colors.onSurface,
    lineHeight: 24,
  },
  duration: {
    ...typography.caption,
  },
  reject: {
    ...typography.caption,
    color: colors.error,
  },
});
