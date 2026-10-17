import { useEffect, useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { useLocalSearchParams, useRouter } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { GhostButton } from "@/components/GhostButton";
import { SongPickCaption } from "@/components/SongPickCaption";
import { getMoment } from "@/lib/moments";
import type { Moment } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";
import { AudioMatchChecklist } from "./AudioMatchChecklist";
import { OsShareButton } from "./OsShareButton";
import { SaveToLibraryButton } from "./SaveToLibraryButton";

export function ShareScreen() {
  const router = useRouter();
  const { momentId } = useLocalSearchParams<{ momentId?: string }>();
  const [moment, setMoment] = useState<Moment | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!momentId) return;
    getMoment(momentId)
      .then(setMoment)
      .catch((e) => setError(e instanceof Error ? e.message : String(e)));
  }, [momentId]);

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
      <ScrollView contentContainerStyle={styles.content}>
        <Text style={styles.title}>Save or share</Text>
        <Text style={styles.body}>
          Keep your Moment, or share the Render and attach the Platform Song in
          Instagram Reels (Audio Match).
        </Text>

        <View style={styles.thumb}>
          <Text style={styles.thumbLabel}>
            {moment?.renderUrl ? "Render ready" : "Preview"}
          </Text>
          <Text style={styles.thumbHint} numberOfLines={2}>
            {moment?.renderUrl ?? "renderUrl pending from gateway"}
          </Text>
        </View>

        <SongPickCaption />
        {error ? <Text style={styles.error}>{error}</Text> : null}

        <OsShareButton renderUrl={moment?.renderUrl} />
        <SaveToLibraryButton renderUrl={moment?.renderUrl} />

        <AudioMatchChecklist />

        <GhostButton
          label="Done"
          onPress={() => router.replace("/(onboarding)")}
          style={styles.done}
        />
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  content: {
    padding: space.lg,
    gap: space.md,
    paddingBottom: space.xl,
  },
  title: { ...typography.title },
  body: { ...typography.body },
  thumb: {
    aspectRatio: 9 / 16,
    maxHeight: 280,
    alignSelf: "center",
    width: "55%",
    borderRadius: radius.sm,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
    padding: space.md,
    gap: space.sm,
  },
  thumbLabel: { ...typography.label, color: colors.primary },
  thumbHint: { ...typography.caption, textAlign: "center" },
  error: { ...typography.caption, color: colors.error },
  done: { marginTop: space.md },
});
