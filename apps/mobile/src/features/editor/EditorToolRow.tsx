import { StyleSheet, View } from "react-native";
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
    <View style={styles.row}>
      <GhostButton
        label="←"
        onPress={onReorderLeft}
        disabled={disabled}
        style={styles.btn}
      />
      <GhostButton
        label="→"
        onPress={onReorderRight}
        disabled={disabled}
        style={styles.btn}
      />
      <GhostButton
        label="Replace"
        onPress={onReplace}
        disabled={disabled}
        style={styles.flex}
      />
      <GhostButton
        label="Remove"
        onPress={onRemove}
        disabled={disabled}
        style={styles.flex}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: "row",
    gap: space.sm,
    flexWrap: "wrap",
  },
  btn: { minWidth: 48, paddingHorizontal: space.md },
  flex: { flex: 1, minWidth: 100 },
});
