import { ScrollView, StyleSheet } from "react-native";
import { GhostButton } from "@/components/GhostButton";
import { space } from "@/theme/tokens";

type Props = {
  disabled?: boolean;
  onReorderLeft: () => void;
  onReorderRight: () => void;
  onReplace: () => void;
  onRemove: () => void;
};

export function EditorToolRow({
  disabled,
  onReorderLeft,
  onReorderRight,
  onReplace,
  onRemove,
}: Props) {
  return (
    <ScrollView
      horizontal
      showsHorizontalScrollIndicator={false}
      contentContainerStyle={styles.row}
    >
      <GhostButton
        label="Earlier"
        onPress={onReorderLeft}
        disabled={disabled}
        style={styles.chip}
      />
      <GhostButton
        label="Later"
        onPress={onReorderRight}
        disabled={disabled}
        style={styles.chip}
      />
      <GhostButton
        label="Replace"
        onPress={onReplace}
        disabled={disabled}
        style={styles.chip}
      />
      <GhostButton
        label="Remove"
        onPress={onRemove}
        disabled={disabled}
        style={styles.chip}
      />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  row: {
    gap: space.sm,
    paddingVertical: space.xs,
  },
  chip: {
    minHeight: 44,
    paddingVertical: space.sm,
    paddingHorizontal: space.md,
  },
});
