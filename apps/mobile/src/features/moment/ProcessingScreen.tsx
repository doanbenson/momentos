import { useEffect } from "react";
import { StyleSheet, Text, View } from "react-native";
import { useLocalSearchParams, useRouter } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandMark } from "@/components/BrandMark";
import { BrassButton } from "@/components/BrassButton";
import { FilmProgressRule } from "@/components/FilmProgressRule";
import { SongPickCaption } from "@/components/SongPickCaption";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";
import { CompositionStatus } from "./CompositionStatus";
import { statusProgress, useMomentPoll } from "./useMomentPoll";

export function ProcessingScreen() {
  const router = useRouter();
  const { momentId } = useLocalSearchParams<{ momentId?: string }>();
  const { moment, error, refresh } = useMomentPoll({ momentId });

  useEffect(() => {
    if (moment?.status === "ready" && momentId) {
      router.replace({
        pathname: "/(editor)",
        params: { momentId },
      });
    }
  }, [moment?.status, momentId, router]);

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
      <View style={styles.content}>
        <BrandMark variant="quiet" />
        <View style={styles.atmosphere}>
          <Text style={styles.atmosphereLabel}>PROCESSING</Text>
        </View>
        <CompositionStatus status={moment?.status} />
        <FilmProgressRule progress={statusProgress(moment?.status)} />
        <SongPickCaption />
        {error ? <Text style={styles.error}>{error}</Text> : null}
        {moment?.status === "failed" ? (
          <BrassButton label="Retry status" onPress={() => void refresh()} />
        ) : null}
        {!momentId ? (
          <Text style={styles.error}>Missing momentId — return to Week Intake.</Text>
        ) : null}
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  content: {
    flex: 1,
    padding: space.lg,
    gap: space.lg,
    justifyContent: "center",
  },
  atmosphere: {
    height: 160,
    borderRadius: 4,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
  },
  atmosphereLabel: {
    ...typography.label,
    color: colors.primary,
  },
  error: {
    ...typography.caption,
    color: colors.error,
  },
});
