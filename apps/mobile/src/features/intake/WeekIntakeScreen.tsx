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
import { MVP_SONG_PICK } from "@/config/song-pick";
import { registerLocalAsset } from "@/lib/assets";
import { createMoment } from "@/lib/moments";
import { colors, space } from "@/theme/tokens";
import { typography } from "@/theme/typography";
import { AddAssetSheet } from "./AddAssetSheet";
import { AssetFilmStrip } from "./AssetFilmStrip";
import { VoiceRecorderSheet } from "./VoiceRecorderSheet";
import type { IntakeAsset } from "./types";
import { MAX_INTAKE_ASSETS, MIN_INTAKE_ASSETS } from "./types";

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
  const [voiceOpen, setVoiceOpen] = useState(false);
  const [composing, setComposing] = useState(false);

  const addAsset = useCallback((asset: Omit<IntakeAsset, "key">) => {
    setAssets((prev) => {
      if (prev.length >= MAX_INTAKE_ASSETS) return prev;
      return [...prev, { ...asset, key: newKey() }];
    });
  }, []);

  const removeAsset = useCallback((key: string) => {
    setAssets((prev) => prev.filter((a) => a.key !== key));
  }, []);

  async function pickLibrary() {
    setAddOpen(false);
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) {
      Alert.alert("Library", "Photo library permission is required.");
      return;
    }
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ImagePicker.MediaTypeOptions.All,
      quality: 0.9,
      allowsMultipleSelection: true,
      selectionLimit: MAX_INTAKE_ASSETS - assets.length,
    });
    if (result.canceled) return;
    for (const asset of result.assets) {
      const isVideo = asset.type === "video";
      addAsset({
        localUri: asset.uri,
        type: isVideo ? "video" : "photo",
        mimeType:
          asset.mimeType ??
          mimeFromUri(asset.uri, isVideo ? "video/mp4" : "image/jpeg"),
      });
    }
  }

  async function recordVideo() {
    setAddOpen(false);
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
    });
  }

  function openVoice() {
    setAddOpen(false);
    setVoiceOpen(true);
  }

  async function compose() {
    if (assets.length < MIN_INTAKE_ASSETS) return;
    setComposing(true);
    try {
      const assetIds: string[] = [];
      for (const draft of assets) {
        if (draft.assetId) {
          assetIds.push(draft.assetId);
          continue;
        }
        // Requires gateway + media-service running with Supabase Storage
        const id = await registerLocalAsset({
          localUri: draft.localUri,
          type: draft.type,
          mimeType: draft.mimeType,
        });
        assetIds.push(id);
      }

      const moment = await createMoment({
        assetIds,
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

  const canCompose = assets.length >= MIN_INTAKE_ASSETS && !composing;

  return (
    <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
      <ScrollView contentContainerStyle={styles.content}>
        <BrandMark variant="hero" />
        <Text style={styles.invite}>This week, in a few frames.</Text>
        <Text style={styles.body}>
          Add up to {MAX_INTAKE_ASSETS} photos, videos, or voice notes. We&apos;ll
          draft a Cut and render your Moment.
        </Text>

        <View style={styles.counterRow}>
          <Text style={styles.counter}>
            {assets.length}/{MAX_INTAKE_ASSETS}
          </Text>
          <Text style={styles.counterHint}>Assets</Text>
        </View>

        <AssetFilmStrip
          assets={assets}
          onAdd={() => setAddOpen(true)}
          onRemove={removeAsset}
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
        onClose={() => setAddOpen(false)}
        onPickLibrary={pickLibrary}
        onRecordVideo={recordVideo}
        onRecordVoice={openVoice}
      />
      <VoiceRecorderSheet
        visible={voiceOpen}
        onClose={() => setVoiceOpen(false)}
        onSaved={(uri, mimeType) =>
          addAsset({ localUri: uri, type: "voice", mimeType })
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
  cta: { marginTop: space.lg },
});
