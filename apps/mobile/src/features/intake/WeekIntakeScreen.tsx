import { useCallback, useState } from "react";
import {
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useRouter } from "expo-router";
import * as ImagePicker from "expo-image-picker";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandMark } from "@/components/BrandMark";
import { BrassButton } from "@/components/BrassButton";
import { MVP_REFLECTION_QUESTION } from "@/config/reflection";
import { MVP_SONG_PICK } from "@/config/song-pick";
import { registerLocalAsset } from "@/lib/assets";
import { createMoment } from "@/lib/moments";
import { colors, radius, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";
import { AddAssetSheet, type AddAssetMode } from "./AddAssetSheet";
import { AssetFilmStrip } from "./AssetFilmStrip";
import { VoiceRecorderSheet } from "./VoiceRecorderSheet";
import type { IntakeAsset } from "./types";
import {
  canBeReflection,
  MAX_INTAKE_ASSETS,
  MIN_INTAKE_ASSETS,
  reflectionAsset,
} from "./types";

function newKey() {
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function mimeFromUri(uri: string, fallback: string) {
  const lower = uri.toLowerCase();
  if (lower.endsWith(".png")) return "image/png";
  if (lower.endsWith(".jpg") || lower.endsWith(".jpeg")) return "image/jpeg";
  if (lower.endsWith(".heic")) return "image/heic";
  if (lower.endsWith(".mp4")) return "video/mp4";
  if (lower.endsWith(".mov")) return "video/quicktime";
  if (lower.endsWith(".m4a")) return "audio/m4a";
  return fallback;
}

export function WeekIntakeScreen() {
  const router = useRouter();
  const [assets, setAssets] = useState<IntakeAsset[]>([]);
  const [addOpen, setAddOpen] = useState(false);
  const [addMode, setAddMode] = useState<AddAssetMode>("montage");
  const [voiceOpen, setVoiceOpen] = useState(false);
  const [composing, setComposing] = useState(false);

  const openAdd = useCallback((mode: AddAssetMode) => {
    setAddMode(mode);
    setAddOpen(true);
  }, []);

  const addAsset = useCallback(
    (asset: Omit<IntakeAsset, "key">) => {
      setAssets((prev) => {
        if (prev.length >= MAX_INTAKE_ASSETS) return prev;
        const next = asset.isReflection
          ? prev.map((a) => ({ ...a, isReflection: false }))
          : prev;
        return [...next, { ...asset, key: newKey() }];
      });
    },
    [],
  );

  const removeAsset = useCallback((key: string) => {
    setAssets((prev) => prev.filter((a) => a.key !== key));
  }, []);

  const setReflection = useCallback((key: string) => {
    setAssets((prev) => {
      const target = prev.find((a) => a.key === key);
      if (!target || !canBeReflection(target.type)) return prev;
      return prev.map((a) => ({
        ...a,
        isReflection: a.key === key,
      }));
    });
  }, []);

  async function pickLibrary() {
    setAddOpen(false);
    const forReflection = addMode === "reflection";
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      Alert.alert("Library", "Photo library permission is required.");
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: forReflection
        ? ImagePicker.MediaTypeOptions.Videos
        : ImagePicker.MediaTypeOptions.All,
      quality: 0.9,
      allowsMultipleSelection: !forReflection,
      selectionLimit: forReflection
        ? 1
        : MAX_INTAKE_ASSETS - assets.length,
    });
    if (result.canceled) return;
    for (const asset of result.assets) {
      const isVideo = asset.type === "video";
      if (forReflection && !isVideo) {
        Alert.alert(
          "Reflection",
          "Pick a talking-head video (or record a voice note).",
        );
        continue;
      }
      addAsset({
        localUri: asset.uri,
        type: isVideo ? "video" : "photo",
        mimeType:
          asset.mimeType ??
          mimeFromUri(asset.uri, isVideo ? "video/mp4" : "image/jpeg"),
        isReflection: forReflection,
      });
    }
  }

  async function recordVideo() {
    setAddOpen(false);
    const forReflection = addMode === "reflection";
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) {
      Alert.alert("Camera", "Camera permission is required.");
      return;
    }
    const result = await ImagePicker.launchCameraAsync({
      mediaTypes: ImagePicker.MediaTypeOptions.Videos,
      quality: 0.9,
    });
    if (result.canceled || !result.assets[0]) return;
    const asset = result.assets[0];
    addAsset({
      localUri: asset.uri,
      type: "video",
      mimeType: asset.mimeType ?? mimeFromUri(asset.uri, "video/mp4"),
      isReflection: forReflection,
    });
  }

  function openVoice() {
    setAddOpen(false);
    setVoiceOpen(true);
  }

  async function compose() {
    const reflection = reflectionAsset(assets);
    if (!reflection) {
      Alert.alert(
        "Reflection required",
        "Add exactly one Reflection (talking-head video or voice) before composing.",
      );
      return;
    }
    if (assets.length < MIN_INTAKE_ASSETS) return;
    setComposing(true);
    try {
      const registered: { assetId: string; isReflection: boolean }[] = [];
      for (const draft of assets) {
        let assetId = draft.assetId;
        if (!assetId) {
          // Requires gateway + media-service running with Supabase Storage
          assetId = await registerLocalAsset({
            localUri: draft.localUri,
            type: draft.type,
            mimeType: draft.mimeType,
          });
        }
        registered.push({
          assetId,
          isReflection: Boolean(draft.isReflection),
        });
      }

      const reflectionAssetId = registered.find((r) => r.isReflection)?.assetId;
      if (!reflectionAssetId) {
        throw new Error("Reflection Asset was not registered.");
      }

      const moment = await createMoment({
        assetIds: registered.map((r) => r.assetId),
        reflectionAssetId,
        reflectionQuestion: MVP_REFLECTION_QUESTION,
        songPickId: MVP_SONG_PICK.id,
      });

      router.replace({
        pathname: "/(processing)",
        params: { momentId: moment.id },
      });
    } catch (e) {
      Alert.alert(
        "Compose blocked",
        e instanceof Error
          ? e.message
          : "Could not create Moment. Asset upload may be missing from the gateway.",
      );
    } finally {
      setComposing(false);
    }
  }

  const hasReflection = Boolean(reflectionAsset(assets));
  const canCompose =
    assets.length >= MIN_INTAKE_ASSETS && hasReflection && !composing;

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
      <ScrollView contentContainerStyle={styles.content}>
        <BrandMark variant="hero" />
        <Text style={styles.invite}>This week, in a few frames.</Text>
        <Text style={styles.body}>
          Answer the Reflection, then add up to {MAX_INTAKE_ASSETS} Assets from
          your week. We&apos;ll draft a Cut and render your Moment.
        </Text>

        <View style={styles.questionCard}>
          <Text style={styles.questionLabel}>REFLECTION</Text>
          <Text style={styles.question}>{MVP_REFLECTION_QUESTION}</Text>
          <Text style={styles.questionHint}>
            {hasReflection
              ? "Reflection ready. Tap another video or voice in the strip to swap."
              : "Talking-head preferred; voice allowed. Required before compose."}
          </Text>
          <BrassButton
            label={hasReflection ? "Replace Reflection" : "Add Reflection"}
            onPress={() => openAdd("reflection")}
            style={styles.reflectionCta}
          />
        </View>

        <View style={styles.counterRow}>
          <Text style={styles.counter}>
            {assets.length}/{MAX_INTAKE_ASSETS}
          </Text>
          <Text style={styles.counterHint}>Assets</Text>
          {!hasReflection ? (
            <Text style={styles.missingHint}>· needs Reflection</Text>
          ) : null}
        </View>

        <AssetFilmStrip
          assets={assets}
          onAdd={() => openAdd("montage")}
          onRemove={removeAsset}
          onSetReflection={setReflection}
        />

        <BrassButton
          label="Compose Moment"
          onPress={compose}
          disabled={!canCompose}
          loading={composing}
          style={styles.cta}
        />
      </ScrollView>

      <AddAssetSheet
        visible={addOpen}
        mode={addMode}
        onClose={() => setAddOpen(false)}
        onPickLibrary={pickLibrary}
        onRecordVideo={recordVideo}
        onRecordVoice={openVoice}
      />
      <VoiceRecorderSheet
        visible={voiceOpen}
        onClose={() => setVoiceOpen(false)}
        onSaved={(uri, mimeType) =>
          addAsset({
            localUri: uri,
            type: "voice",
            mimeType,
            isReflection: true,
          })
        }
      />
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
  invite: { ...typography.title, marginTop: space.md },
  body: { ...typography.body },
  questionCard: {
    marginTop: space.sm,
    padding: space.md,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.outline,
    backgroundColor: colors.surface,
    gap: space.sm,
  },
  questionLabel: {
    ...typography.label,
    color: colors.primary,
  },
  question: {
    ...typography.bodyStrong,
  },
  questionHint: {
    ...typography.caption,
  },
  reflectionCta: {
    marginTop: space.xs,
  },
  counterRow: {
    flexDirection: "row",
    alignItems: "baseline",
    gap: space.sm,
    marginTop: space.sm,
  },
  counter: {
    fontFamily: typography.label.fontFamily,
    fontSize: 20,
    color: colors.primary,
    letterSpacing: 1,
  },
  counterHint: { ...typography.label },
  missingHint: {
    ...typography.caption,
    color: colors.error,
  },
  cta: { marginTop: space.lg },
});
