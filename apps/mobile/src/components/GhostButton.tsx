import { Pressable, StyleSheet, Text, ViewStyle } from "react-native";
import { colors, radius, space } from "@/theme/tokens";
import { fonts } from "@/theme/typography";

type Props = {
  label: string;
  onPress: () => void;
  disabled?: boolean;
  style?: ViewStyle;
};

export function GhostButton({ label, onPress, disabled, style }: Props) {
  return (
    <Pressable
      accessibilityRole="button"
      onPress={onPress}
      disabled={disabled}
      style={({ pressed }) => [
        styles.base,
        disabled && styles.disabled,
        pressed && !disabled && styles.pressed,
        style,
      ]}
    >
      <Text style={styles.label}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    borderRadius: radius.full,
    borderWidth: 1,
    borderColor: colors.outline,
    paddingVertical: space.md,
    paddingHorizontal: space.lg,
    alignItems: "center",
    justifyContent: "center",
    minHeight: 52,
  },
  pressed: { backgroundColor: colors.surface },
  disabled: { opacity: 0.4 },
  label: {
    fontFamily: fonts.label,
    fontSize: 15,
    letterSpacing: 0.6,
    color: colors.onSurface,
  },
});
