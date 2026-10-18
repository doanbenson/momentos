import { useEffect, useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { useEventListener } from "expo";
import { useVideoPlayer, VideoView } from "expo-video";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";

type Props = {
  renderUrl?: string;
};

function formatMs(ms: number) {
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  const rem = s % 60;
  return `${m}:${rem.toString().padStart(2, "0")}`;
}

export function MomentPreviewPlayer({ renderUrl }: Props) {
  const [position, setPosition] = useState(0);
  const [duration, setDuration] = useState(0);

  const player = useVideoPlayer(renderUrl ?? null, (p) => {
    p.timeUpdateEventInterval = 0.25;
  });

  useEffect(() => {
    setPosition(0);
    setDuration(0);
  }, [renderUrl]);

  useEventListener(player, "timeUpdate", ({ currentTime }) => {
    setPosition(Math.round(currentTime * 1000));
    setDuration(Math.round((player.duration || 0) * 1000));
  });

  return (
    <View style={styles.wrap}>
      {renderUrl ? (
        <VideoView
          style={styles.video}
          player={player}
          nativeControls
          contentFit="cover"
        />
      ) : (
        <View style={styles.placeholder}>
          <Text style={styles.placeholderText}>9:16 preview</Text>
          <Text style={styles.placeholderHint}>
            Waiting for renderUrl from Moment…
          </Text>
        </View>
      )}
      <Text style={styles.timecode}>
        {formatMs(position)} / {formatMs(duration)}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: space.sm },
  video: {
    width: "100%",
    aspectRatio: 9 / 16,
    maxHeight: 420,
    alignSelf: "center",
    borderRadius: radius.sm,
    backgroundColor: colors.surface,
  },
  placeholder: {
    width: "100%",
    aspectRatio: 9 / 16,
    maxHeight: 420,
    alignSelf: "center",
    borderRadius: radius.sm,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.outline,
    alignItems: "center",
    justifyContent: "center",
    gap: space.sm,
  },
  placeholderText: {
    fontFamily: fonts.headline,
    fontSize: 18,
    color: colors.onSurface,
  },
  placeholderHint: { ...typography.caption, textAlign: "center", paddingHorizontal: space.lg },
  timecode: {
    fontFamily: fonts.label,
    fontSize: 12,
    letterSpacing: 1,
    color: colors.tertiary,
    textAlign: "center",
  },
});
