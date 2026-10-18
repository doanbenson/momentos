import { useCallback, useState } from "react";
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useFocusEffect, useRouter } from "expo-router";
import { SafeAreaView, useSafeAreaInsets } from "react-native-safe-area-context";
import { GhostButton } from "@/components/GhostButton";
import { MVP_SONG_PICK } from "@/config/song-pick";
import { isCompositionInProgress } from "@/features/moment/compositionState";
import { useMomentJobs } from "@/features/moment/MomentJobsProvider";
import { listMoments } from "@/lib/moments";
import type { MomentListItem, MomentStatus } from "@/lib/types";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";

const STATUS_LABEL: Record<MomentStatus, string> = {
  intake: "Intake",
  drafting: "Drafting",
  rendering: "Rendering",
  ready: "Ready",
  failed: "Failed",
};

function friendlyListError(error: unknown): string {
  const message = error instanceof Error ? error.message : String(error);
  if (/network request failed|failed to fetch|network error/i.test(message)) {
    return "The gateway isn’t reachable. Start it, then try again.";
  }
  return message;
}

function formatCreatedAt(iso: string): string {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "";
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
  }).format(new Date(t));
}

function statusColor(status: MomentStatus): string {
  if (status === "failed") return colors.error;
  if (status === "ready") return colors.primary;
  if (status === "drafting" || status === "rendering") return colors.secondary;
  return colors.onSurfaceMuted;
}

function songLabel(songPickId: string): string {
  if (!songPickId) return "Song Pick";
  if (songPickId === MVP_SONG_PICK.id) return MVP_SONG_PICK.title;
  return songPickId;
}

export function HomeScreen() {
  const router = useRouter();
  const insets = useSafeAreaInsets();
  const { activeMomentId, moment, readyToastVisible } = useMomentJobs();
  const [moments, setMoments] = useState<MomentListItem[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const rows = await listMoments();
      setMoments(rows);
      setError(null);
    } catch (e) {
      setError(friendlyListError(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      void load();
    }, [load]),
  );

  function startWeekIntake() {
    router.push("/(onboarding)");
  }

  function openEditor(momentId: string) {
    router.push({ pathname: "/(editor)", params: { momentId } });
  }

  const rows = moments ?? [];
  const activeCount = rows.filter((item) =>
    isCompositionInProgress(item.status),
  ).length;
  const pillVisible = Boolean(
    activeMomentId &&
      moment &&
      !readyToastVisible &&
      (isCompositionInProgress(moment.status) || moment.status === "failed"),
  );
  const fabLift = pillVisible ? 72 : 0;

  return (
    <SafeAreaView style={styles.safe} edges={["top"]}>
      <View style={styles.header}>
        <View style={styles.brand}>
          <View style={styles.brandDot} />
          <Text style={styles.brandName}>Moments</Text>
        </View>
      </View>

      <ScrollView
        contentContainerStyle={[
          styles.content,
          { paddingBottom: insets.bottom + 96 + fabLift },
        ]}
      >
        <View style={styles.sectionHead}>
          <View style={styles.sectionTitleRow}>
            <Text style={styles.sectionTitle}>My Moments</Text>
            <View style={styles.countPill}>
              <Text style={styles.countText}>
                {activeCount} Active
              </Text>
            </View>
          </View>
        </View>

        {loading && moments == null ? (
          <ActivityIndicator color={colors.primary} style={styles.spinner} />
        ) : null}

        {error ? (
          <View style={styles.errorCard}>
            <Text style={styles.errorTitle}>Couldn’t load Moments</Text>
            <Text style={styles.errorBody}>{error}</Text>
            <GhostButton label="Try again" onPress={() => void load()} />
          </View>
        ) : null}

        <ScrollView
          horizontal
          showsHorizontalScrollIndicator={false}
          contentContainerStyle={styles.carousel}
        >
          <Pressable
            accessibilityRole="button"
            accessibilityLabel="Start Week Intake"
            onPress={startWeekIntake}
            style={({ pressed }) => [styles.startCard, pressed && styles.pressed]}
          >
            <View style={styles.plusCircle}>
              <Text style={styles.plusGlyph}>+</Text>
            </View>
            <Text style={styles.startTitle}>Start a Moment</Text>
            <Text style={styles.startBody}>
              Week Intake — photos, video, and a Reflection.
            </Text>
            <Text style={styles.startKicker}>NEW</Text>
          </Pressable>

          {rows.map((item) => {
            const when = formatCreatedAt(item.createdAt);
            const status = STATUS_LABEL[item.status];
            return (
              <Pressable
                key={item.id}
                accessibilityRole="button"
                accessibilityLabel={`Moment, ${status}${when ? `, ${when}` : ""}. Open in Editor.`}
                onPress={() => openEditor(item.id)}
                style={({ pressed }) => [styles.card, pressed && styles.pressed]}
              >
                <View style={styles.poster}>
                  <Text style={[styles.posterStatus, { color: statusColor(item.status) }]}>
                    {status}
                  </Text>
                  {item.renderUrl ? (
                    <Text style={styles.posterHint}>Render ready</Text>
                  ) : (
                    <Text style={styles.posterHint}>Open Editor</Text>
                  )}
                </View>
                <Text style={styles.cardTitle} numberOfLines={1}>
                  {when || "Moment"}
                </Text>
                <Text style={styles.cardSong} numberOfLines={1}>
                  {songLabel(item.songPickId)}
                </Text>
              </Pressable>
            );
          })}
        </ScrollView>

        {!loading && !error && rows.length === 0 ? (
          <Text style={styles.empty}>No Moments yet. Start with Week Intake.</Text>
        ) : null}
      </ScrollView>

      <View
        pointerEvents="box-none"
        style={[styles.fabHost, { bottom: insets.bottom + space.md + fabLift }]}
      >
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Start Week Intake"
          onPress={startWeekIntake}
          style={({ pressed }) => [styles.fab, pressed && styles.pressed]}
        >
          <Text style={styles.fabGlyph}>+</Text>
        </Pressable>
      </View>
    </SafeAreaView>
  );
}

const CARD_W = 168;

const styles = StyleSheet.create({
  safe: { flex: 1, backgroundColor: colors.background },
  header: {
    minHeight: 56,
    paddingHorizontal: space.md,
    borderBottomWidth: 1,
    borderBottomColor: colors.outline,
    flexDirection: "row",
    alignItems: "center",
  },
  brand: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.sm,
    minHeight: 44,
  },
  brandDot: {
    width: 10,
    height: 10,
    borderRadius: radius.full,
    backgroundColor: colors.primary,
  },
  brandName: {
    fontFamily: fonts.headline,
    fontSize: 20,
    letterSpacing: -0.3,
    color: colors.onSurface,
  },
  content: {
    paddingTop: space.md,
    gap: space.md,
  },
  sectionHead: {
    paddingHorizontal: space.md,
  },
  sectionTitleRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: space.sm,
  },
  sectionTitle: {
    fontFamily: fonts.headlineRegular,
    fontSize: 16,
    color: colors.onSurface,
  },
  countPill: {
    borderRadius: radius.full,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    paddingHorizontal: space.sm,
    paddingVertical: 2,
  },
  countText: {
    fontFamily: fonts.label,
    fontSize: 11,
    letterSpacing: 0.4,
    color: colors.onSurfaceMuted,
  },
  spinner: { marginVertical: space.lg },
  errorCard: {
    marginHorizontal: space.md,
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.outline,
    padding: space.md,
    gap: space.sm,
  },
  errorTitle: {
    fontFamily: fonts.headlineRegular,
    fontSize: 16,
    color: colors.onSurface,
  },
  errorBody: {
    ...typography.body,
    color: colors.error,
  },
  carousel: {
    paddingHorizontal: space.md,
    gap: space.md,
    alignItems: "stretch",
  },
  startCard: {
    width: CARD_W,
    minHeight: 220,
    borderRadius: 20,
    borderWidth: 2,
    borderStyle: "dashed",
    borderColor: colors.outlineStrong,
    backgroundColor: colors.surface,
    padding: space.md,
    alignItems: "center",
    justifyContent: "center",
  },
  plusCircle: {
    width: 48,
    height: 48,
    borderRadius: radius.full,
    backgroundColor: colors.surfaceAlt,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: space.sm,
  },
  plusGlyph: {
    fontFamily: fonts.headline,
    fontSize: 28,
    lineHeight: 32,
    color: colors.primary,
  },
  startTitle: {
    fontFamily: fonts.headlineRegular,
    fontSize: 14,
    color: colors.onSurface,
    textAlign: "center",
  },
  startBody: {
    ...typography.caption,
    textAlign: "center",
    marginTop: space.xs,
  },
  startKicker: {
    marginTop: space.md,
    fontFamily: fonts.labelBold,
    fontSize: 10,
    letterSpacing: 1,
    color: colors.primary,
  },
  card: {
    width: CARD_W,
    minHeight: 220,
    borderRadius: 20,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    padding: space.sm,
    gap: space.sm,
  },
  poster: {
    flex: 1,
    minHeight: 120,
    borderRadius: radius.md,
    backgroundColor: colors.surfaceContainer,
    alignItems: "center",
    justifyContent: "center",
    gap: space.xs,
    padding: space.sm,
  },
  posterStatus: {
    fontFamily: fonts.labelBold,
    fontSize: 12,
    letterSpacing: 0.8,
    textTransform: "uppercase",
    color: colors.primary,
  },
  posterHint: {
    ...typography.caption,
    textAlign: "center",
  },
  cardTitle: {
    fontFamily: fonts.headlineRegular,
    fontSize: 14,
    color: colors.onSurface,
    paddingHorizontal: space.xs,
  },
  cardSong: {
    ...typography.caption,
    paddingHorizontal: space.xs,
    marginBottom: space.xs,
  },
  empty: {
    ...typography.caption,
    paddingHorizontal: space.md,
  },
  pressed: { opacity: 0.9, transform: [{ scale: 0.98 }] },
  fabHost: {
    position: "absolute",
    left: 0,
    right: 0,
    alignItems: "center",
    zIndex: 30,
  },
  fab: {
    width: 56,
    height: 56,
    borderRadius: radius.full,
    backgroundColor: colors.primary,
    alignItems: "center",
    justifyContent: "center",
    zIndex: 30,
  },
  fabGlyph: {
    fontFamily: fonts.headline,
    fontSize: 30,
    lineHeight: 34,
    color: colors.onPrimary,
  },
});
