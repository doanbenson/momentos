import { Text, View, StyleSheet } from "react-native";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";

type Props = {
  variant?: "hero" | "quiet";
};

export function BrandMark({ variant = "hero" }: Props) {
  return (
    <View style={styles.wrap}>
      <Text style={variant === "hero" ? styles.hero : styles.quiet}>MOMENTOS</Text>
      {variant === "hero" ? (
        <View style={styles.rule} />
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { alignItems: "flex-start" },
  hero: {
    ...typography.brand,
    fontSize: 32,
  },
  quiet: {
    ...typography.label,
    color: colors.primary,
    letterSpacing: 2,
  },
  rule: {
    marginTop: space.sm,
    height: 2,
    width: 48,
    backgroundColor: colors.primary,
  },
});
