import { StyleSheet, Text, View } from "react-native";
import { MVP_SONG_PICK } from "@/config/song-pick";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";

type Props = { compact?: boolean };

export function SongPickCaption({ compact }: Props) {
  return (
    <View style={styles.wrap}>
      <Text style={styles.label}>SONG PICK</Text>
      <Text style={compact ? styles.titleCompact : styles.title}>
        {MVP_SONG_PICK.title}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: space.xs },
  label: { ...typography.label, color: colors.tertiary },
  title: { ...typography.bodyStrong, color: colors.onSurface },
  titleCompact: {
    ...typography.caption,
    color: colors.onSurfaceMuted,
  },
});
