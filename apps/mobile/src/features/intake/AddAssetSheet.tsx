import { Modal, Pressable, StyleSheet, Text, View } from "react-native";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";

type Props = {
  visible: boolean;
  onClose: () => void;
  onPickLibrary: () => void;
  onRecordVideo: () => void;
  onRecordVoice: () => void;
};

export function AddAssetSheet({
  visible,
  onClose,
  onPickLibrary,
  onRecordVideo,
  onRecordVoice,
}: Props) {
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={styles.backdrop} onPress={onClose}>
        <Pressable style={styles.sheet} onPress={(e) => e.stopPropagation()}>
          <Text style={styles.title}>Add Asset</Text>
          <Text style={styles.sub}>Photos, videos, or a voice note from your week.</Text>

          <Action label="Library" hint="Photos & videos" onPress={onPickLibrary} />
          <Action label="Record video" hint="Camera" onPress={onRecordVideo} />
          <Action label="Record voice" hint="In-app mic" onPress={onRecordVoice} />

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
    borderTopLeftRadius: radius.md,
    borderTopRightRadius: radius.md,
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
    borderRadius: radius.sm,
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
