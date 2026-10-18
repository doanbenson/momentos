import { Modal, Pressable, StyleSheet, Text, View } from "react-native";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";

export type AddAssetMode = "montage" | "reflection";

type Props = {
  visible: boolean;
  mode?: AddAssetMode;
  onClose: () => void;
  onPickLibrary: () => void;
  onRecordVideo: () => void;
  onRecordVoice: () => void;
};

export function AddAssetSheet({
  visible,
  mode = "montage",
  onClose,
  onPickLibrary,
  onRecordVideo,
  onRecordVoice,
}: Props) {
  const isReflection = mode === "reflection";

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={styles.backdrop} onPress={onClose}>
        <Pressable style={styles.sheet} onPress={(e) => e.stopPropagation()}>
          <Text style={styles.title}>
            {isReflection ? "Add Reflection" : "Add Asset"}
          </Text>
          <Text style={styles.sub}>
            {isReflection
              ? "Talking-head video preferred; voice note allowed."
              : "Photos or videos from your week for the montage."}
          </Text>

          <Action
            label="Library"
            hint={isReflection ? "Pick a video" : "Photos & videos"}
            onPress={onPickLibrary}
          />
          <Action label="Record video" hint="Camera" onPress={onRecordVideo} />
          {isReflection ? (
            <Action label="Record voice" hint="In-app mic" onPress={onRecordVoice} />
          ) : null}

          <Pressable onPress={onClose} style={styles.cancel}>
            <Text style={styles.cancelText}>Cancel</Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

function Action({
  label,
  hint,
  onPress,
}: {
  label: string;
  hint: string;
  onPress: () => void;
}) {
  return (
    <Pressable style={styles.action} onPress={onPress}>
      <Text style={styles.actionLabel}>{label}</Text>
      <Text style={styles.actionHint}>{hint}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    justifyContent: "flex-end",
    backgroundColor: "rgba(0,0,0,0.55)",
  },
  sheet: {
    backgroundColor: colors.surface,
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    padding: space.lg,
    gap: space.sm,
    borderTopWidth: 1,
    borderColor: colors.outline,
  },
  title: { ...typography.title },
  sub: { ...typography.caption, marginBottom: space.sm },
  action: {
    paddingVertical: space.md,
    paddingHorizontal: space.md,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.outline,
    backgroundColor: colors.surfaceAlt,
  },
  actionLabel: {
    fontFamily: fonts.bodyMedium,
    fontSize: 16,
    color: colors.onSurface,
  },
  actionHint: {
    ...typography.caption,
    marginTop: 2,
  },
  cancel: {
    alignItems: "center",
    paddingVertical: space.md,
    marginTop: space.sm,
  },
  cancelText: {
    fontFamily: fonts.label,
    color: colors.onSurfaceMuted,
    letterSpacing: 1,
  },
});
