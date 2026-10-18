import { useEffect, useState } from "react";
import { Alert, Modal, Pressable, StyleSheet, Text, View } from "react-native";
import {
  AudioModule,
  RecordingPresets,
  setAudioModeAsync,
  useAudioRecorder,
  useAudioRecorderState,
} from "expo-audio";
import { BrassButton } from "@/components/BrassButton";
import { GhostButton } from "@/components/GhostButton";
import { colors, radius, space } from "@/theme/tokens";
import { fonts, typography } from "@/theme/typography";

type Props = {
  visible: boolean;
  onClose: () => void;
  onSaved: (uri: string, mimeType: string) => void;
};

export function VoiceRecorderSheet({ visible, onClose, onSaved }: Props) {
  const audioRecorder = useAudioRecorder(RecordingPresets.HIGH_QUALITY);
  const recorderState = useAudioRecorderState(audioRecorder);
  const [uri, setUri] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!visible) {
      void teardown();
    }
  }, [visible]);

  async function teardown() {
    try {
      if (recorderState.isRecording) {
        await audioRecorder.stop();
      }
    } catch {
      /* ignore */
    }
    setUri(null);
  }

  async function start() {
    try {
      setBusy(true);
      const perm = await AudioModule.requestRecordingPermissionsAsync();
      if (!perm.granted) {
        Alert.alert("Microphone", "Permission is required to record voice.");
        return;
      }
      await setAudioModeAsync({
        allowsRecording: true,
        playsInSilentMode: true,
      });
      await audioRecorder.prepareToRecordAsync();
      audioRecorder.record();
      setUri(null);
    } catch (e) {
      Alert.alert("Record failed", e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function stop() {
    try {
      setBusy(true);
      await audioRecorder.stop();
      setUri(audioRecorder.uri);
    } catch (e) {
      Alert.alert("Stop failed", e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function useRecording() {
    if (!uri) return;
    onSaved(uri, "audio/m4a");
    onClose();
  }

  const recording = recorderState.isRecording;

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={styles.backdrop} onPress={onClose}>
        <Pressable style={styles.sheet} onPress={(e) => e.stopPropagation()}>
          <Text style={styles.title}>Record voice</Text>
          <Text style={styles.sub}>
            Capture a short voice note. It becomes a VOICE Asset in your Week Intake.
          </Text>

          <View style={styles.status}>
            <Text style={styles.statusText}>
              {recording ? "Recording…" : uri ? "Ready to add" : "Idle"}
            </Text>
          </View>

          {!recording && !uri ? (
            <BrassButton label="Start recording" onPress={start} loading={busy} />
          ) : null}
          {recording ? (
            <BrassButton label="Stop" onPress={stop} loading={busy} />
          ) : null}
          {uri && !recording ? (
            <>
              <BrassButton label="Use recording" onPress={useRecording} />
              <GhostButton label="Re-record" onPress={start} />
            </>
          ) : null}

          <Pressable onPress={onClose} style={styles.cancel}>
            <Text style={styles.cancelText}>Cancel</Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: {
    flex: 1,
    justifyContent: "flex-end",
    backgroundColor: "rgba(0,0,0,0.55)",
  },
  sheet: {
    backgroundColor: colors.surface,
    borderTopLeftRadius: radius.md,
    borderTopRightRadius: radius.md,
    padding: space.lg,
    gap: space.sm,
    borderTopWidth: 1,
    borderColor: colors.outline,
  },
  title: { ...typography.title },
  sub: { ...typography.caption, marginBottom: space.sm },
  status: {
    padding: space.md,
    borderRadius: radius.sm,
    backgroundColor: colors.surfaceAlt,
    borderWidth: 1,
    borderColor: colors.outline,
    marginBottom: space.sm,
  },
  statusText: {
    fontFamily: fonts.label,
    color: colors.primary,
    letterSpacing: 1,
    textAlign: "center",
  },
  cancel: {
    alignItems: "center",
    paddingVertical: space.md,
    marginTop: space.sm,
  },
  cancelText: {
    fontFamily: fonts.label,
    color: colors.onSurfaceMuted,
    letterSpacing: 1,
  },
});
