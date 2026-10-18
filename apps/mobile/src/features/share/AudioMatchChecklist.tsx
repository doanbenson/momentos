import { StyleSheet, Text, View } from "react-native";
import { MVP_SONG_PICK } from "@/config/song-pick";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";

/** GRILL Q5 — locked Share checklist copy (Platform Song vs Edit Audio). */
export function AudioMatchChecklist() {
  const songTitle = MVP_SONG_PICK.title;

  const steps = [
    "Open Instagram and create a Reel",
    "Import the video you just shared",
    `Add audio — search ${songTitle} (this is the Platform Song)`,
    "Post — Instagram mutes our temporary Edit Audio and uses the cleared track with artist credit",
  ];

  return (
    <View style={styles.card}>
      <Text style={styles.heading}>Audio Match</Text>
      <Text style={styles.intro}>
        Attach the Platform Song when you post so Instagram can credit the artist.
      </Text>
      {steps.map((step, i) => (
        <View key={i} style={styles.stepRow}>
          <Text style={styles.num}>{i + 1}</Text>
          <Text style={styles.step}>{step}</Text>
        </View>
      ))}
      <Text style={styles.footnote}>Edit Audio is temporary for cutting only.</Text>
      {!MVP_SONG_PICK.platformSong.musicId ? (
        <Text style={styles.todo}>
          {/* musicId not set yet — deep-link accuracy blocked */}
          TODO: Platform Song musicId for checklist deep-link
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.outline,
    padding: space.lg,
    gap: space.md,
  },
  heading: { ...typography.title, fontSize: 18 },
  intro: { ...typography.caption },
  stepRow: {
    flexDirection: "row",
    gap: space.md,
    alignItems: "flex-start",
  },
  num: {
    fontFamily: fonts.labelBold,
    fontSize: 12,
    color: colors.onPrimary,
    width: 22,
    height: 22,
    lineHeight: 22,
    textAlign: "center",
    borderRadius: radius.full,
    backgroundColor: colors.primary,
    overflow: "hidden",
  },
  step: {
    ...typography.body,
    flex: 1,
    color: colors.onSurface,
  },
  footnote: {
    ...typography.caption,
    fontStyle: "italic",
    marginTop: space.sm,
  },
  todo: {
    ...typography.caption,
    color: colors.tertiary,
  },
});
